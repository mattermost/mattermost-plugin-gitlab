package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	gitlabLib "github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
	"github.com/mattermost/mattermost-plugin-gitlab/server/webhook"

	"github.com/mattermost/mattermost-server/v6/model"
)

type gitlabRetreiver struct {
	p *Plugin
}

func (g *gitlabRetreiver) GetPipelineURL(pathWithNamespace string, pipelineID int) string {
	config := g.p.getConfiguration()
	return fmt.Sprintf("%s/%s/-/pipelines/%d", config.GitlabURL, pathWithNamespace, pipelineID)
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
	namespace string,
	project string,
	isPublicVisibility bool,
) []*subscription.Subscription {
	return g.p.GetSubscribedChannelsForProject(namespace, project, isPublicVisibility)
}

func (p *Plugin) handleWebhook(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	signature := r.Header.Get("X-Gitlab-Token")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request body", http.StatusBadRequest)
		return
	}

	if config.WebhookSecret != signature {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	event, err := gitlabLib.ParseWebhook(gitlabLib.WebhookEventType(r), body)
	if err != nil {
		p.API.LogError("can't parse webhook", "err", err.Error(), "header", r.Header.Get("X-Gitlab-Event"), "event", string(body))
		http.Error(w, "Unable to handle request", http.StatusBadRequest)
		return
	}

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
		handlers, errHandler = p.WebhookHandler.HandleMergeRequest(event)
	case *gitlabLib.IssueEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandleIssue(event)
	case *gitlabLib.IssueCommentEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandleIssueComment(event)
	case *gitlabLib.MergeCommentEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandleMergeRequestComment(event)
	case *gitlabLib.PushEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.UserName
		handlers, errHandler = p.WebhookHandler.HandlePush(event)
	case *gitlabLib.PipelineEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.User.Username
		handlers, errHandler = p.WebhookHandler.HandlePipeline(event)
	case *gitlabLib.TagEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		fromUser = event.UserName
		handlers, errHandler = p.WebhookHandler.HandleTag(event)
	default:
		p.API.LogWarn("event type not implemented", "type", string(gitlabLib.WebhookEventType(r)))
		return
	}

	if repoPrivate && !config.EnablePrivateRepo {
		return
	}

	if errCheckGroup := p.isNamespaceAllowed(pathWithNamespace); errCheckGroup != nil {
		return
	}

	if errHandler != nil {
		p.API.LogError("error handler when building webhook notif", "err", err)
		return
	}

	alreadySentRefresh := make(map[string]bool)
	p.sendRefreshIfNotAlreadySent(alreadySentRefresh, fromUser)
	for _, res := range handlers {
		p.API.LogInfo("new msg", "message", res.Message, "from", res.From)
		for _, to := range res.ToUsers {
			userTo := p.sendRefreshIfNotAlreadySent(alreadySentRefresh, to)
			if len(userTo) > 0 && len(res.Message) > 0 {
				info, err := p.getGitlabUserInfoByMattermostID(userTo)
				if err != nil {
					p.API.LogError("can't get user info to know if user wants to receive notifications", "err", err.Message)
					continue
				}
				if info.Settings.Notifications {
					if err := p.CreateBotDMPost(userTo, res.Message, "custom_git_review_request"); err != nil {
						p.API.LogError("can't send dm post", "err", err.Error())
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
				if _, err := p.API.CreatePost(post); err != nil {
					p.API.LogError("can't create post for webhook event", "err", err.Error())
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

func (p *Plugin) permissionToProject(userID, namespace, project string) bool {
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

	if result, err := p.GitlabClient.GetProject(info, namespace, project); result == nil || err != nil {
		if err != nil {
			p.API.LogError("can't get project in webhook", "err", err.Error(), "project", namespace+"/"+project)
		}
		return false
	}
	return true
}
