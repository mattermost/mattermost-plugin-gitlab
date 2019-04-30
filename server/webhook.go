package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	gitlabLib "github.com/manland/go-gitlab"

	"github.com/manland/mattermost-plugin-gitlab/server/gitlab"
	"github.com/manland/mattermost-plugin-gitlab/server/subscription"
	"github.com/manland/mattermost-plugin-gitlab/server/webhook"

	"github.com/mattermost/mattermost-server/model"
)

type gitlabRetreiver struct {
	p *Plugin
}

func (g *gitlabRetreiver) GetUserURL(username string) string {
	config := g.p.getConfiguration()
	url := "https://gitlab.com"
	if config.EnterpriseBaseURL != "" {
		url = config.EnterpriseBaseURL
	}
	return fmt.Sprintf("%s/%s", url, username)
}

func (g *gitlabRetreiver) GetUsernameByID(id int) string {
	return g.p.getGitlabIDToUsernameMapping(fmt.Sprintf("%d", id))
}

func (g *gitlabRetreiver) ParseGitlabUsernamesFromText(text string) []string {
	return parseGitlabUsernamesFromText(text)
}

func (g *gitlabRetreiver) GetSubscribedChannelsForRepository(repoWithNamespace string, isPublicVisibility bool) []*subscription.Subscription {
	return g.p.GetSubscribedChannelsForRepository(repoWithNamespace, isPublicVisibility)
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
		return
	}

	var repoPrivate bool
	var pathWithNamespace string
	var handlers []*webhook.HandleWebhook
	var errHandler error

	switch event := event.(type) {
	case *gitlabLib.MergeEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		handlers, errHandler = p.WebhookHandler.HandleMergeRequest(event)
	case *gitlabLib.IssueEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		handlers, errHandler = p.WebhookHandler.HandleIssue(event)
	case *gitlabLib.IssueCommentEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		handlers, errHandler = p.WebhookHandler.HandleIssueComment(event)
	case *gitlabLib.MergeCommentEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		handlers, errHandler = p.WebhookHandler.HandleMergeRequestComment(event)
	case *gitlabLib.PushEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		handlers, errHandler = p.WebhookHandler.HandlePush(event)
	case *gitlabLib.PipelineEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		handlers, errHandler = p.WebhookHandler.HandlePipeline(event)
	case *gitlabLib.TagEvent:
		repoPrivate = event.Project.Visibility == gitlabLib.PrivateVisibility
		pathWithNamespace = event.Project.PathWithNamespace
		handlers, errHandler = p.WebhookHandler.HandleTag(event)
	default:
		p.API.LogWarn("event type not implemented", "type", string(gitlabLib.WebhookEventType(r)))
		return
	}

	if repoPrivate && !config.EnablePrivateRepo {
		return
	}

	if errCheckGroup := p.checkGroup(pathWithNamespace); errCheckGroup != nil {
		return
	}

	if errHandler != nil {
		p.API.LogError("error handler when building webhook notif", "err", err)
		return
	}

	alreadySentRefresh := make(map[string]bool)
	for _, res := range handlers {
		p.API.LogInfo("new msg", "message", res.Message, "from", res.From)
		for _, to := range res.ToUsers {
			userTo := p.getGitlabToUserIDMapping(to)
			if !alreadySentRefresh[userTo] {
				alreadySentRefresh[userTo] = true
				p.sendRefreshEvent(userTo)
			}
			if len(res.Message) > 0 {
				if err := p.CreateBotDMPost(userTo, res.Message, "custom_git_review_request"); err != nil {
					p.API.LogError("can't send dm post", "err", err.DetailedError)
				}
			}
		}
		for _, to := range res.ToChannels {
			if len(res.Message) > 0 {
				post := &model.Post{
					UserId:    p.BotUserID,
					Message:   res.Message,
					ChannelId: to,
					Props: map[string]interface{}{
						"from_webhook":      "true",
						"override_username": GITLAB_USERNAME,
						"override_icon_url": config.ProfileImageURL,
					},
				}
				if _, err := p.API.CreatePost(post); err != nil {
					p.API.LogError("can't crate post for webhook event", "err", err.Error())
				}
			}
		}
		if len(res.From) > 0 {
			userFrom := p.getGitlabToUserIDMapping(res.From)
			p.API.LogInfo("userFrom", "from", userFrom)
			if !alreadySentRefresh[userFrom] {
				alreadySentRefresh[userFrom] = true
				p.sendRefreshEvent(userFrom)
			}
		}
	}
}

func (p *Plugin) permissionToRepo(userID string, fullPath string) bool {
	if userID == "" {
		return false
	}

	config := p.getConfiguration()
	_, owner, repo := parseOwnerAndRepo(fullPath, config.EnterpriseBaseURL)

	if owner == "" {
		return false
	}

	if err := p.checkGroup(fullPath); err != nil {
		return false
	}

	info, apiErr := p.getGitlabUserInfoByMattermostID(userID)
	if apiErr != nil {
		return false
	}

	if result, err := gitlab.New(config.EnterpriseBaseURL).GetProject(info, owner, repo); result == nil || err != nil {
		if err != nil {
			p.API.LogError("can't get project in webhook", "err", err.Error(), "project", owner+"/"+repo)
		}
		return false
	}
	return true
}
