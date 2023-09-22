package webhook

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandleJobs(ctx context.Context, event *gitlab.JobEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleChannelJob(ctx, event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(handlers), nil
}

func (w *webhook) handleChannelJob(ctx context.Context, event *gitlab.JobEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Name
	repo := event.Repository
	res := []*HandleWebhook{}
	message := ""
	message = fmt.Sprintf("### Pipeline Job Stage: **%s**\n", event.BuildStage)

	switch event.BuildStatus {
	case statusRunning:
		message += fmt.Sprintf(":rocket: **Status**: %s\n", event.BuildStatus)
	case statusPending:
		message = fmt.Sprintf("### Pipeline Job Stage: **%s**\n", event.BuildStage)
		message += fmt.Sprintf(":clock1: **Status**: %s\n", event.BuildStatus)
	case statusSuccess:
		message = fmt.Sprintf("### Pipeline Job Stage: **%s**\n", event.BuildStage)
		message += fmt.Sprintf(":large_green_circle: **Status**: %s\n", event.BuildStatus)
	case statusFailed:
		message = fmt.Sprintf("### Pipeline Job Stage: **%s**\n", event.BuildStage)
		message += fmt.Sprintf(":red_circle: **Status**: %s\n", event.BuildStatus)
		message += fmt.Sprintf("**Reason Failed**: %s\n", event.BuildFailureReason)
	default:
		return res, nil
	}
	namespaceMetadata, err := normalizeNamespacedProjectByHomepage(event.Repository.Homepage)
	if err != nil {
		return nil, err
	}
	fullNamespacePath := fmt.Sprintf("%s/%s", namespaceMetadata.Namespace, namespaceMetadata.Project)
	message += fmt.Sprintf("**Repository**: [%s](%s)\n", fullNamespacePath, event.Repository.GitHTTPURL)
	message += fmt.Sprintf("**Triggered By**: %s\n", senderGitlabUsername)
	message += fmt.Sprintf("**Visit job [here](%s)** \n", w.gitlabRetreiver.GetJobURL(fullNamespacePath, event.BuildID))
	toChannels := make([]string, 0)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
		ctx, namespaceMetadata.Namespace, strconv.Itoa(event.User.ID), namespaceMetadata.Project,
		repo.Visibility == gitlab.PublicVisibility,
	)
	for _, sub := range subs {
		if !sub.Jobs() {
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
