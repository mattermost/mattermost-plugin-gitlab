package webhook

import (
	"fmt"

	"github.com/manland/mattermost-plugin-gitlab/server/subscription"
)

type fakeWebhook struct {
	subs []*subscription.Subscription
}

func newFakeWebhook(subs []*subscription.Subscription) *fakeWebhook {
	return &fakeWebhook{
		subs: subs,
	}
}

func (*fakeWebhook) GetUserURL(username string) string {
	return fmt.Sprintf("http://my.gitlab.com/%s", username)
}

func (*fakeWebhook) GetUsernameByID(id int) string {
	if id == 1 {
		return "root"
	} else if id == 50 {
		return "manland"
	} else {
		return ""
	}
}

func (*fakeWebhook) ParseGitlabUsernamesFromText(body string) []string {
	return []string{}
}

func (f *fakeWebhook) GetSubscribedChannelsForProject(namespace, project string, isPublicVisibility bool) []*subscription.Subscription {
	return f.subs
}
