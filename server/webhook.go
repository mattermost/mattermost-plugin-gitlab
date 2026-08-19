// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
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

	webhookDedupTTL         = 30 * time.Second
	notificationDedupKeyFmt = "notif_dedup_%s"
	channelPostDedupKeyFmt  = "chan_dedup_%s"
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
				p.sendChannelNotification(to, res.Message)
			}
		}
		p.sendRefreshIfNotAlreadySent(alreadySentRefresh, res.From)
	}
}

// writeDedupToken length-prefixes s so distinct token sequences can't hash to
// the same key (e.g. "a_b"+"c" versus "a"+"b_c").
func writeDedupToken(h hash.Hash, s string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(s)))
	// hash.Hash.Write never returns an error.
	_, _ = h.Write(length[:])
	_, _ = h.Write([]byte(s))
}

func notificationDedupKey(recipientID, message string) string {
	h := sha256.New()
	for _, token := range []string{recipientID, message} {
		writeDedupToken(h, token)
	}
	return fmt.Sprintf(notificationDedupKeyFmt, hex.EncodeToString(h.Sum(nil)))
}

func channelPostDedupKey(channelID, message string) string {
	h := sha256.New()
	for _, token := range []string{channelID, message} {
		writeDedupToken(h, token)
	}
	return fmt.Sprintf(channelPostDedupKeyFmt, hex.EncodeToString(h.Sum(nil)))
}

// claimDedupKey atomically claims dedupKey: SetAtomic(nil) only writes when the
// key is absent, so concurrent deliveries can't both win. A KV error fails open,
// returning deliver without owned since nothing was written.
func (p *Plugin) claimDedupKey(caller, dedupKey string) (deliver, owned bool) {
	claimed, err := p.client.KV.Set(dedupKey, true,
		pluginapi.SetExpiry(webhookDedupTTL),
		pluginapi.SetAtomic(nil),
	)
	switch {
	case err != nil:
		p.client.Log.Warn(caller+": failed to claim dedup key, delivering anyway", "dedup_key", dedupKey, "err", err.Error())
		return true, false
	case !claimed:
		p.client.Log.Debug(caller+": another delivery already claimed this notification, skipping", "dedup_key", dedupKey)
		return false, false
	}
	return true, true
}

// releaseDedupKey lets a redelivery retry instead of waiting out the TTL. Only
// call it when this delivery owns the claim and nothing was posted.
func (p *Plugin) releaseDedupKey(caller, dedupKey string) {
	if err := p.client.KV.Delete(dedupKey); err != nil {
		p.client.Log.Warn(caller+": failed to release dedup key; redeliveries of this event will be skipped until the claim expires",
			"dedup_key", dedupKey, "err", err.Error())
	}
}

func (p *Plugin) sendDMNotification(userID, message string) {
	dedupKey := notificationDedupKey(userID, message)
	deliver, owned := p.claimDedupKey("sendDMNotification", dedupKey)
	if !deliver {
		return
	}

	if err := p.CreateBotDMPost(userID, message, "custom_git_review_request"); err != nil {
		p.client.Log.Warn("can't send dm post", "err", err.Error())
		// CreatePost may have persisted the DM despite erroring, so only the
		// pre-post lookup failure is safe to release.
		if owned && errors.Is(err, errDMChannelUnavailable) {
			p.releaseDedupKey("sendDMNotification", dedupKey)
		}
	}
}

// sendChannelNotification keys the claim on channel and message rather than on
// the subscription, so overlapping subscriptions collapse into one post.
func (p *Plugin) sendChannelNotification(channelID, message string) {
	dedupKey := channelPostDedupKey(channelID, message)
	if deliver, _ := p.claimDedupKey("sendChannelNotification", dedupKey); !deliver {
		return
	}

	post := &model.Post{
		UserId:    p.BotUserID,
		Message:   message,
		ChannelId: channelID,
	}
	// The claim is deliberately kept on failure: CreatePost may have persisted
	// the post, and releasing would let a redelivery duplicate it.
	if err := p.client.Post.CreatePost(post); err != nil {
		p.client.Log.Warn("can't create post for webhook event", "err", err.Error())
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
