package webhook

import (
	"encoding/json"
	"testing"

	"github.com/manland/go-gitlab"
	"github.com/stretchr/testify/assert"
)

type testDataMergeRequestStr struct {
	testTitle string
	fixture   string
	res       []*HandleWebhook
}

var testDataMergeRequest = []testDataMergeRequestStr{
	{
		testTitle: "root open merge request for manland",
		fixture:   OpenMergeRequest,
		res: []*HandleWebhook{{
			Message: "[root](http://my.gitlab.com/root) requested your review on [manland/webhook#4](http://localhost:3000/manland/webhook/merge_requests/4)",
			To:      "manland",
			From:    "root",
		}},
	}, {
		testTitle: "manland close merge request of root",
		fixture:   CloseMergeRequestByAssignee,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) closed your pull request [manland/webhook#4](http://localhost:3000/manland/webhook/merge_requests/4)",
			To:      "root",
			From:    "manland",
		}},
	}, {
		testTitle: "manland reopen merge request of root",
		fixture:   ReopenMerge,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) reopen your pull request [manland/webhook#1](http://localhost:3000/manland/webhook/merge_requests/1)",
			To:      "", //no assignee
			From:    "manland",
		}, {
			Message: "[manland](http://my.gitlab.com/manland) reopen your pull request [manland/webhook#1](http://localhost:3000/manland/webhook/merge_requests/1)",
			To:      "root",
			From:    "manland",
		}},
	}, {
		testTitle: "root affect manland to merge-request",
		fixture:   AssigneeMergeRequest,
		res: []*HandleWebhook{{
			Message: "[root](http://my.gitlab.com/root) assigned you to pull request [manland/webhook#4](http://localhost:3000/manland/webhook/merge_requests/4)",
			To:      "manland",
			From:    "root",
		}},
	}, {
		testTitle: "manland merge root merge-request",
		fixture:   MergeRequestMerged,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) merged your pull request [manland/webhook#4](http://localhost:3000/manland/webhook/merge_requests/4)",
			To:      "root",
			From:    "manland",
		}},
	},
}

func TestMergeRequestWebhook(t *testing.T) {
	t.Parallel()
	w := NewWebhook(fakeWebhook{})
	for _, test := range testDataMergeRequest {
		t.Run(test.testTitle, func(t *testing.T) {
			mergeEvent := &gitlab.MergeEvent{}
			json.Unmarshal([]byte(test.fixture), mergeEvent)
			res, err := w.HandleMergeRequest(mergeEvent)
			assert.Empty(t, err)
			assert.Equal(t, len(test.res), len(res))
			for index := range res {
				assert.Equal(t, test.res[index].Message, res[index].Message)
				assert.Equal(t, test.res[index].To, res[index].To)
				assert.Equal(t, test.res[index].From, res[index].From)
			}
		})
	}
}
