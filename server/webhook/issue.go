package webhook

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandleIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMIssue(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.handleChannelIssue(event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), nil
}

func (w *webhook) handleDMIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error) {
	authorGitlabUsername := w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AuthorID)
	senderGitlabUsername := event.User.Username

	message := ""

	switch event.ObjectAttributes.Action {
	case actionOpen:
		if len(event.Assignees) > 0 {
			message = fmt.Sprintf("[%s](%s) assigned you to issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
		}
	case actionClose:
		message = fmt.Sprintf("[%s](%s) closed your issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	case actionReopen:
		message = fmt.Sprintf("[%s](%s) reopened your issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	}

	if len(message) > 0 {
		toUsers := make([]string, len(event.Assignees)+1)
		for index, assignee := range event.Assignees {
			toUsers[index] = assignee.Username
		}
		toUsers[len(toUsers)-1] = authorGitlabUsername

		handlers := []*HandleWebhook{{
			Message: message,
			ToUsers: toUsers,
			From:    senderGitlabUsername,
		}}

		if mention := w.handleMention(mentionDetails{
			senderUsername:    senderGitlabUsername,
			pathWithNamespace: event.Project.PathWithNamespace,
			IID:               fmt.Sprintf("%d", event.ObjectAttributes.IID),
			URL:               event.ObjectAttributes.URL,
			body:              event.ObjectAttributes.Description,
		}); mention != nil {
			handlers = append(handlers, mention)
		}
		return handlers, nil
	}
	return []*HandleWebhook{}, nil
}

func (w *webhook) handleChannelIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error) {
	issue := event.ObjectAttributes
	senderGitlabUsername := event.User.Username
	repo := event.Project
	res := []*HandleWebhook{}

	message := ""

	switch issue.Action {
	case actionOpen:
		message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n###### new issue by [%s](%s) on [%s](%s)\n\n%s", issue.Title, repo.PathWithNamespace, issue.IID, issue.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), issue.CreatedAt, issue.URL, issue.Description)
	case actionClose:
		message = fmt.Sprintf("[%s] Issue [%s](%s) closed by [%s](%s)", repo.PathWithNamespace, issue.Title, issue.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case actionReopen:
		message = fmt.Sprintf("[%s] Issue [%s](%s) reopened by [%s](%s)", repo.PathWithNamespace, issue.Title, issue.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case actionUpdate:
		if len(event.Changes.Labels.Current) > 0 && !sameLabels(event.Changes.Labels.Current, event.Changes.Labels.Previous) {
			message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n###### issue labeled `%s` by [%s](%s) on [%s](%s)\n\n%s", issue.Title, repo.PathWithNamespace, issue.IID, issue.URL, labelToString(event.Changes.Labels.Current), event.User.Username, event.User.WebsiteURL, issue.UpdatedAt, issue.URL, issue.Description)
		}
	}

	if len(message) > 0 {
		toChannels := make([]string, 0)
		namespace, project := normalizeNamespacedProject(repo.PathWithNamespace)
		subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
			namespace, project,
			repo.Visibility == gitlab.PublicVisibility,
		)
		for _, sub := range subs {
			if !sub.Issues() {
				continue
			}

			if sub.Label() != "" && !containsLabel(event.Labels, sub.Label()) {
				continue
			}

			toChannels = append(toChannels, sub.ChannelID)
		}

		if len(toChannels) > 0 {
			res = append(res, &HandleWebhook{
				From:       senderGitlabUsername,
				Message:    message,
				ToUsers:    []string{},
				ToChannels: toChannels,
			})
		}
	}
	return res, nil
}
