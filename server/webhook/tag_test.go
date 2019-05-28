package webhook

import (
	"encoding/json"
	"testing"

	"github.com/manland/mattermost-plugin-gitlab/server/subscription"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

type testDataTagStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataTag = []testDataTagStr{
	{
		testTitle: "manland create a tag",
		fixture:   SimpleTag,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "tag", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) New tag [tag1](http://localhost:3000/manland/webhook/commit/c30217b62542c586fdbadc7b5ee762bfdca10663) by [manland](http://my.gitlab.com/manland): Really beatiful tag",
			ToUsers:    []string{}, // No DM because user know he has created a tag
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	},
}

func TestTagWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataTag {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			tagEvent := &gitlab.TagEvent{}
			if err := json.Unmarshal([]byte(test.fixture), tagEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, err := w.HandleTag(tagEvent)
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
