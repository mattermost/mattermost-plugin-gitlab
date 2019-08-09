package webhook

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandlePush(event *gitlab.PushEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMPush(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.handleChannelPush(event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), nil
}

func (w *webhook) handleDMPush(event *gitlab.PushEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.UserName
	handlers := []*HandleWebhook{}

	for _, commit := range event.Commits {
		if mention := w.handleMention(mentionDetails{
			senderUsername:    senderGitlabUsername,
			pathWithNamespace: event.Project.PathWithNamespace,
			IID:               commit.ID,
			URL:               commit.URL,
			body:              commit.Message,
		}); mention != nil {
			handlers = append(handlers, mention)
		}
	}

	return handlers, nil
}

func (w *webhook) handleChannelPush(event *gitlab.PushEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.UserName
	repo := event.Project
	res := []*HandleWebhook{}

	if event.TotalCommitsCount == 0 {
		return nil, nil
	}

	message := fmt.Sprintf("[%s](%s) has pushed %d commit(s) to [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), len(event.Commits), event.Project.PathWithNamespace, event.Project.WebURL)

	toChannels := make([]string, 0)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForRepository(repo.PathWithNamespace, repo.Visibility == gitlab.PublicVisibility)
	for _, sub := range subs {
		if !sub.Pushes() {
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
