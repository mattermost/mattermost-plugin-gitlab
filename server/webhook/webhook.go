package webhook

import (
	"fmt"

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
}

type HandleWebhook struct {
	Message string
	To      string
	From    string
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
		if handle.From != handle.To {
			res = append(res, handle)
		}
	}
	return res
}

type mentionDetails struct {
	senderUsername    string
	pathWithNamespace string
	IID               int
	URL               string
	body              string
}

func (w *webhook) handleMention(m mentionDetails) []*HandleWebhook {
	mentionedUsernames := w.gitlabRetreiver.ParseGitlabUsernamesFromText(m.body)
	handlers := make([]*HandleWebhook, len(mentionedUsernames))
	for index, mentionedUsername := range mentionedUsernames {
		handlers[index] = &HandleWebhook{
			Message: fmt.Sprintf("[%s](%s) mentioned you on [%s#%v](%s):\n>%s", m.senderUsername, w.gitlabRetreiver.GetUserURL(m.senderUsername), m.pathWithNamespace, m.IID, m.URL, m.body),
			From:    m.senderUsername,
			To:      mentionedUsername,
		}
	}
	return handlers
}
