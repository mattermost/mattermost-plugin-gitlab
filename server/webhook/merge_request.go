package webhook

import (
	"context"
	"fmt"

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

	toUsers := []string{authorGitlabUsername}
	for _, assigneeID := range event.ObjectAttributes.AssigneeIDs {
		toUsers = append(toUsers, w.gitlabRetreiver.GetUsernameByID(assigneeID))
	}

	message := ""
	handlers := []*HandleWebhook{}
	switch event.ObjectAttributes.State {
	case stateOpened:
		switch event.ObjectAttributes.Action {
		case actionOpen:
			message = fmt.Sprintf("[%s](%s) requested your review on [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
			handlers = []*HandleWebhook{{
				Message: message,
				ToUsers: toUsers,
				From:    senderGitlabUsername,
			}}
		case actionReopen:
			message = fmt.Sprintf("[%s](%s) reopened your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
			handlers = []*HandleWebhook{{
				Message: message,
				ToUsers: toUsers,
				From:    senderGitlabUsername,
			}}
		case actionUpdate:
			// Not going to show notification in case of commit push.
			if event.ObjectAttributes.OldRev != "" {
				break
			}

			// Handle change in assignees
			if event.Changes.Assignees.Current != nil || event.Changes.Assignees.Previous != nil {
				isAuthorPresent, updatedPreviousUsers, updatedCurrentUsers := w.handleDMNotificationUsers(event.ObjectAttributes.AuthorID, event.Changes.Assignees.Previous, event.Changes.Assignees.Current)

				// Check if the MR author is present in the assignee changes
				if !isAuthorPresent {
					message = fmt.Sprintf("[%s](%s) updated the list of assignees for the merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
					handlers = append(handlers, &HandleWebhook{
						Message: message,
						ToUsers: []string{authorGitlabUsername},
						From:    senderGitlabUsername,
					})
				}

				if len(updatedPreviousUsers) != 0 {
					message = fmt.Sprintf("[%s](%s) removed you as an assignee from the merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
					handlers = append(handlers, &HandleWebhook{
						Message: message,
						ToUsers: updatedPreviousUsers,
						From:    senderGitlabUsername,
					})
				}

				if len(updatedCurrentUsers) != 0 {
					message = fmt.Sprintf("[%s](%s) assigned you to merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
					handlers = append(handlers, &HandleWebhook{
						Message: message,
						ToUsers: updatedCurrentUsers,
						From:    senderGitlabUsername,
					})
				}
			}

			// Handle change in reviewers
			if event.Changes.Reviewers.Current != nil || event.Changes.Reviewers.Previous != nil {
				isAuthorPresent, updatedPreviousUsers, updatedCurrentUsers := w.handleDMNotificationUsers(event.ObjectAttributes.AuthorID, event.Changes.Reviewers.Previous, event.Changes.Reviewers.Current)

				// Check if the MR author is present in the assignee changes
				if !isAuthorPresent {
					message = fmt.Sprintf("[%s](%s) updated the list of reviewers for the merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
					handlers = append(handlers, &HandleWebhook{
						Message: message,
						ToUsers: []string{authorGitlabUsername},
						From:    senderGitlabUsername,
					})
				}

				if len(updatedPreviousUsers) != 0 {
					message = fmt.Sprintf("[%s](%s) removed you as a reviewer from the merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
					handlers = append(handlers, &HandleWebhook{
						Message: message,
						ToUsers: updatedPreviousUsers,
						From:    senderGitlabUsername,
					})
				}

				if len(updatedCurrentUsers) != 0 {
					message = fmt.Sprintf("[%s](%s) assigned you as a reviewer to merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
					handlers = append(handlers, &HandleWebhook{
						Message: message,
						ToUsers: updatedCurrentUsers,
						From:    senderGitlabUsername,
					})
				}
			}

			// Check if multiple events happened in single webhook event
			if w.checkForMultipleEvents(event) {
				message = fmt.Sprintf("[%s](%s) updated the merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
				handlers = []*HandleWebhook{{
					Message: message,
					ToUsers: toUsers,
					From:    senderGitlabUsername,
				}}
			}

		case actionApproved:
			message = fmt.Sprintf("[%s](%s) approved your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
			handlers = []*HandleWebhook{{
				Message: message,
				ToUsers: toUsers,
				From:    senderGitlabUsername,
			}}
		case actionUnapproved:
			message = fmt.Sprintf("[%s](%s) requested changes to your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
			handlers = []*HandleWebhook{{
				Message: message,
				ToUsers: toUsers,
				From:    senderGitlabUsername,
			}}
		}
	case stateClosed:
		message = fmt.Sprintf("[%s](%s) closed your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
		handlers = []*HandleWebhook{{
			Message: message,
			ToUsers: toUsers,
			From:    senderGitlabUsername,
		}}
	case stateMerged:
		if event.ObjectAttributes.Action == actionMerge {
			message = fmt.Sprintf("[%s](%s) merged your merge request [%s!%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.Target.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
			handlers = []*HandleWebhook{{
				Message: message,
				ToUsers: toUsers,
				From:    senderGitlabUsername,
			}}
		}
	}

	if len(handlers) > 0 {
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

func (w *webhook) handleDMNotificationUsers(authorID int, previousUsers, currentUsers []*gitlab.EventUser) (bool, []string, []string) {
	mapPreviousUsers := map[int]*gitlab.EventUser{}
	mapCurrentUsers := map[int]*gitlab.EventUser{}
	updatedPreviousUsers := []string{}
	updatedCurrentUsers := []string{}
	isAuthorPresent := false
	for _, previousUser := range previousUsers {
		mapPreviousUsers[previousUser.ID] = previousUser
	}

	for _, currentUser := range currentUsers {
		mapCurrentUsers[currentUser.ID] = currentUser
		if _, exist := mapPreviousUsers[currentUser.ID]; !exist {
			if currentUser.ID == authorID {
				isAuthorPresent = true
			}
			updatedCurrentUsers = append(updatedCurrentUsers, w.gitlabRetreiver.GetUsernameByID(currentUser.ID))
		}
	}

	for _, previousUser := range previousUsers {
		if _, exist := mapCurrentUsers[previousUser.ID]; !exist {
			if previousUser.ID == authorID {
				isAuthorPresent = true
			}
			updatedPreviousUsers = append(updatedPreviousUsers, w.gitlabRetreiver.GetUsernameByID(previousUser.ID))
		}
	}

	return isAuthorPresent, updatedPreviousUsers, updatedCurrentUsers
}

func (w *webhook) checkForMultipleEvents(event *gitlab.MergeEvent) bool {
	eventCount := 0
	if event.Changes.Labels.Current != nil || event.Changes.Labels.Previous != nil {
		eventCount++
	}

	if event.Changes.Description.Current != "" || event.Changes.Description.Previous != "" {
		eventCount++
	}

	if event.Changes.Title.Current != "" || event.Changes.Title.Previous != "" {
		eventCount++
	}

	if event.Changes.MilestoneID.Current != 0 || event.Changes.MilestoneID.Previous != 0 {
		eventCount++
	}

	if event.Changes.Draft.Current || event.Changes.Draft.Previous {
		eventCount++
	}

	if event.Changes.TargetBranch.Current != "" || event.Changes.TargetBranch.Previous != "" {
		eventCount++
	}

	if event.Changes.SourceBranch.Current != "" || event.Changes.SourceBranch.Previous != "" {
		eventCount++
	}

	if (event.Changes.Assignees.Previous != nil || event.Changes.Assignees.Current != nil) && (event.Changes.Reviewers.Previous != nil || event.Changes.Reviewers.Current != nil) {
		eventCount++
	}

	return eventCount > 0
}
