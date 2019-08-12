package webhook

import (
	"encoding/json"
	"testing"

	"github.com/manland/mattermost-plugin-gitlab/server/subscription"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

type testDataPushStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataPush = []testDataPushStr{	{
		testTitle: "manland push 1 commit",
		fixture:   PushEvent,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "pushes", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) has pushed 1 commit to [manland/webhook](http://localhost:3000/manland/webhook)\n" +
				"- [really cool commit](http://localhost:3000/manland/webhook/commit/c30217b62542c586fdbadc7b5ee762bfdca10663)",
			ToUsers:    []string{}, // No DM because user know he has push commits
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle: "manland push 2 commits",
		fixture:   pushEventWithTwoCommits,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "pushes", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) has pushed 2 commits to [manland/webhook](http://localhost:3000/manland/webhook)\n" +
				"- [really cool commit](http://localhost:3000/manland/webhook/commit/c30217b62542c586fdbadc7b5ee762bfdca10663)\n"+
				"- [another cool commit](http://localhost:3000/manland/webhook/commit/595f2a068cce60954565b224bc7c966c9e708cbf)",
			ToUsers:    []string{}, // No DM because user know he has push commits
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	},
}

func TestPushWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataPush {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			pushEvent := &gitlab.PushEvent{}
			if err := json.Unmarshal([]byte(test.fixture), pushEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, err := w.HandlePush(pushEvent)
			assert.Empty(t, err)
			assert.Equal(t, len(test.res), len(res))
			for index := range res {
				assert.Equal(t, test.res[index].Message, res[index].Message)
				assert.Equal(t, test.res[index].ToUsers, res[index].ToUsers)
				assert.Equal(t, test.res[index].From, res[index].From)
			}
		})
	}
}
