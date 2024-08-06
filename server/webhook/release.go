package webhook

import (
	"context"
	"fmt"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandleRelease(ctx context.Context, event *gitlab.ReleaseEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleChannelRelease(ctx, event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(handlers), nil
}

func (w *webhook) handleChannelRelease(ctx context.Context, event *gitlab.ReleaseEvent) ([]*HandleWebhook, error) {
	release := event.Project
	res := []*HandleWebhook{}
	message := fmt.Sprintf("### Release: **%s**\n", event.Action)

	switch event.Action {
	case statusCreate:
		message += fmt.Sprintf(":new: **Status**: %s\n", event.Action)
	case statusUpdate:
		message += fmt.Sprintf(":arrows_counterclockwise: **Status**: %s\n", event.Action)
	case statusDelete:
		message += fmt.Sprintf(":red_circle: **Status**: %s\n", event.Action)
	default:
		return res, nil
	}

	namespaceMetadata, err := normalizeNamespacedProjectByHomepage(event.Project.Homepage)
	if err != nil {
		return nil, err
	}

	fullNamespacePath := fmt.Sprintf("%s/%s", namespaceMetadata.Namespace, namespaceMetadata.Project)
	message += fmt.Sprintf("**Repository**: [%s](%s)\n", fullNamespacePath, event.Project.GitHTTPURL)
	if event.Action != statusDelete {
		message += fmt.Sprintf("**Release**: [%s](%s)\n", event.Name, event.URL)
	} else {
		message += fmt.Sprintf("**Release**: %s\n", event.Name)
	}

	toChannels := make([]string, 0)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
		ctx, namespaceMetadata.Namespace,
		namespaceMetadata.Project,
		release.VisibilityLevel == PublicVisibilityLevel,
	)
	for _, sub := range subs {
		if !sub.Releases() {
			continue
		}

		toChannels = append(toChannels, sub.ChannelID)
	}

	if len(toChannels) > 0 {
		res = append(res, &HandleWebhook{
			From:       "",
			Message:    message,
			ToUsers:    []string{},
			ToChannels: toChannels,
		})
	}

	return res, nil
}
