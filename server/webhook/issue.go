package webhook

import (
	"fmt"

	"github.com/manland/go-gitlab"
)

func (w *webhook) HandleIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error) {
	return w.handleDMIssue(event)
}

func (w *webhook) handleDMIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error) {
	authorGitlabUsername := w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AuthorID)
	senderGitlabUsername := event.User.Username

	message := ""

	if event.ObjectAttributes.Action == "open" && len(event.Assignees) > 0 {
		message = fmt.Sprintf("[%s](%s) assigned you to issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.Action == "close" {
		message = fmt.Sprintf("[%s](%s) closed your issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.Action == "reopen" {
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
			IID:               event.ObjectAttributes.IID,
			URL:               event.ObjectAttributes.URL,
			body:              event.ObjectAttributes.Description,
		}); mention != nil {
			handlers = append(handlers, mention)
		}
		return cleanWebhookHandlers(handlers), nil
	}
	return []*HandleWebhook{}, nil
}

func (w *webhook) handleChannelIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error) {
	//TODO FINISH ME
	return nil, nil
}
