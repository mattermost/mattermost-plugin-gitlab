package webhook

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"
)

func (w *webhook) HandleTag(ctx context.Context, event *gitlab.TagEvent) ([]*HandleWebhook, error) {
	handlers, err := w.handleDMTag(event)
	if err != nil {
		return nil, err
	}
	handlers2, err := w.handleChannelTag(ctx, event)
	if err != nil {
		return nil, err
	}
	return cleanWebhookHandlers(append(handlers, handlers2...)), nil
}

func (w *webhook) handleDMTag(event *gitlab.TagEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := event.UserName
	handlers := []*HandleWebhook{}
	tagNames := strings.Split(event.Ref, "/")
	tagName := tagNames[len(tagNames)-1]
	URL := fmt.Sprintf("%s/-/tags/%s", event.Project.WebURL, tagName)

	if mention := w.handleMention(mentionDetails{
		senderUsername:    senderGitlabUsername,
		pathWithNamespace: event.Project.PathWithNamespace,
		IID:               tagName,
		URL:               URL,
		body:              event.Message,
	}); mention != nil {
		handlers = append(handlers, mention)
	}

	return handlers, nil
}

func (w *webhook) handleChannelTag(ctx context.Context, event *gitlab.TagEvent) ([]*HandleWebhook, error) {
	senderGitlabUsername := w.gitlabRetreiver.GetUsernameByID(event.UserID)
	repo := event.Project
	tagNames := strings.Split(event.Ref, "/")
	tagName := tagNames[len(tagNames)-1]
	URL := fmt.Sprintf("%s/-/tags/%s", repo.WebURL, tagName)
	res := []*HandleWebhook{}

	message := fmt.Sprintf("[%s](%s) New tag [%s](%s) by [%s](%s): %s", repo.PathWithNamespace, repo.WebURL, tagName, URL, senderGitlabUsername, w.gitlabRetreiver.GetUserURL(senderGitlabUsername), event.Message)

	toChannels := make([]string, 0)
	namespace, project := normalizeNamespacedProject(repo.PathWithNamespace)
	subs := w.gitlabRetreiver.GetSubscribedChannelsForProject(
		ctx, namespace, strconv.Itoa(event.UserID), project,
		repo.Visibility == gitlab.PublicVisibility,
	)
	for _, sub := range subs {
		if !sub.Tag() {
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
