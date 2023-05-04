package webhook

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"

	"github.com/microcosm-cc/bluemonday"
	"github.com/xanzy/go-gitlab"
)

// GitlabRetreiver return infos of current GitLab instance
type GitlabRetreiver interface {
	// GetPipelineURL return the url of this pipeline depending on the instance and project path
	GetPipelineURL(pathWithNamespace string, pipelineID int) string
	GetJobURL(pathWithNamespace string, jobID int) string
	// GetUserURL return the url of this GitLab user depending on domain3 instance (e.g. https://gitlab.com/username)
	GetUserURL(username string) string
	// GetUsernameById return a username by GitLab id
	GetUsernameByID(id int) string
	// ParseGitlabUsernamesFromText from a text return an array of username
	ParseGitlabUsernamesFromText(text string) []string
	// GetSubscribedChannelsForProject returns all subscriptions for given project.
	GetSubscribedChannelsForProject(ctx context.Context, namespace, project string, isPublicVisibility bool) []*subscription.Subscription
}

type HandleWebhook struct {
	Message    string
	From       string
	ToUsers    []string
	ToChannels []string
}

type Webhook interface {
	HandleIssue(ctx context.Context, event *gitlab.IssueEvent, eventType gitlab.EventType) ([]*HandleWebhook, error)
	HandleMergeRequest(ctx context.Context, event *gitlab.MergeEvent) ([]*HandleWebhook, error)
	HandleIssueComment(ctx context.Context, event *gitlab.IssueCommentEvent) ([]*HandleWebhook, error)
	HandleMergeRequestComment(ctx context.Context, event *gitlab.MergeCommentEvent) ([]*HandleWebhook, error)
	HandlePipeline(ctx context.Context, event *gitlab.PipelineEvent) ([]*HandleWebhook, error)
	HandleTag(ctx context.Context, event *gitlab.TagEvent) ([]*HandleWebhook, error)
	HandlePush(ctx context.Context, event *gitlab.PushEvent) ([]*HandleWebhook, error)
	HandleJobs(ctx context.Context, event *gitlab.JobEvent) ([]*HandleWebhook, error)
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
		if handler.From != v && v != "" { // don't send message to webhook sender or unknown user
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
	IID               string
	URL               string
	body              string
}

func (w *webhook) handleMention(m mentionDetails) *HandleWebhook {
	mentionedUsernames := w.gitlabRetreiver.ParseGitlabUsernamesFromText(m.body)
	if len(mentionedUsernames) > 0 {
		return &HandleWebhook{
			From:    m.senderUsername,
			Message: fmt.Sprintf("[%s](%s) mentioned you on [%s#%s](%s):\n>%s", m.senderUsername, w.gitlabRetreiver.GetUserURL(m.senderUsername), m.pathWithNamespace, m.IID, m.URL, m.body),
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

func containsLabelPointer(a []*gitlab.Label, labelName string) bool {
	for _, l := range a {
		if l != nil && l.Name == labelName {
			return true
		}
	}
	return false
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

// normalizeNamespacedProject converts data from web hooks to format expected by our plugin.
//
// The difference is that this plugin requires separate namespace and project path parts.
// However in web hooks only pathWithNamespace is available.
// In other words,
// group/subgroup/project
// becomes
// namespace = group/subgroup; project = project
func normalizeNamespacedProject(pathWithNamespace string) (namespace string, project string) {
	splits := strings.Split(pathWithNamespace, "/")
	if len(splits) < 2 {
		return "", ""
	}
	return strings.Join(splits[:len(splits)-1], "/"), splits[len(splits)-1]
}

type namespaceProjectMetadata struct {
	Namespace string
	Project   string
}

// normalizeNamespacedProjectByHomepage converts data from web hooks to format expected by our plugin.
func normalizeNamespacedProjectByHomepage(homepage string) (*namespaceProjectMetadata, error) {
	u, err := url.Parse(homepage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse homepage URL")
	}
	splits := strings.Split(u.Path, "/")
	if len(splits) < 2 {
		return nil, errors.New("")
	}

	return &namespaceProjectMetadata{
		Namespace: strings.Join(splits[1:len(splits)-1], "/"),
		Project:   splits[len(splits)-1],
	}, nil
}

func sanitizeDescription(description string) string {
	var policy = bluemonday.StrictPolicy()
	policy.SkipElementsContent("details")
	return strings.TrimSpace(policy.Sanitize(description))
}
