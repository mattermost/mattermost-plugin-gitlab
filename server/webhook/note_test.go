package webhook

import (
	"encoding/json"
	"testing"

	"github.com/manland/mattermost-plugin-gitlab/server/subscription"

	"github.com/manland/go-gitlab"
	"github.com/stretchr/testify/assert"
)

type testDataNoteStr struct {
	testTitle string
	fixture   string
	kind      string
	res       []*HandleWebhook
}

var testDataNote = []testDataNoteStr{
	{
		testTitle: "manland comment issue of root",
		kind:      "issue",
		fixture:   IssueComment,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) commented on your issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1#note_997)",
			ToUsers: []string{"root"},
			From:    "manland",
		}},
	}, {
		testTitle: "manland comment merge request of root",
		kind:      "mr",
		fixture:   MergeRequestComment,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) commented on your merge request [manland/webhook#6](http://localhost:3000/manland/webhook/merge_requests/6#note_999)",
			ToUsers: []string{"root"},
			From:    "manland",
		}},
	},
}

func TestNoteWebhook(t *testing.T) {
	t.Parallel()
	w := NewWebhook(newFakeWebhook([]*subscription.Subscription{}))
	for _, test := range testDataNote {
		t.Run(test.testTitle, func(t *testing.T) {
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
