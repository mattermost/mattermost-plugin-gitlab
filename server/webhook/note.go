package webhook

import (
	"fmt"

	"github.com/manland/go-gitlab"
)

func (w *webhook) HandleIssueComment(event *gitlab.IssueCommentEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	message := fmt.Sprintf("[%s](%s) commented on your issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.Issue.IID, event.ObjectAttributes.URL)

	toUsers := make([]string, len(event.Issue.AssigneeIDs)+1)
	for index, assigneeID := range event.Issue.AssigneeIDs {
		toUsers[index] = w.gitlabRetreiver.GetUsernameByID(assigneeID)
	}
	toUsers[len(toUsers)-1] = w.gitlabRetreiver.GetUsernameByID(event.Issue.AuthorID)

	handlers := []*HandleWebhook{
		{
			Message: message,
			ToUsers: toUsers,
			From:    senderGitlabUsername,
		},
	}

	if mention := w.handleMention(mentionDetails{
		senderUsername:    senderGitlabUsername,
		pathWithNamespace: event.Project.PathWithNamespace,
		IID:               event.Issue.IID,
		URL:               event.ObjectAttributes.URL,
		body:              event.ObjectAttributes.Note,
	}); mention != nil {
		handlers = append(handlers, mention)
	}

	return cleanWebhookHandlers(handlers), nil
}

func (w *webhook) HandleMergeRequestComment(event *gitlab.MergeCommentEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	message := fmt.Sprintf("[%s](%s) commented on your merge request [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.MergeRequest.IID, event.ObjectAttributes.URL)

	handlers := []*HandleWebhook{{
		Message: message,
		ToUsers: []string{w.gitlabRetreiver.GetUsernameByID(event.MergeRequest.AssigneeID), w.gitlabRetreiver.GetUsernameByID(event.MergeRequest.AuthorID)},
		From:    senderGitlabUsername,
	}}

	if mention := w.handleMention(mentionDetails{
		senderUsername:    senderGitlabUsername,
		pathWithNamespace: event.Project.PathWithNamespace,
		IID:               event.MergeRequest.IID,
		URL:               event.ObjectAttributes.URL,
		body:              event.ObjectAttributes.Note,
	}); mention != nil {
		handlers = append(handlers, mention)
	}
	return cleanWebhookHandlers(handlers), nil
}
