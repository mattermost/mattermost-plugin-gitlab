// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	gitlabLib "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
	"github.com/mattermost/mattermost-plugin-gitlab/server/webhook"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const (
	webhookTimeout            = 10 * time.Second
	eventSourceParentPipeline = "parent_pipeline"

	// notificationDedupTTL is the window during which a duplicate DM for the
	// same recipient and message is suppressed (e.g. duplicate webhook
	// deliveries from overlapping group/project hooks or GitLab retries).
	notificationDedupTTL    = 30 * time.Second
	notificationDedupKeyFmt = "notif_dedup_%s"
)

type gitlabRetreiver struct {
	p *Plugin
}

func (g *gitlabRetreiver) GetPipelineURL(pathWithNamespace string, pipelineID int) string {
	config := g.p.getConfiguration()
	return fmt.Sprintf("%s/%s/-/pipelines/%d", config.GitlabURL, pathWithNamespace, pipelineID)
}

func (g *gitlabRetreiver) GetJobURL(pathWithNamespace string, jobID int) string {
	config := g.p.getConfiguration()
	return fmt.Sprintf("%s/%s/-/jobs/%d", config.GitlabURL, pathWithNamespace, jobID)
}

func (g *gitlabRetreiver) GetUserURL(username string) string {
	config := g.p.getConfiguration()
	return fmt.Sprintf("%s/%s", config.GitlabURL, username)
}

func (g *gitlabRetreiver) GetUsernameByID(id int) string {
	return g.p.getGitlabIDToUsernameMapping(fmt.Sprintf("%d", id))
}

func (g *gitlabRetreiver) ParseGitlabUsernamesFromText(text string) []string {
	return parseGitlabUsernamesFromText(text)
}

