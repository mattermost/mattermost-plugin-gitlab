package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	gitlabLib "github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
	"github.com/mattermost/mattermost-plugin-gitlab/server/webhook"

	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	webhookTimeout = 10 * time.Second
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
	fromUser := ""

	switch event := event.(type) {
	case *gitlabLib.MergeEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandleMergeRequest(ctx, event)
	case *gitlabLib.IssueEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandleIssue(ctx, event, eventType)
	case *gitlabLib.IssueCommentEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandleIssueComment(ctx, event)
	case *gitlabLib.MergeCommentEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandleMergeRequestComment(ctx, event)
	case *gitlabLib.PushEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.UserName
		handlers, errHandler = p.WebhookHandler.HandlePush(ctx, event)
	case *gitlabLib.PipelineEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
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
	default:
		p.client.Log.Debug("Event type not implemented", "type", string(gitlabLib.WebhookEventType(r)))
		return
	}

	if repoPrivate && !config.EnablePrivateRepo {
		return
	}

	if err = p.isNamespaceAllowed(pathWithNamespace); err != nil {
		return
	}

	if errHandler != nil {
		p.client.Log.Debug("Error when handling webhook event", "err", errHandler)
		return
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
					if err := p.CreateBotDMPost(userTo, res.Message, "custom_git_review_request"); err != nil {
						p.client.Log.Warn("can't send dm post", "err", err.Error())
					}
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

	result, err := p.GitlabClient.GetProject(ctx, info, namespace, project)
	if result == nil || err != nil {
		if err != nil {
			p.API.LogWarn("Can't get project in webhook", "err", err.Error(), "project", namespace+"/"+project)
		}
		return false
	}

	// Check for guest level permissions
	if result.Permissions.ProjectAccess != nil && result.Permissions.ProjectAccess.AccessLevel == gitlabLib.GuestPermissions {
		return false
	}

	return true
}

func CreateHook(ctx context.Context, gitlabClient gitlab.Gitlab, info *gitlab.UserInfo, group, project string, hookOptions *gitlab.AddWebhookOptions) (*gitlab.WebhookInfo, error) {
	// If project scope
	if project != "" {
		project, err := gitlabClient.GetProject(ctx, info, group, project)
		if err != nil {
			return nil, err
		}
		newWebhook, err := gitlabClient.NewProjectHook(ctx, info, project.ID, hookOptions)
		if err != nil {
			return nil, err
		}
		return newWebhook, nil
	}

	// If webhook is group scoped
	newWebhook, err := gitlabClient.NewGroupHook(ctx, info, group, hookOptions)
	if err != nil {
		return nil, err
	}

	return newWebhook, nil
}
