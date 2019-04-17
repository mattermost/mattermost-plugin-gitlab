package webhook

import (
	"fmt"

	"github.com/manland/go-gitlab"
)

func (w *webhook) HandleIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error) {
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
		handlers := make([]*HandleWebhook, len(event.Assignees)+1)
		for index, assignee := range event.Assignees {
			handlers[index] = &HandleWebhook{
				Message: message,
				To:      assignee.Username,
				From:    senderGitlabUsername,
			}
		}
		handlers[len(handlers)-1] = &HandleWebhook{
			Message: message,
			To:      authorGitlabUsername,
			From:    senderGitlabUsername,
		}

		mentions := w.handleMention(mentionDetails{
			senderUsername:    senderGitlabUsername,
			pathWithNamespace: event.Project.PathWithNamespace,
			IID:               event.ObjectAttributes.IID,
			URL:               event.ObjectAttributes.URL,
			body:              event.ObjectAttributes.Description,
		})
		return cleanWebhookHandlers(append(handlers, mentions...)), nil
	}
	return []*HandleWebhook{}, nil
}
