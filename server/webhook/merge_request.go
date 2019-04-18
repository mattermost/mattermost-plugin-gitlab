package webhook

import (
	"fmt"

	"github.com/manland/go-gitlab"
)

func (w *webhook) HandleMergeRequest(event *gitlab.MergeEvent) ([]*HandleWebhook, error) {
	authorGitlabUsername := w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AuthorID)
	senderGitlabUsername := event.User.Username

	message := ""

	if event.ObjectAttributes.State == "opened" && event.ObjectAttributes.Action == "open" {
		message = fmt.Sprintf("[%s](%s) requested your review on [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.State == "closed" {
		message = fmt.Sprintf("[%s](%s) closed your pull request [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.State == "opened" && event.ObjectAttributes.Action == "reopen" {
		message = fmt.Sprintf("[%s](%s) reopen your pull request [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.State == "opened" && event.ObjectAttributes.Action == "update" {
		// TODO not enough check (opened/update) to say assignee to you...
		message = fmt.Sprintf("[%s](%s) assigned you to pull request [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.State == "merged" && event.ObjectAttributes.Action == "merge" {
		message = fmt.Sprintf("[%s](%s) merged your pull request [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	}

	if len(message) > 0 {
		handlers := []*HandleWebhook{{
			Message: message,
			ToUsers: []string{w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AssigneeID), authorGitlabUsername},
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
	return []*HandleWebhook{{From: senderGitlabUsername}}, nil
}
