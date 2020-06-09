package webhook

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandleMergeRequest(event *gitlab.MergeEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMMergeRequest(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.handleChannelMergeRequest(event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), nil
}

func (w *webhook) handleDMMergeRequest(event *gitlab.MergeEvent) ([]*HandleWebhook, error) {
	authorGitlabUsername := w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AuthorID)
	senderGitlabUsername := event.User.Username

	message := ""

	if event.ObjectAttributes.State == "opened" && event.ObjectAttributes.Action == "open" {
		message = fmt.Sprintf("[%s](%s) requested your review on [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.State == "closed" {
		message = fmt.Sprintf("[%s](%s) closed your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.State == "opened" && event.ObjectAttributes.Action == "reopen" {
		message = fmt.Sprintf("[%s](%s) reopen your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.State == "opened" && event.ObjectAttributes.Action == "update" {
		// TODO not enough check (opened/update) to say assigned to you...
		message = fmt.Sprintf("[%s](%s) assigned you to merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	} else if event.ObjectAttributes.State == "merged" && event.ObjectAttributes.Action == "merge" {
		message = fmt.Sprintf("[%s](%s) merged your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	}

	if len(message) > 0 {
		handlers := []*HandleWebhook{{
			Message:    message,
			ToUsers:    []string{w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AssigneeID), authorGitlabUsername},
			ToChannels: []string{},
			From:       senderGitlabUsername,
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
	return []*HandleWebhook{{From: senderGitlabUsername}}, nil
}

func (w *webhook) handleChannelMergeRequest(event *gitlab.MergeEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	pr := event.ObjectAttributes
	repo := event.Project
	res := []*HandleWebhook{}

	message := ""

	switch pr.Action {
	case "open":
		message = fmt.Sprintf("#### %s\n##### [%s!%v](%s) new merge-request by [%s](%s) on [%s](%s)\n\n%s", pr.Title, repo.PathWithNamespace, pr.IID, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), pr.CreatedAt, pr.URL, pr.Description)
	case "merge":
		message = fmt.Sprintf("[%s] Merge request [!%v %s](%s) was merged by [%s](%s)", repo.PathWithNamespace, pr.IID, pr.Title, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case "close":
		message = fmt.Sprintf("[%s] Merge request [!%v %s](%s) was closed by [%s](%s)", repo.PathWithNamespace, pr.IID, pr.Title, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case "reopen":
		message = fmt.Sprintf("[%s] Merge request [!%v %s](%s) was reopened by [%s](%s)", repo.PathWithNamespace, pr.IID, pr.Title, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	}

	if len(message) > 0 {
		toChannels := make([]string, 0)
		namespace, project := normalizeNamespacedProject(repo.PathWithNamespace)
		subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
			namespace, project,
			repo.Visibility == gitlab.PublicVisibility,
		)
		for _, sub := range subs {
			if !sub.Merges() {
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
