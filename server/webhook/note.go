package webhook

import (
	"fmt"

	"github.com/manland/go-gitlab"
)

func (w *webhook) HandleIssueComment(event *gitlab.IssueCommentEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	message := fmt.Sprintf("[%s](%s) commented on your issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.Issue.IID, event.ObjectAttributes.URL)

	handlers := make([]*HandleWebhook, len(event.Issue.AssigneeIDs)+1)
	for index, assigneeID := range event.Issue.AssigneeIDs {
		handlers[index] = &HandleWebhook{
			Message: message,
			To:      w.gitlabRetreiver.GetUsernameByID(assigneeID),
			From:    senderGitlabUsername,
		}
	}
	handlers[len(handlers)-1] = &HandleWebhook{
		Message: message,
		To:      w.gitlabRetreiver.GetUsernameByID(event.Issue.AuthorID),
		From:    senderGitlabUsername,
	}

	mentions := w.handleMention(mentionDetails{
		senderUsername:    senderGitlabUsername,
		pathWithNamespace: event.Project.PathWithNamespace,
		IID:               event.Issue.IID,
		URL:               event.ObjectAttributes.URL,
		body:              event.ObjectAttributes.Note,
	})
	return cleanWebhookHandlers(append(handlers, mentions...)), nil
}

func (w *webhook) HandleMergeRequestComment(event *gitlab.MergeCommentEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	message := fmt.Sprintf("[%s](%s) commented on your merge request [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.MergeRequest.IID, event.ObjectAttributes.URL)

	handlers := []*HandleWebhook{{
		Message: message,
		To:      w.gitlabRetreiver.GetUsernameByID(event.MergeRequest.AssigneeID),
		From:    senderGitlabUsername,
	}, {
		Message: message,
		To:      w.gitlabRetreiver.GetUsernameByID(event.MergeRequest.AuthorID),
		From:    senderGitlabUsername,
	}}

	mentions := w.handleMention(mentionDetails{
		senderUsername:    senderGitlabUsername,
		pathWithNamespace: event.Project.PathWithNamespace,
		IID:               event.MergeRequest.IID,
		URL:               event.ObjectAttributes.URL,
		body:              event.ObjectAttributes.Note,
	})
	return cleanWebhookHandlers(append(handlers, mentions...)), nil
}
