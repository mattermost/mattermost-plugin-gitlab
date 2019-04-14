package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/manland/mattermost-plugin-gitlab/server/webhook"
	"github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-server/mlog"
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

	p.API.LogWarn("webhook", "header", r.Header.Get("X-Gitlab-Event"), "event", string(body))

	event, err := gitlab.ParseWebhook(gitlab.WebhookEventType(r), body)
	if err != nil {
		mlog.Error(err.Error())
		return
	}

	var repoPrivate bool
	var handlers []*webhook.HandleWebhook
	var errHandler error

	webhookManager := webhook.NewWebhook(&gitlabRetreiver{p: p}) // TODO build it at init instead at each call

	//TODO move postXXX to package webhook and test it
	switch event := event.(type) {
	case *gitlab.MergeEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		p.postMergeRequestEvent(event)
		handlers, errHandler = webhookManager.HandleMergeRequest(event)
	case *gitlab.IssueEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		p.postIssueEvent(event)
		handlers, errHandler = webhookManager.HandleIssue(event)
	case *gitlab.IssueCommentEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		p.postIssueCommentEvent(event)
		handlers, errHandler = webhookManager.HandleIssueComment(event)
	case *gitlab.MergeCommentEvent:
		repoPrivate = event.Project.Visibility == gitlab.PrivateVisibility
		p.postMergeCommentEvent(event)
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

	for _, res := range handlers {
		p.API.LogError("new msg", "message", res.Message, "to", res.To, "from", res.From)
		if len(res.To) > 0 {
			userTo := p.getGitlabToUserIDMapping(res.To)
			p.API.LogError("userTo", "to", userTo)
			p.sendRefreshEvent(userTo)
			if len(res.Message) > 0 {
				p.CreateBotDMPost(userTo, res.Message, "custom_git_review_request")
			}
		}
		if len(res.From) > 0 {
			userFrom := p.getGitlabToUserIDMapping(res.From)
			p.API.LogError("userFrom", "from", userFrom)
			p.sendRefreshEvent(userFrom)
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
			mlog.Error(err.Error())
		}
		return false
	}
	return true
}

