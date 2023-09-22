package webhook

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandlePipeline(ctx context.Context, event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMPipeline(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.handleChannelPipeline(ctx, event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), nil
}

func (w *webhook) handleDMPipeline(event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	repo := event.Project

	handlers := []*HandleWebhook{}

	if event.ObjectAttributes.Status == statusFailed {
		message := fmt.Sprintf("[%s](%s) Your pipeline has failed for %s [%s](%s)", repo.PathWithNamespace, repo.WebURL, event.Commit.Message, "View Pipeline", w.gitlabRetreiver.GetPipelineURL(repo.PathWithNamespace, event.ObjectAttributes.ID))
		handlers = append(handlers, &HandleWebhook{
			Message:    message,
			From:       "", // don't put senderGitlabUsername because we filter message where from == to
			ToUsers:    []string{senderGitlabUsername},
			ToChannels: []string{},
		})
	}

	if mention := w.handleMention(mentionDetails{
		senderUsername:    senderGitlabUsername,
		pathWithNamespace: event.Project.PathWithNamespace,
		IID:               fmt.Sprintf("%d", event.ObjectAttributes.ID),
		URL:               event.Commit.URL,
		body:              event.Commit.Message,
	}); mention != nil {
		handlers = append(handlers, mention)
	}

	return handlers, nil
}

func (w *webhook) handleChannelPipeline(ctx context.Context, event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	repo := event.Project
	res := []*HandleWebhook{}
	message := ""

	switch event.ObjectAttributes.Status {
	case statusRunning:
		message = fmt.Sprintf("[%s](%s) New pipeline from %s by [%s](%s) for %s [%s](%s)", repo.PathWithNamespace, repo.WebURL, event.ObjectAttributes.Source, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Commit.Message, "View Pipeline", w.gitlabRetreiver.GetPipelineURL(repo.PathWithNamespace, event.ObjectAttributes.ID))
	case statusSuccess:
		message = fmt.Sprintf("[%s](%s) Pipeline by [%s](%s) success for %s [%s](%s)", repo.PathWithNamespace, repo.WebURL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Commit.Message, "View Pipeline", w.gitlabRetreiver.GetPipelineURL(repo.PathWithNamespace, event.ObjectAttributes.ID))
	case statusFailed:
		message = fmt.Sprintf("[%s](%s) Pipeline by [%s](%s) fail for %s [%s](%s)", repo.PathWithNamespace, repo.WebURL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Commit.Message, "View Pipeline", w.gitlabRetreiver.GetPipelineURL(repo.PathWithNamespace, event.ObjectAttributes.ID))
	default:
		return res, nil
	}

	toChannels := make([]string, 0)
	namespace, project := normalizeNamespacedProject(repo.PathWithNamespace)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
		ctx, namespace, strconv.Itoa(event.User.ID), project,
		repo.Visibility == gitlab.PublicVisibility,
	)
	for _, sub := range subs {
		if !sub.Pipeline() {
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
