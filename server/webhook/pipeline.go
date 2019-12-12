package webhook

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandlePipeline(event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMPipeline(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.handleChannelPipeline(event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), nil
}

func (w *webhook) handleDMPipeline(event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	repo := event.Project

	handlers := []*HandleWebhook{}

	if event.ObjectAttributes.Status == "failed" {
		message := fmt.Sprintf("[%s](%s) Your pipeline has failed for [%s](%s)", repo.PathWithNamespace, repo.WebURL, event.Commit.Message, event.Commit.URL)
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

func (w *webhook) handleChannelPipeline(event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	repo := event.Project
	res := []*HandleWebhook{}
	message := ""

	switch event.ObjectAttributes.Status {
	case "running":
		message = fmt.Sprintf("[%s](%s) New pipeline by [%s](%s) for [%s](%s)", repo.PathWithNamespace, repo.WebURL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Commit.Message, event.Commit.URL)
	case "success":
		message = fmt.Sprintf("[%s](%s) Pipeline by [%s](%s) success for [%s](%s)", repo.PathWithNamespace, repo.WebURL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Commit.Message, event.Commit.URL)
	case "failed":
		message = fmt.Sprintf("[%s](%s) Pipeline by [%s](%s) fail for [%s](%s)", repo.PathWithNamespace, repo.WebURL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Commit.Message, event.Commit.URL)
	default:
		return res, nil
	}

	toChannels := make([]string, 0)
	namespace, project := normalizeNamespacedProject(repo.PathWithNamespace)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
		namespace, project,
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