func (p *Plugin) postMergeRequestEvent(event *gitlab.MergeEvent) {
	config := p.getConfiguration()
	repo := event.Project

	subs := p.GetSubscribedChannelsForRepository(repo.PathWithNamespace, repo.Visibility == gitlab.PublicVisibility)
	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	pr := event.ObjectAttributes
	prUser := event.User

	if pr.Action == "update" {
		return
	}

	newPRMessage := fmt.Sprintf(`
#### %s
##### [%s#%v](%s)
# new-pull-request by [%s](%s) on [%s](%s)

%s
`, pr.Title, repo.PathWithNamespace, pr.IID, pr.URL, prUser.Username, prUser.WebsiteURL, pr.CreatedAt, pr.URL, pr.Description)

	fmtCloseMessage := ""
	if pr.MergeStatus == "merged" {
		fmtCloseMessage = "[%s] Pull request [#%v %s](%s) was merged by [%s](%s)"
	} else {
		fmtCloseMessage = "[%s] Pull request [#%v %s](%s) was closed by [%s](%s)"
	}
	closedPRMessage := fmt.Sprintf(fmtCloseMessage, repo.PathWithNamespace, pr.IID, pr.Title, pr.URL, prUser.Username, prUser.WebsiteURL)

	post := &model.Post{
		UserId: userID,
		Type:   "custom_git_pr",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITLAB_USERNAME,
			"override_icon_url": config.ProfileImageURL,
		},
	}

	for _, sub := range subs {
		if !sub.Pulls() {
			continue
		}

		//TODO manage label like issues
		label := sub.Label()

		contained := false
		for _, v := range event.Changes.Labels.Current {
			if v.Name == label {
				contained = true
			}
		}

		if !contained && label != "" {
			continue
		}

		if pr.State == "opened" {
			post.Message = newPRMessage
		}

		if pr.State == "closed" {
			post.Message = closedPRMessage
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postIssueEvent(event *gitlab.IssueEvent) {
	config := p.getConfiguration()
	repo := event.Project

	subs := p.GetSubscribedChannelsForRepository(repo.PathWithNamespace, repo.Visibility == gitlab.PublicVisibility)
	if subs == nil || len(subs) == 0 {
		return
	}

	action := event.ObjectAttributes.Action
	if action != "open" && action != "update" && action != "close" {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	issue := event.ObjectAttributes
	issueUser := event.User
	labels := make([]string, len(event.Labels))
	for i, v := range event.Labels {
		labels[i] = v.Name
	}

	newIssueMessage := fmt.Sprintf(`
#### %s
##### [%s#%v](%s)
# new-issue by [%s](%s) on [%s](%s)

%s
`, issue.Title, repo.PathWithNamespace, issue.IID, issue.URL, issueUser.Username, issueUser.WebsiteURL, issue.CreatedAt, issue.URL, issue.Description)

	closedIssueMessage := fmt.Sprintf("\\[%s] Issue [%s](%s) closed by [%s](%s)",
		repo.PathWithNamespace, issue.Title, issue.URL, issueUser.Username, issueUser.WebsiteURL)

	post := &model.Post{
		UserId: userID,
		Type:   "custom_git_issue",
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITLAB_USERNAME,
			"override_icon_url": config.ProfileImageURL,
		},
	}

	for _, sub := range subs {
		if !sub.Issues() {
			continue
		}

		label := sub.Label()

		contained := false
		for _, v := range labels {
			if v == label {
				contained = true
			}
		}

		if !contained && label != "" {
			continue
		}

		if action == "update" && len(event.Changes.Labels.Current) > 0 && !sameLabels(event.Changes.Labels.Current, event.Changes.Labels.Previous) {
			if label == "" || containsLabel(event.Labels, label) {
				post.Message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n#issue-labeled `%s` by [%s](%s) on [%s](%s)\n\n%s", issue.Title, repo.PathWithNamespace, issue.IID, issue.URL, labelToString(event.Changes.Labels.Current), event.User.Username, event.User.WebsiteURL, issue.UpdatedAt, issue.URL, issue.Description)
			} else {
				continue
			}
		}

		if action == "open" {
			post.Message = newIssueMessage
		}

		if action == "close" {
			post.Message = closedIssueMessage
		}

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func sameLabels(a []gitlab.Label, b []gitlab.Label) bool {
	if len(a) != len(b) {
		return false
	}
	for index, l := range a {
		if l.ID != b[index].ID {
			return false
		}
	}
	return true
}

func containsLabel(a []gitlab.Label, labelName string) bool {
	for _, l := range a {
		if l.Name == labelName {
			return true
		}
	}
	return false
}

func labelToString(a []gitlab.Label) string {
	names := make([]string, len(a))
	for index, l := range a {
		names[index] = l.Name
	}
	return strings.Join(names, ", ")
}

// func (p *Plugin) postPushEvent(event *github.PushEvent) {
// 	config := p.getConfiguration()
// 	repo := event.GetRepo()

// 	subs := p.GetSubscribedChannelsForRepository(ConvertPushEventRepositoryToRepository(repo))

// 	if subs == nil || len(subs) == 0 {
// 		return
// 	}

// 	userID := ""
// 	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
// 		mlog.Error(err.Error())
// 		return
// 	} else {
// 		userID = user.Id
// 	}

// 	forced := event.GetForced()
// 	branch := strings.Replace(event.GetRef(), "refs/heads/", "", 1)
// 	commits := event.Commits
// 	compare_url := event.GetCompare()
// 	pusher := event.GetSender()

// 	if len(commits) == 0 {
// 		return
// 	}

// 	fmtMessage := ``
// 	if forced {
// 		fmtMessage = "[%s](%s) force-pushed [%d new commits](%s) to [\\[%s:%s\\]](%s/tree/%s):\n"
// 	} else {
// 		fmtMessage = "[%s](%s) pushed [%d new commits](%s) to [\\[%s:%s\\]](%s/tree/%s):\n"
// 	}
// 	newPushMessage := fmt.Sprintf(fmtMessage, pusher.GetLogin(), pusher.GetHTMLURL(), len(commits), compare_url, repo.GetName(), branch, repo.GetHTMLURL(), branch)
// 	for _, commit := range commits {
// 		newPushMessage += fmt.Sprintf("[`%s`](%s) %s - %s\n",
// 			commit.GetID()[:6], commit.GetURL(), commit.GetMessage(), commit.GetCommitter().GetName())
// 	}

// 	post := &model.Post{
// 		UserId: userID,
// 		Type:   "custom_git_push",
// 		Props: map[string]interface{}{
// 			"from_webhook":      "true",
// 			"override_username": GITHUB_USERNAME,
// 			"override_icon_url": config.ProfileImageURL,
// 		},
// 		Message: newPushMessage,
// 	}

// 	for _, sub := range subs {
// 		if !sub.Pushes() {
// 			continue
// 		}

// 		post.ChannelId = sub.ChannelID
// 		if _, err := p.API.CreatePost(post); err != nil {
// 			mlog.Error(err.Error())
// 		}
// 	}
// }

func (p *Plugin) postIssueCommentEvent(event *gitlab.IssueCommentEvent) {
	config := p.getConfiguration()
	repo := event.Project

	subs := p.GetSubscribedChannelsForRepository(repo.PathWithNamespace, repo.Visibility == gitlab.PublicVisibility)
	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	body := event.ObjectAttributes.Note

	message := fmt.Sprintf("[\\[%s\\]](%s) New comment by [%s](%s) on [#%v %s]:\n\n%s",
		repo.PathWithNamespace, repo.URL, event.User.Username, event.User.WebsiteURL, event.Issue.IID, event.Issue.Title, body)

	post := &model.Post{
		UserId:  userID,
		Type:    "custom_git_comment",
		Message: message,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITLAB_USERNAME,
			"override_icon_url": config.ProfileImageURL,
		},
	}

	// TODO labels !?
	// labels := make([]string, len(event.Issue.Labels))
	// for i, v := range event.Issue.Labels {
	// 	labels[i] = v
	// }

	for _, sub := range subs {
		if !sub.IssueComments() {
			continue
		}

		// label := sub.Label()

		// contained := false
		// for _, v := range labels {
		// 	if v == label {
		// 		contained = true
		// 	}
		// }

		// if !contained && label != "" {
		// 	continue
		// }

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}

func (p *Plugin) postMergeCommentEvent(event *gitlab.MergeCommentEvent) {
	config := p.getConfiguration()
	repo := event.Project

	subs := p.GetSubscribedChannelsForRepository(repo.PathWithNamespace, repo.Visibility == gitlab.PublicVisibility)
	if subs == nil || len(subs) == 0 {
		return
	}

	userID := ""
	if user, err := p.API.GetUserByUsername(config.Username); err != nil {
		mlog.Error(err.Error())
		return
	} else {
		userID = user.Id
	}

	body := event.ObjectAttributes.Note

	message := fmt.Sprintf("[\\[%s\\]](%s) New comment by [%s](%s) on [#%v %s]:\n\n%s",
		repo.PathWithNamespace, repo.URL, event.User.Username, event.User.WebsiteURL, event.MergeRequest.IID, event.MergeRequest.Title, body)

	post := &model.Post{
		UserId:  userID,
		Type:    "custom_git_comment",
		Message: message,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITLAB_USERNAME,
			"override_icon_url": config.ProfileImageURL,
		},
	}

	// TODO labels !?
	// labels := make([]string, len(event.Issue.Labels))
	// for i, v := range event.Issue.Labels {
	// 	labels[i] = v
	// }

	for _, sub := range subs {
		if !sub.IssueComments() {
			continue
		}

		// label := sub.Label()

		// contained := false
		// for _, v := range labels {
		// 	if v == label {
		// 		contained = true
		// 	}
		// }

		// if !contained && label != "" {
		// 	continue
		// }

		post.ChannelId = sub.ChannelID
		if _, err := p.API.CreatePost(post); err != nil {
			mlog.Error(err.Error())
		}
	}
}
