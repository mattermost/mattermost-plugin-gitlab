package webhook

import (
	"context"
	"fmt"
	"strings"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandleMergeRequest(ctx context.Context, event *gitlab.MergeEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMMergeRequest(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.handleChannelMergeRequest(ctx, event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), nil
}

func (w *webhook) handleDMMergeRequest(event *gitlab.MergeEvent) ([]*HandleWebhook, error) {
	authorGitlabUsername := w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AuthorID)
	senderGitlabUsername := event.User.Username

	var added, removed []*gitlab.EventUser
	message := ""

	switch event.ObjectAttributes.State {
	case stateOpened:
		switch event.ObjectAttributes.Action {
		case actionOpen:
			message = fmt.Sprintf("[%s](%s) requested your review on [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
		case actionReopen:
			message = fmt.Sprintf("[%s](%s) reopen your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
		case actionUpdate:
			if event.ObjectAttributes.OldRev != "" {
				message = fmt.Sprintf("[%s](%s) pushed a new commit to merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
			} else {
				prevAssignees := event.Changes.Assignees.Previous
				currAssignees := event.Changes.Assignees.Current
				added, removed = diffAssignees(prevAssignees, currAssignees)

				if len(added) > 0 && len(removed) > 0 {
					message = fmt.Sprintf("[%s](%s) updated assignees, added: %s, removed: %s in merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), formatAssignees(added), formatAssignees(removed), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
				} else if len(added) > 0 {
					message = fmt.Sprintf("[%s](%s) added assignees: %s to merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), formatAssignees(added), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
				} else if len(removed) > 0 {
					message = fmt.Sprintf("[%s](%s) removed assignees: %s from merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), formatAssignees(removed), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
				}
			}
		case actionApproved:
			message = fmt.Sprintf("[%s](%s) approved your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
		case actionUnapproved:
			message = fmt.Sprintf("[%s](%s) requested changes to your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
		}
	case stateClosed:
		message = fmt.Sprintf("[%s](%s) closed your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	case stateMerged:
		if event.ObjectAttributes.Action == actionMerge {
			message = fmt.Sprintf("[%s](%s) merged your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
		}
	}

	if len(message) > 0 {
		toUsers := []string{w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AssigneeID), authorGitlabUsername}
		for _, u := range added {
			toUsers = append(toUsers, w.gitlabRetreiver.GetUsernameByID(u.ID))
		}
		for _, u := range removed {
			toUsers = append(toUsers, w.gitlabRetreiver.GetUsernameByID(u.ID))
		}
		handlers := []*HandleWebhook{{
			Message:    message,
			ToUsers:    toUsers,
			ToChannels: []string{},
			From:       senderGitlabUsername,
		}}

		if mention := w.handleMention(mentionDetails{
			senderUsername:    senderGitlabUsername,
			pathWithNamespace: event.Project.PathWithNamespace,
			IID:               fmt.Sprintf("%d", event.ObjectAttributes.IID),
			URL:               event.ObjectAttributes.URL,
			body:              sanitizeDescription(event.ObjectAttributes.Description),
		}); mention != nil {
			handlers = append(handlers, mention)
		}
		return handlers, nil
	}
	return []*HandleWebhook{{From: senderGitlabUsername}}, nil
}

func (w *webhook) handleChannelMergeRequest(ctx context.Context, event *gitlab.MergeEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	pr := event.ObjectAttributes
	repo := event.Project
	res := []*HandleWebhook{}

	message := ""

	switch pr.Action {
	case actionOpen:
		message = fmt.Sprintf("#### %s\n##### [%s!%v](%s) new merge-request by [%s](%s) on [%s](%s)\n\n%s", pr.Title, repo.PathWithNamespace, pr.IID, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), pr.CreatedAt, pr.URL, sanitizeDescription(pr.Description))
	case actionMerge:
		message = fmt.Sprintf("[%s](%s) Merge request [!%v %s](%s) was merged by [%s](%s)", repo.PathWithNamespace, repo.WebURL, pr.IID, pr.Title, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case actionClose:
		message = fmt.Sprintf("[%s](%s) Merge request [!%v %s](%s) was closed by [%s](%s)", repo.PathWithNamespace, repo.WebURL, pr.IID, pr.Title, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case actionReopen:
		message = fmt.Sprintf("[%s](%s) Merge request [!%v %s](%s) was reopened by [%s](%s)", repo.PathWithNamespace, repo.WebURL, pr.IID, pr.Title, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case actionApproved:
		message = fmt.Sprintf("[%s](%s) Merge request [!%v %s](%s) was approved by [%s](%s)", repo.PathWithNamespace, repo.WebURL, pr.IID, pr.Title, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case actionUnapproved:
		message = fmt.Sprintf("[%s](%s) Merge request [!%v %s](%s) changes were requested by [%s](%s)", repo.PathWithNamespace, repo.WebURL, pr.IID, pr.Title, pr.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	}

	if len(message) > 0 {
		toChannels := make([]string, 0)
		namespace, project := normalizeNamespacedProject(repo.PathWithNamespace)
		subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
			ctx, namespace, project,
			repo.Visibility == gitlab.PublicVisibility,
		)
		for _, sub := range subs {
			if !sub.Merges() {
				continue
			}

			if sub.Label() != "" && !containsLabelPointer(event.Labels, sub.Label()) {
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

func diffAssignees(prev, curr []*gitlab.EventUser) (added, removed []*gitlab.EventUser) {
	prevMap := make(map[int]*gitlab.EventUser)
	currMap := make(map[int]*gitlab.EventUser)

	for _, user := range prev {
		prevMap[user.ID] = user
	}

	for _, user := range curr {
		currMap[user.ID] = user
	}

	for id, user := range prevMap {
		if _, exists := currMap[id]; !exists {
			removed = append(removed, user)
		}
	}

	for id, user := range currMap {
		if _, exists := prevMap[id]; !exists {
			added = append(added, user)
		}
	}

	return
}

func formatAssignees(users []*gitlab.EventUser) string {
	names := []string{}
	for _, user := range users {
		names = append(names, user.Username)
	}
	return strings.Join(names, ", ")
}
