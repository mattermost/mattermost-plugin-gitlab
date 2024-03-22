package webhook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

type testDataIssueStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataIssue = []testDataIssueStr{
	{
		testTitle: "root open issue with manland assignee and display in channel1",
		fixture:   NewIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) assigned you to issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}, {
			Message:    "#### test new issue\n##### [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)\n###### new issue by [root](http://my.gitlab.com/root) on [2019-04-06 21:03:04 UTC](http://localhost:3000/manland/webhook/issues/1)\n\nhello world!",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
	}, {
		testTitle: "root open issue with manland assignee and display in channel1 (subgroup)",
		fixture:   strings.ReplaceAll(NewIssue, "manland/webhook", "manland/subgroup/webhook"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/subgroup/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) assigned you to issue [manland/subgroup/webhook#1](http://localhost:3000/manland/subgroup/webhook/issues/1)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}, {
			Message:    "#### test new issue\n##### [manland/subgroup/webhook#1](http://localhost:3000/manland/subgroup/webhook/issues/1)\n###### new issue by [root](http://my.gitlab.com/root) on [2019-04-06 21:03:04 UTC](http://localhost:3000/manland/subgroup/webhook/issues/1)\n\nhello world!",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
	}, {
		testTitle: "root open unassigned issue and display in channel",
		fixture:   NewIssueUnassigned,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "#### new issue\n##### [manland/webhook#2](http://localhost:3000/manland/webhook/issues/2)\n###### new issue by [root](http://my.gitlab.com/root) on [2019-04-06 21:13:03 UTC](http://localhost:3000/manland/webhook/issues/2)\n\nHello world",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}}, // no DM message because root don't received its own action and manland is not assigned
	}, {
		testTitle: "manland close issue of root",
		fixture:   CloseIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) closed your issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Issue [test new issue](http://localhost:3000/manland/webhook/issues/1) closed by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle: "manland reopen issue of root and display in channel",
		fixture:   ReopenIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) reopened your issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
			ToUsers: []string{"root"},
			From:    "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Issue [test new issue](http://localhost:3000/manland/webhook/issues/1) reopened by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	},
}

func TestIssueWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataIssue {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			issueEvent := &gitlab.IssueEvent{}
			if err := json.Unmarshal([]byte(test.fixture), issueEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, err := w.HandleIssue(context.Background(), issueEvent, gitlab.EventTypeIssue)
			assert.Empty(t, err)
			assert.Equal(t, len(test.res), len(res))
			for index := range res {
				assert.Equal(t, test.res[index].Message, res[index].Message)
				assert.EqualValues(t, test.res[index].ToUsers, res[index].ToUsers)
				assert.Equal(t, test.res[index].From, res[index].From)
			}
		})
	}
}
