package webhook

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

type testDataIssueStr struct {
	testTitle string
	fixture   string
	res       []*HandleWebhook
}

var testDataIssue = []testDataIssueStr{
	{
		testTitle: "root open issue with manland assignee",
		fixture:   NewIssue,
		res: []*HandleWebhook{{
			Message: "[root](http://my.gitlab.com/root) assigned you to issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
			To:      "manland",
			From:    "root",
		}},
	}, {
		testTitle: "manland close issue of root",
		fixture:   CloseIssue,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) closed your issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
			To:      "root",
			From:    "manland",
		}},
	}, {
		testTitle: "manland reopen issue of root",
		fixture:   ReopenIssue,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) reopened your issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
			To:      "root",
			From:    "manland",
		}},
	},
}

func TestIssueWebhook(t *testing.T) {
	t.Parallel()
	w := NewWebhook(fakeWebhook{})
	for _, test := range testDataIssue {
		t.Run(test.testTitle, func(t *testing.T) {
			issueEvent := &gitlab.IssueEvent{}
			json.Unmarshal([]byte(test.fixture), issueEvent)
			res, err := w.HandleIssue(issueEvent)
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
