package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/manland/go-gitlab"
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

	event, err := gitlab.ParseWebhook(gitlab.WebhookEventType(r), body)
	if err != nil {
		p.API.LogError("can't parse webhook", "err", err.Error(), "header", r.Header.Get("X-Gitlab-Event"), "event", string(body))
		return
	}

	var repoPrivate bool
	var handlers []*webhook.HandleWebhook
	var errHandler error

	webhookManager := webhook.NewWebhook(&gitlabRetreiver{p: p}) // TODO build it at init instead at each call
	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil { // TODO build it at init instead at each call
		p.API.LogError("can't get user by username in mattermost api for post merge request event", "err", err.Error())
		return
	} else {
		userID = user.Id
	}

	switch event := event.(type) {
	case *gitlab.MergeEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		handlers, errHandler = webhookManager.HandleMergeRequest(event)
	case *gitlab.IssueEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		handlers, errHandler = webhookManager.HandleIssue(event)
	case *gitlab.IssueCommentEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		handlers, errHandler = webhookManager.HandleIssueComment(event)
	case *gitlab.MergeCommentEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		handlers, errHandler = webhookManager.HandleMergeRequestComment(event)
	case *gitlab.PushEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		// p.postPushEvent(event)
	case *gitlab.PipelineEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		// p.postPipelineEvent(event)
	case *gitlab.TagEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		// p.postTagEvent(event)
	case *gitlab.BuildEvent:
		repoPrivate = event.Repository.Visibility == gitlab.PrivateVisibility
		// p.postBuildEvent(event)
	default:
		p.API.LogWarn("event type not implemented", "type", string(gitlab.WebhookEventType(r)))
		return
	}

	if repoPrivate && !config.EnablePrivateRepo {
		return
	}

	if errHandler != nil {
		p.API.LogError("error handler when building webhook notif", "err", err)
		return
	}

	alreadySentRefresh := make(map[string]bool)
	for _, res := range handlers {
		p.API.LogInfo("new msg", "message", res.Message, "to", "from", res.From)
		for _, to := range res.ToUsers {
			userTo := p.getGitlabToUserIDMapping(to)
			p.API.LogInfo("userTo", "to", userTo)
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
					UserId:    userID,
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

	client := p.gitlabConnect(*info.Token)

	if result, _, err := client.Projects.GetProject(owner+"/"+repo, &gitlab.GetProjectOptions{}); result == nil || err != nil {
		if err != nil {
			p.API.LogError("can't get project in webhook", "err", err.Error(), "project", owner+"/"+repo)
		}
		return false
	}
	return true
}
