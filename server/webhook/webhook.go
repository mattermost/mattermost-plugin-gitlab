package webhook

import (
	"fmt"
	"strings"

	"github.com/manland/mattermost-plugin-gitlab/server/subscription"

	"github.com/manland/go-gitlab"
)

// GitlabRetreiver return infos of current gitlab instance
type GitlabRetreiver interface {
	// GetUserURL return the url of this gitlab user depending on domain instance (e.g. https://gitlab.com/username)
	GetUserURL(username string) string
	// GetUsernameById return a username by gitlab id
	GetUsernameByID(id int) string
	// ParseGitlabUsernamesFromText from a text return an array of username
	ParseGitlabUsernamesFromText(text string) []string
	// GetSubscribedChannelsForRepository return all subscription for this repository
	GetSubscribedChannelsForRepository(repoWithNamespace string, isPublicVisibility bool) []*subscription.Subscription
}

type HandleWebhook struct {
	Message    string
	From       string
	ToUsers    []string
	ToChannels []string
}

type Webhook interface {
	HandleIssue(event *gitlab.IssueEvent) ([]*HandleWebhook, error)
	HandleMergeRequest(event *gitlab.MergeEvent) ([]*HandleWebhook, error)
	HandleIssueComment(event *gitlab.IssueCommentEvent) ([]*HandleWebhook, error)
	HandleMergeRequestComment(event *gitlab.MergeCommentEvent) ([]*HandleWebhook, error)
}

type webhook struct {
	gitlabRetreiver GitlabRetreiver
}

func NewWebhook(g GitlabRetreiver) Webhook {
	return &webhook{gitlabRetreiver: g}
}

func cleanWebhookHandlers(handlers []*HandleWebhook) []*HandleWebhook {
	res := make([]*HandleWebhook, 0)
	for _, handle := range handlers {
		res = append(res, cleanWebhookHandlerTo(handle))
	}
	return res
}

func cleanWebhookHandlerTo(handler *HandleWebhook) *HandleWebhook {
	users := map[string]bool{}
	for _, v := range handler.ToUsers {
		if handler.From != v && v != "" { // don't send message to author or unknown user
			users[v] = true
		}
	}

	cleanedUsers := []string{}
	for key := range users {
		cleanedUsers = append(cleanedUsers, key)
	}

	channels := map[string]bool{}
	for _, v := range handler.ToChannels {
		channels[v] = true
	}

	cleanedChannels := []string{}
	for key := range channels {
		cleanedChannels = append(cleanedChannels, key)
	}

	return &HandleWebhook{
		From:       handler.From,
		Message:    handler.Message,
		ToUsers:    cleanedUsers,
		ToChannels: cleanedChannels,
	}
}

type mentionDetails struct {
	senderUsername    string
	pathWithNamespace string
	IID               int
	URL               string
	body              string
}

func (w *webhook) handleMention(m mentionDetails) *HandleWebhook {
	mentionedUsernames := w.gitlabRetreiver.ParseGitlabUsernamesFromText(m.body)
	if len(mentionedUsernames) > 0 {
		return &HandleWebhook{
			From:    m.senderUsername,
			Message: fmt.Sprintf("[%s](%s) mentioned you on [%s#%v](%s):\n>%s", m.senderUsername, w.gitlabRetreiver.GetUserURL(m.senderUsername), m.pathWithNamespace, m.IID, m.URL, m.body),
			ToUsers: mentionedUsernames,
		}
	}
	return nil
}

func sameLabels(a []gitlab.Label, b []gitlab.Label) bool {
	if len(a) != len(b) {
		return false
	}
	for index, l := range a {
		if l.ID != b[index].ID {
			return false
		}
	}
	return true
}

func containsLabel(a []gitlab.Label, labelName string) bool {
	for _, l := range a {
		if l.Name == labelName {
			return true
		}
	}
	return false
}

func labelToString(a []gitlab.Label) string {
	names := make([]string, len(a))
	for index, l := range a {
		names[index] = l.Name
	}
	return strings.Join(names, ", ")
}
