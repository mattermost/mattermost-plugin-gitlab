package webhook

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

type fakeWebhook struct {
	subs []*subscription.Subscription
}

func newFakeWebhook(subs []*subscription.Subscription) *fakeWebhook {
	return &fakeWebhook{
		subs: subs,
	}
}

func (*fakeWebhook) GetPipelineURL(pathWithNamespace string, pipelineID int) string {
	return fmt.Sprintf("http://my.gitlab.com/%s/-/pipelines/%d", pathWithNamespace, pipelineID)
}

func (*fakeWebhook) GetUserURL(username string) string {
	return fmt.Sprintf("http://my.gitlab.com/%s", username)
}

func (*fakeWebhook) GetUsernameByID(id int) string {
	switch id {
	case 1:
		return "root"
	case 50:
		return "manland"
	default:
		return ""
	}
}

func (*fakeWebhook) ParseGitlabUsernamesFromText(body string) []string {
	return []string{}
}

func (f *fakeWebhook) GetSubscribedChannelsForProject(namespace, project string, isPublicVisibility bool) []*subscription.Subscription {
	return f.subs
}

type testDataNormalizeNamespacedProjectStr struct {
	Title                  string
	InputNamespace         string
	InputPathWithNamespace string
	ExpectedNamespace      string
	ExpectedProject        string
}

var testDataNormalizeNamespacedProject = []testDataNormalizeNamespacedProjectStr{
	{
		Title:                  "project in group",
		InputPathWithNamespace: "group/project",
		ExpectedNamespace:      "group",
		ExpectedProject:        "project",
	},
	{
		Title:                  "project in subgroup",
		InputPathWithNamespace: "group/subgroup/project",
		ExpectedNamespace:      "group/subgroup",
		ExpectedProject:        "project",
	},
}

func TestNormalizeNamespacedProject(t *testing.T) {
	t.Parallel()
	for _, test := range testDataNormalizeNamespacedProject {
		t.Run(test.Title, func(t *testing.T) {
			namespace, project := normalizeNamespacedProject(test.InputPathWithNamespace)
			assert.Equal(t, test.ExpectedNamespace, namespace)
			assert.Equal(t, test.ExpectedProject, project)
		})
	}
}
