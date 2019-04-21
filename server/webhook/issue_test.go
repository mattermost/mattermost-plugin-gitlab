package webhook

import (
	"encoding/json"
	"testing"

	"github.com/manland/go-gitlab"
	"github.com/manland/mattermost-plugin-gitlab/server/subscription"
	"github.com/stretchr/testify/assert"
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
			ToUsers: []string{"manland"},
			From:    "root",
		}},
	}, {
		testTitle: "root open unassigned issue",
		fixture:   NewIssueUnassigned,
		res:       []*HandleWebhook{}, // no message because root don't received its own action and manland is not assigned
	}, {
		testTitle: "manland close issue of root",
		fixture:   CloseIssue,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) closed your issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
			ToUsers: []string{"root"},
			From:    "manland",
		}},
	}, {
		testTitle: "manland reopen issue of root",
		fixture:   ReopenIssue,
		res: []*HandleWebhook{{
			Message: "[manland](http://my.gitlab.com/manland) reopened your issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
			ToUsers: []string{"root"},
			From:    "manland",
		}},
	},
}

func TestIssueWebhook(t *testing.T) {
	t.Parallel()
	w := NewWebhook(newFakeWebhook([]*subscription.Subscription{}))
	for _, test := range testDataIssue {
		t.Run(test.testTitle, func(t *testing.T) {
			issueEvent := &gitlab.IssueEvent{}
			if err := json.Unmarshal([]byte(test.fixture), issueEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, err := w.HandleIssue(issueEvent)
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
