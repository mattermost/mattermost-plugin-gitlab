package webhook

import (
	"fmt"

	"github.com/manland/go-gitlab"
)

func (w *webhook) HandlePipeline(event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMPipeline(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.HandleChannelPipeline(event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), nil
}

func (w *webhook) handleDMPipeline(event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username

	handlers := []*HandleWebhook{}

	//TODO add failed pipeline to author

	if mention := w.handleMention(mentionDetails{
		senderUsername:    senderGitlabUsername,
		pathWithNamespace: event.Project.PathWithNamespace,
		IID:               event.ObjectAttributes.ID,
		URL:               event.Commit.URL,
		body:              event.Commit.Message,
	}); mention != nil {
		handlers = append(handlers, mention)
	}

	return handlers, nil
}

func (w *webhook) HandleChannelPipeline(event *gitlab.PipelineEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.User.Username
	repo := event.Project
	res := []*HandleWebhook{}

	message := fmt.Sprintf("[%s](%s) New pipeline by [%s](%s) for [%s](%s)", repo.PathWithNamespace, repo.WebURL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Commit.Message, event.Commit.URL)

	toChannels := make([]string, 0)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForRepository(repo.PathWithNamespace, repo.Visibility == gitlab.PublicVisibility)
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
