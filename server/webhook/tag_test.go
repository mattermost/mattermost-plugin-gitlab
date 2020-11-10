package webhook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
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
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) New tag [tag1](http://localhost:3000/manland/webhook/-/tags/tag1) by [manland](http://my.gitlab.com/manland): Really beautiful tag",
			ToUsers:    []string{}, // No DM because user know he has created a tag
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	},
	{
		testTitle: "manland create a tag (subgroup)",
		fixture:   strings.ReplaceAll(SimpleTag, "manland/webhook", "manland/subgroup/webhook"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "tag", Repository: "manland/subgroup/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland/subgroup/webhook](http://localhost:3000/manland/subgroup/webhook) New tag [tag1](http://localhost:3000/manland/subgroup/webhook/-/tags/tag1) by [manland](http://my.gitlab.com/manland): Really beautiful tag",
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
