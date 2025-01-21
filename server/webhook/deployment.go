// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package webhook

import (
	"context"
	"fmt"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandleDeployment(ctx context.Context, event *gitlab.DeploymentEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleChannelDeployment(ctx, event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(handlers), nil
}

func (w *webhook) handleChannelDeployment(ctx context.Context, event *gitlab.DeploymentEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	project := event.Project
	res := []*HandleWebhook{}
	message := fmt.Sprintf("### Deployment Stage: **%s**\n", event.Status)

	switch event.Status {
	case statusRunning:
		message += fmt.Sprintf(":rocket: **Status**: %s\n", event.Status)
	case statusCreated:
		message += fmt.Sprintf(":clock1: **Status**: %s\n", event.Status)
	case statusCanceled:
		message += fmt.Sprintf(":no_entry_sign: **Status**: %s\n", event.Status)
	case statusSuccess:
		message += fmt.Sprintf(":large_green_circle: **Status**: %s\n", event.Status)
	case statusFailed:
		message += fmt.Sprintf(":red_circle: **Status**: %s\n", event.Status)
	default:
		return res, nil
	}

	namespaceMetadata, err := normalizeNamespacedProjectByHomepage(event.Project.Homepage)
	if err != nil {
		return nil, err
	}

	fullNamespacePath := fmt.Sprintf("%s/%s", namespaceMetadata.Namespace, namespaceMetadata.Project)
	message += fmt.Sprintf("**Repository**: [%s](%s)\n", fullNamespacePath, event.Project.GitHTTPURL)
	message += fmt.Sprintf("**Triggered By**: %s\n", senderGitlabUsername)
	message += fmt.Sprintf("**Visit deployment [here](%s)** \n", event.DeployableURL)

	toChannels := make([]string, 0)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
		ctx, namespaceMetadata.Namespace,
		namespaceMetadata.Project,
		project.VisibilityLevel == PublicVisibilityLevel,
	)
	for _, sub := range subs {
		if !sub.Deployments() {
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

	return res, nil
}
