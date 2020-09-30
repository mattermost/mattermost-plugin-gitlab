package webhook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

type testDataNoteStr struct {
	testTitle       string
	fixture         string
	kind            string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataNote = []testDataNoteStr{
	{
		testTitle: "manland comment issue of root",
		kind:      "issue",
		fixture:   IssueComment,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issue_comments", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) commented on your issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1#note_997)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) New comment by [manland](http://my.gitlab.com/manland) on [#1 test new issue](http://localhost:3000/manland/webhook/issues/1#note_997):\n\ncoucou3",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle: "manland comment issue of root (subgroup)",
		kind:      "issue",
		fixture:   strings.ReplaceAll(IssueComment, "manland/webhook", "manland/subgroup/webhook"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issue_comments", Repository: "manland/subgroup/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) commented on your issue [manland/subgroup/webhook#1](http://localhost:3000/manland/subgroup/webhook/issues/1#note_997)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/subgroup/webhook](http://localhost:3000/manland/subgroup/webhook) New comment by [manland](http://my.gitlab.com/manland) on [#1 test new issue](http://localhost:3000/manland/subgroup/webhook/issues/1#note_997):\n\ncoucou3",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle: "manland comment merge request of root",
		kind:      "mr",
		fixture:   MergeRequestComment,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merge_request_comments", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) commented on your merge request [manland/webhook#6](http://localhost:3000/manland/webhook/merge_requests/6#note_999)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) New comment by [manland](http://my.gitlab.com/manland) on [#6 Update README.md](http://localhost:3000/manland/webhook/merge_requests/6#note_999):\n\ncoucou",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	},
}

func TestNoteWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataNote {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			var res []*HandleWebhook
			var err error
			if test.kind == "issue" {
				issueCommentEvent := &gitlab.IssueCommentEvent{}
				if err = json.Unmarshal([]byte(test.fixture), issueCommentEvent); err != nil {
					assert.Fail(t, "can't unmarshal fixture")
				}
				res, err = w.HandleIssueComment(issueCommentEvent)
			} else {
				mergeCommentEvent := &gitlab.MergeCommentEvent{}
				if err = json.Unmarshal([]byte(test.fixture), mergeCommentEvent); err != nil {
					assert.Fail(t, "can't unmarshal fixture")
				}
				res, err = w.HandleMergeRequestComment(mergeCommentEvent)
			}
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
