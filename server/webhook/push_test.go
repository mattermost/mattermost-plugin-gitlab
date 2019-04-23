package webhook

import (
	"encoding/json"
	"testing"

	"github.com/manland/go-gitlab"
	"github.com/manland/mattermost-plugin-gitlab/server/subscription"
	"github.com/stretchr/testify/assert"
)

type testDataPushStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataPush = []testDataPushStr{
	{
		testTitle:       "manland push 1 commit",
		fixture:         PushEvent,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{subscription.New("channel1", "1", "pushes", "manland/webhook")}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) has pushed 1 commit(s) to [manland/webhook](http://localhost:3000/manland/webhook)",
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
