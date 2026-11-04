// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package webhook

import (
	"context"
	"fmt"

	"github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

func (w *webhook) HandleIssue(ctx context.Context, event *gitlab.IssueEvent, eventType gitlab.EventType) ([]*HandleWebhook, []string, error) {
	var warnings []string
	handlers, err := w.handleDMIssue(event)
	if err != nil {
		return nil, warnings, err
	}
	handlers2, warnings, err := w.handleChannelIssue(ctx, event, eventType)
	if err != nil {
		return nil, warnings, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), warnings, nil
}

func (w *webhook) handleDMIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error) {
	authorGitlabUsername := w.gitlabRetreiver.GetUsernameByID(event.ObjectAttributes.AuthorID)
	senderGitlabUsername := event.User.Username

	message := ""
	handlers := []*HandleWebhook{}

	switch event.ObjectAttributes.Action {
	case actionOpen:
		if event.Assignees != nil && len(*event.Assignees) > 0 {
			message = fmt.Sprintf("[%s](%s) assigned you to issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
		}
	case actionClose:
		message = fmt.Sprintf("[%s](%s) closed your issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	case actionReopen:
		message = fmt.Sprintf("[%s](%s) reopened your issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL)
	case actionUpdate:
		if event.Changes.Assignees.Current != nil || event.Changes.Assignees.Previous != nil {
			newlyAssigned := w.calculateUserDiffs(event.Changes.Assignees.Previous, event.Changes.Assignees.Current)
			newlyUnassigned := w.calculateUserDiffs(event.Changes.Assignees.Current, event.Changes.Assignees.Previous)

			if len(newlyAssigned) != 0 {
				handlers = append(handlers, &HandleWebhook{
					Message: fmt.Sprintf("[%s](%s) assigned you to issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL),
					ToUsers: newlyAssigned,
					From:    senderGitlabUsername,
				})
			}

			if len(newlyUnassigned) != 0 {
				handlers = append(handlers, &HandleWebhook{
					Message: fmt.Sprintf("[%s](%s) unassigned you from issue [%s#%v](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Project.PathWithNamespace, event.ObjectAttributes.IID, event.ObjectAttributes.URL),
					ToUsers: newlyUnassigned,
					From:    senderGitlabUsername,
				})
			}
		}
	}

	if message != "" {
		toUsers := []string{}
		if event.Assignees != nil {
			for _, assignee := range *event.Assignees {
				toUsers = append(toUsers, assignee.Username)
			}
		}
		toUsers = append(toUsers, authorGitlabUsername)

		handlers = append(handlers, &HandleWebhook{
			Message: message,
			ToUsers: toUsers,
			From:    senderGitlabUsername,
		})

		// Only parse mentions for actions that change the description, otherwise
		// every assignee update would re-notify everyone mentioned in the issue.
		if mention := w.handleMention(mentionDetails{
			senderUsername:    senderGitlabUsername,
			pathWithNamespace: event.Project.PathWithNamespace,
			IID:               fmt.Sprintf("%d", event.ObjectAttributes.IID),
			URL:               event.ObjectAttributes.URL,
			body:              sanitizeDescription(event.ObjectAttributes.Description),
		}); mention != nil {
			handlers = append(handlers, mention)
		}
	}

	return handlers, nil
}

func (w *webhook) handleChannelIssue(ctx context.Context, event *gitlab.IssueEvent, eventType gitlab.EventType) ([]*HandleWebhook, []string, error) {
	issue := event.ObjectAttributes
	senderGitlabUsername := event.User.Username
	repo := event.Project
	res := []*HandleWebhook{}

	message := ""
	var assignMessages []string
	var warnings []string

	switch issue.Action {
	case actionOpen:
		message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n###### new issue by [%s](%s) on [%s](%s)\n\n%s", issue.Title, repo.PathWithNamespace, issue.IID, issue.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), issue.CreatedAt, issue.URL, sanitizeDescription(issue.Description))
	case actionClose:
		message = fmt.Sprintf("[%s](%s) Issue [%s](%s) closed by [%s](%s)", repo.PathWithNamespace, repo.WebURL, issue.Title, issue.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case actionReopen:
		message = fmt.Sprintf("[%s](%s) Issue [%s](%s) reopened by [%s](%s)", repo.PathWithNamespace, repo.WebURL, issue.Title, issue.URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername))
	case actionUpdate:
		if len(event.Changes.Labels.Current) > 0 && !sameLabels(event.Changes.Labels.Current, event.Changes.Labels.Previous) {
			message = fmt.Sprintf("#### %s\n##### [%s#%v](%s)\n###### issue labeled `%s` by [%s](%s) on [%s](%s)\n\n%s", issue.Title, repo.PathWithNamespace, issue.IID, issue.URL, labelToString(event.Changes.Labels.Current), event.User.Username, w.gitlabRetreiver.GetUserURL(event.User.Username), issue.UpdatedAt, issue.URL, sanitizeDescription(issue.Description))
		}

		if event.Changes.Assignees.Current != nil || event.Changes.Assignees.Previous != nil {
			newlyAssigned := w.calculateUserDiffs(event.Changes.Assignees.Previous, event.Changes.Assignees.Current)
			newlyUnassigned := w.calculateUserDiffs(event.Changes.Assignees.Current, event.Changes.Assignees.Previous)

			for _, username := range newlyAssigned {
				assignMessages = append(assignMessages, fmt.Sprintf("[%s](%s) Issue [%s](%s) was assigned to [%s](%s) by [%s](%s)",
					repo.PathWithNamespace, repo.WebURL, issue.Title, issue.URL,
					username, w.gitlabRetreiver.GetUserURL(username),
					senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername)))
			}
			for _, username := range newlyUnassigned {
				assignMessages = append(assignMessages, fmt.Sprintf("[%s](%s) Issue [%s](%s) was unassigned from [%s](%s) by [%s](%s)",
					repo.PathWithNamespace, repo.WebURL, issue.Title, issue.URL,
					username, w.gitlabRetreiver.GetUserURL(username),
					senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername)))
			}
		}
	}

	if len(message) == 0 && len(assignMessages) == 0 {
		return res, warnings, nil
	}

	// Trust the payload's confidential flag in addition to the event type, so a
	// confidential issue delivered under the regular Issue Hook is still gated.
	isConfidential := issue.Confidential || eventType == gitlab.EventConfidentialIssue
	confidentialAllowed := func(sub *subscription.Subscription) bool {
		return !isConfidential || sub.ConfidentialIssues()
	}

	namespace, project := normalizeNamespacedProject(repo.PathWithNamespace)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
		ctx, namespace, project,
		repo.Visibility == gitlab.PublicVisibility,
		isConfidential,
	)

	if len(message) > 0 {
		toChannels, ws := filterChannelsByFeature(subs, event.Labels, func(sub *subscription.Subscription) bool {
			return sub.Issues() && confidentialAllowed(sub)
		})
		warnings = appendUniqueWarnings(warnings, ws...)
		if len(toChannels) > 0 {
			res = append(res, &HandleWebhook{
				From:       senderGitlabUsername,
				Message:    message,
				ToUsers:    []string{},
				ToChannels: toChannels,
			})
		}
	}

	if len(assignMessages) > 0 {
		toChannels, ws := filterChannelsByFeature(subs, event.Labels, func(sub *subscription.Subscription) bool {
			return (sub.Issues() || sub.IssueAssigns()) && confidentialAllowed(sub)
		})
		warnings = appendUniqueWarnings(warnings, ws...)
		if len(toChannels) > 0 {
			for _, msg := range assignMessages {
				res = append(res, &HandleWebhook{
					From:       senderGitlabUsername,
					Message:    msg,
					ToUsers:    []string{},
					ToChannels: toChannels,
				})
			}
		}
	}

	return res, warnings, nil
}
