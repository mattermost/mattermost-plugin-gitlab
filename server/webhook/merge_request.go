// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

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
			message = fmt.Sprintf("[%s](%s) requested your review on [#%d](%s) in [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.IID, event.ObjectAttributes.URL, event.ObjectAttributes.Target.PathWithNamespace, event.Repository.Homepage)
			handlers = []*HandleWebhook{{
				Message: message,
				ToUsers: toUsers,
				From:    senderGitlabUsername,
			}}
		case actionReopen:
			message = fmt.Sprintf("[%s](%s) reopened your merge request [#%d](%s) in [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.IID, event.ObjectAttributes.URL, event.ObjectAttributes.Target.PathWithNamespace, event.Repository.Homepage)
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
				updatedCurrentUsers := w.calculateUserDiffs(event.Changes.Assignees.Previous, event.Changes.Assignees.Current)

				if len(updatedCurrentUsers) != 0 {
					message = fmt.Sprintf("[%s](%s) assigned you to merge request [#%d](%s) in [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.IID, event.ObjectAttributes.URL, event.ObjectAttributes.Target.PathWithNamespace, event.Repository.Homepage)
					handlers = append(handlers, &HandleWebhook{
						Message: message,
						ToUsers: updatedCurrentUsers,
						From:    senderGitlabUsername,
					})
				}
			}

			// Handle change in reviewers
			if event.Changes.Reviewers.Current != nil || event.Changes.Reviewers.Previous != nil {
				updatedCurrentUsers := w.calculateUserDiffs(event.Changes.Reviewers.Previous, event.Changes.Reviewers.Current)

				if len(updatedCurrentUsers) != 0 {
					message = fmt.Sprintf("[%s](%s) requested your review on merge request [#%d](%s) in [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.IID, event.ObjectAttributes.URL, event.ObjectAttributes.Target.PathWithNamespace, event.Repository.Homepage)
					handlers = append(handlers, &HandleWebhook{
						Message: message,
						ToUsers: updatedCurrentUsers,
						From:    senderGitlabUsername,
					})
				}
			}
		case actionApproved:
			message = fmt.Sprintf("[%s](%s) approved your merge request [#%d](%s) in [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.IID, event.ObjectAttributes.URL, event.ObjectAttributes.Target.PathWithNamespace, event.Repository.Homepage)
			handlers = []*HandleWebhook{{
				Message: message,
				ToUsers: toUsers,
				From:    senderGitlabUsername,
			}}
		case actionUnapproved:
			message = fmt.Sprintf("[%s](%s) requested changes to your merge request [#%d](%s) in [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.IID, event.ObjectAttributes.URL, event.ObjectAttributes.Target.PathWithNamespace, event.Repository.Homepage)
			handlers = []*HandleWebhook{{
				Message: message,
				ToUsers: toUsers,
				From:    senderGitlabUsername,
			}}
		}
	case stateClosed:
		message = fmt.Sprintf("[%s](%s) closed your merge request [#%d](%s) in [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.IID, event.ObjectAttributes.URL, event.ObjectAttributes.Target.PathWithNamespace, event.Repository.Homepage)
		handlers = []*HandleWebhook{{
			Message: message,
			ToUsers: toUsers,
			From:    senderGitlabUsername,
		}}
	case stateMerged:
		if event.ObjectAttributes.Action == actionMerge {
			message = fmt.Sprintf("[%s](%s) merged your merge request [#%d](%s) in [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.ObjectAttributes.IID, event.ObjectAttributes.URL, event.ObjectAttributes.Target.PathWithNamespace, event.Repository.Homepage)
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

			if len(sub.Labels()) > 0 && !containsAnyLabel(event.Labels, sub.Labels()) {
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

// calculateUserDiffs function takes previousUsers and currentUsers of an event,
// finds the change in the user list, and returns the updated current user list.
func (w *webhook) calculateUserDiffs(previousUsers, currentUsers []*gitlab.EventUser) []string {
	mapPreviousUsers := map[int]*gitlab.EventUser{}
	updatedCurrentUsers := []string{}
	for _, previousUser := range previousUsers {
		mapPreviousUsers[previousUser.ID] = previousUser
	}

	for _, currentUser := range currentUsers {
		if _, exist := mapPreviousUsers[currentUser.ID]; !exist {
			updatedCurrentUsers = append(updatedCurrentUsers, w.gitlabRetreiver.GetUsernameByID(currentUser.ID))
		}
	}

	return updatedCurrentUsers
}
