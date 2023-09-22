package webhook

import (
	"context"
	"fmt"
	"strconv"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandlePush(ctx context.Context, event *gitlab.PushEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMPush(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.handleChannelPush(ctx, event)
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

func (w *webhook) handleChannelPush(ctx context.Context, event *gitlab.PushEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.UserUsername
	repo := event.Project
	res := []*HandleWebhook{}

	if event.TotalCommitsCount == 0 {
		return nil, nil
	}

	plural := "commits"

	if event.TotalCommitsCount == 1 {
		plural = "commit"
	}

	message := fmt.Sprintf("[%s](%s) has pushed %d %s to [%s](%s)", senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.TotalCommitsCount, plural, event.Project.PathWithNamespace, event.Project.WebURL)

	for _, commit := range event.Commits {
		message += fmt.Sprintf("\n%s [%s](%s)", commit.Message, "View Commit", commit.URL)
	}

	toChannels := make([]string, 0)
	namespace, project := normalizeNamespacedProject(repo.PathWithNamespace)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
		ctx, namespace, strconv.Itoa(event.UserID), project,
		repo.Visibility == gitlab.PublicVisibility,
	)
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