func (g *gitlabRetreiver) GetSubscribedChannelsForProject(
	ctx context.Context,
	namespace string,
	project string,
	isPublicVisibility bool,
) []*subscription.Subscription {
	return g.p.GetSubscribedChannelsForProject(ctx, namespace, project, isPublicVisibility)
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	signature := r.Header.Get("X-Gitlab-Token")
	if config.WebhookSecret != signature {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	eventType := gitlabLib.WebhookEventType(r)
	event, err := gitlabLib.ParseWebhook(eventType, body)
	if err != nil {
		p.client.Log.Debug("Can't parse webhook", "err", err.Error(), "header", r.Header.Get("X-Gitlab-Event"), "event", string(body))
		http.Error(w, "Unable to handle request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeout)
	defer cancel()

	var repoPrivate bool
	var pathWithNamespace string
	var handlers []*webhook.HandleWebhook
	var errHandler error
	var warnings []string
	fromUser := ""

	switch event := event.(type) {
	case *gitlabLib.MergeEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, warnings, errHandler = p.WebhookHandler.HandleMergeRequest(ctx, event)
	case *gitlabLib.IssueEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, warnings, errHandler = p.WebhookHandler.HandleIssue(ctx, event, eventType)
	case *gitlabLib.IssueCommentEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, warnings, errHandler = p.WebhookHandler.HandleIssueComment(ctx, event)
	case *gitlabLib.MergeCommentEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, warnings, errHandler = p.WebhookHandler.HandleMergeRequestComment(ctx, event)
	case *gitlabLib.PushEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.UserName
		handlers, errHandler = p.WebhookHandler.HandlePush(ctx, event)
	case *gitlabLib.PipelineEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		if !p.configuration.EnableChildPipelineNotifications && event.ObjectAttributes.Source == eventSourceParentPipeline {
			return
		}

		handlers, errHandler = p.WebhookHandler.HandlePipeline(ctx, event)
	case *gitlabLib.JobEvent:
		repoPrivate = event.Repository.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.ProjectName
		fromUser = event.User.Name
		handlers, errHandler = p.WebhookHandler.HandleJobs(ctx, event)
	case *gitlabLib.TagEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.UserName
		handlers, errHandler = p.WebhookHandler.HandleTag(ctx, event)
	case *gitlabLib.ReleaseEvent:
		repoPrivate = event.Project.VisibilityLevel == webhook.PrivateVisibilityLevel
		pathWithNamespace = event.Project.PathWithNamespace
		handlers, errHandler = p.WebhookHandler.HandleRelease(ctx, event)
	case *gitlabLib.DeploymentEvent:
		repoPrivate = event.Project.VisibilityLevel == webhook.PrivateVisibilityLevel
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandleDeployment(ctx, event)
	default:
		p.client.Log.Debug("Event type not implemented", "type", string(gitlabLib.WebhookEventType(r)))
		return
	}

	if repoPrivate && !config.EnablePrivateRepo {
		return
	}

	if err = p.isNamespaceAllowed(pathWithNamespace); err != nil {
		p.client.Log.Info("Webhook event skipped: project is not in the allowed GitLab group", "path_with_namespace", pathWithNamespace, "reason", err.Error())
		return
	}

	if errHandler != nil {
		p.client.Log.Debug("Error when handling webhook event", "err", errHandler)
		return
	}

	if warnings != nil {
		p.logWarnings(warnings)
	}

	alreadySentRefresh := make(map[string]bool)
	p.sendRefreshIfNotAlreadySent(alreadySentRefresh, fromUser)
	for _, res := range handlers {
		p.client.Log.Info("new msg", "message", res.Message, "from", res.From)
		for _, to := range res.ToUsers {
			userTo := p.sendRefreshIfNotAlreadySent(alreadySentRefresh, to)
			if len(userTo) > 0 && len(res.Message) > 0 {
				info, err := p.getGitlabUserInfoByMattermostID(userTo)
				if err != nil {
					p.client.Log.Warn("can't get user info to know if user wants to receive notifications", "err", err.Message)
					continue
				}
				if info.Settings.Notifications {
					p.sendDMNotification(userTo, res.Message)
				}
			}
		}
		for _, to := range res.ToChannels {
			if len(res.Message) > 0 {
				post := &model.Post{
					UserId:    p.BotUserID,
					Message:   res.Message,
					ChannelId: to,
				}
				if err := p.client.Post.CreatePost(post); err != nil {
					p.client.Log.Warn("can't create post for webhook event", "err", err.Error())
				}
			}
		}
		p.sendRefreshIfNotAlreadySent(alreadySentRefresh, res.From)
	}
}

// notificationDedupKey returns a KV key that uniquely identifies a DM
// notification by its recipient and rendered message, so that repeated
// deliveries of the same notification collapse onto the same key.
func notificationDedupKey(recipientID, message string) string {
	hash := sha256.Sum256([]byte(recipientID + "_" + message))
	return fmt.Sprintf(notificationDedupKeyFmt, hex.EncodeToString(hash[:]))
}

// sendDMNotification sends a bot DM to userID, deduplicating against
// duplicate webhook deliveries (e.g. overlapping group/project hooks, GitLab
// retries, or concurrent delivery across cluster nodes). It atomically claims
// a short-lived KV key before posting, so only the first delivery to claim
// the key actually sends the DM.
func (p *Plugin) sendDMNotification(userID, message string) {
	dedupKey := notificationDedupKey(userID, message)
	claimed, kvErr := p.client.KV.Set(dedupKey, true,
		pluginapi.SetExpiry(notificationDedupTTL),
		pluginapi.SetAtomic(nil),
	)
	if kvErr != nil {
		// Fail open: send the notification rather than dropping it.
		p.client.Log.Warn("failed to claim notification dedup key, sending DM anyway", "err", kvErr.Error())
	} else if !claimed {
		// Another delivery already claimed this notification.
		p.client.Log.Debug("notification already claimed, skipping", "dedup_key", dedupKey)
		return
	}

	if err := p.CreateBotDMPost(userID, message, "custom_git_review_request"); err != nil {
		// Keep the claim: CreateBotDMPost may have persisted the post despite
		// returning an error, so releasing it would let a retry post a
		// duplicate. Let the claim TTL expire on its own.
		p.client.Log.Warn("can't send dm post", "err", err.Error())
	}
}

func (p *Plugin) sendRefreshIfNotAlreadySent(alreadySentRefresh map[string]bool, gitlabUsername string) string {
	if len(gitlabUsername) == 0 || alreadySentRefresh[gitlabUsername] {
		return ""
	}
	alreadySentRefresh[gitlabUsername] = true
	userMattermostID := p.getGitlabToUserIDMapping(gitlabUsername)
	if len(userMattermostID) > 0 {
		p.sendRefreshEvent(userMattermostID)
	}
	return userMattermostID
}

func (p *Plugin) permissionToProject(ctx context.Context, userID, namespace, project string) bool {
	if userID == "" {
		return false
	}

	if err := p.isNamespaceAllowed(namespace); err != nil {
		return false
	}

	info, apiErr := p.getGitlabUserInfoByMattermostID(userID)
	if apiErr != nil {
		return false
	}

	var result *gitlabLib.Project
	err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetProject(ctx, info, token, namespace, project)
		if err != nil {
			return err
		}
		result = resp
		return nil
	})
	if result == nil || err != nil {
		if err != nil {
			p.client.Log.Warn("Can't get project in webhook", "err", err.Error(), "project", namespace+"/"+project)
		}
		return false
	}

	// User permission for the project
	userPermission := result.Permissions

	// Check if the user has guest permission or less for both project and group level
	if (userPermission.ProjectAccess != nil && userPermission.ProjectAccess.AccessLevel <= gitlabLib.GuestPermissions) || (userPermission.GroupAccess != nil && userPermission.GroupAccess.AccessLevel <= gitlabLib.GuestPermissions) {
		return false
	}

	return true
}

func (p *Plugin) createHook(ctx context.Context, gitlabClient gitlab.Gitlab, info *gitlab.UserInfo, group, project string, hookOptions *gitlab.AddWebhookOptions) (*gitlab.WebhookInfo, error) {
	// If project scope
	if project != "" {
		var gitProject *gitlabLib.Project
		getProjectErr := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			resp, err := p.GitlabClient.GetProject(ctx, info, token, group, project)
			if err != nil {
				return err
			}
			gitProject = resp
			return nil
		})
		if getProjectErr != nil {
			return nil, getProjectErr
		}
		var newWebhook *gitlab.WebhookInfo
		getGroupErr := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			resp, err := p.GitlabClient.NewProjectHook(ctx, info, token, gitProject.ID, hookOptions)
			if err != nil {
				return err
			}
			newWebhook = resp
			return nil
		})
		if getGroupErr != nil {
			return nil, getGroupErr
		}
		return newWebhook, nil
	}

	// If webhook is group scoped
	var newWebhook *gitlab.WebhookInfo
	err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.NewGroupHook(ctx, info, token, group, hookOptions)
		if err != nil {
			return err
		}
		newWebhook = resp
		return nil
	})
	if err != nil {
		return nil, err
	}

	return newWebhook, nil
}
