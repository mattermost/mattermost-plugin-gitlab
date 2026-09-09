// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package webhook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

type testDataIssueStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
	warnings        []string
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
		warnings: []string{},
	},
	{
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
		warnings: []string{},
	},
	{
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
		warnings: []string{},
	},
	{
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
		warnings: []string{},
	},
	{
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
		warnings: []string{},
	},
	{
		testTitle:       "root assign manland to issue (DM only, no channel subscription)",
		fixture:         AssignIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{}),
		res: []*HandleWebhook{
			{
				Message:    "[root](http://my.gitlab.com/root) assigned you to issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			},
			{
				Message:    "[root](http://my.gitlab.com/root) unassigned you from issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"user"},
				ToChannels: []string{},
				From:       "root",
			},
		},
		warnings: []string{},
	},
	{
		testTitle:       "root unassign manland from issue (DM only)",
		fixture:         UnassignIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{}),
		res: []*HandleWebhook{
			{
				Message:    "[root](http://my.gitlab.com/root) unassigned you from issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			},
		},
		warnings: []string{},
	},
	{
		testTitle: "root assign manland to issue and display in channel1 with issues subscription",
		fixture:   AssignIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{
			{
				Message:    "[root](http://my.gitlab.com/root) assigned you to issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			},
			{
				Message:    "[root](http://my.gitlab.com/root) unassigned you from issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"user"},
				ToChannels: []string{},
				From:       "root",
			},
			{
				Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Issue [test new issue](http://localhost:3000/manland/webhook/issues/1) was assigned to [manland](http://my.gitlab.com/manland) by [root](http://my.gitlab.com/root)",
				ToUsers:    []string{},
				ToChannels: []string{"channel1"},
				From:       "root",
			},
			{
				Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Issue [test new issue](http://localhost:3000/manland/webhook/issues/1) was unassigned from [user](http://my.gitlab.com/user) by [root](http://my.gitlab.com/root)",
				ToUsers:    []string{},
				ToChannels: []string{"channel1"},
				From:       "root",
			},
		},
		warnings: []string{},
	},
	{
		testTitle: "root assign manland to issue and display in channel1 with issue_assigns subscription",
		fixture:   AssignIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issue_assigns", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{
			{
				Message:    "[root](http://my.gitlab.com/root) assigned you to issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			},
			{
				Message:    "[root](http://my.gitlab.com/root) unassigned you from issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"user"},
				ToChannels: []string{},
				From:       "root",
			},
			{
				Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Issue [test new issue](http://localhost:3000/manland/webhook/issues/1) was assigned to [manland](http://my.gitlab.com/manland) by [root](http://my.gitlab.com/root)",
				ToUsers:    []string{},
				ToChannels: []string{"channel1"},
				From:       "root",
			},
			{
				Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Issue [test new issue](http://localhost:3000/manland/webhook/issues/1) was unassigned from [user](http://my.gitlab.com/user) by [root](http://my.gitlab.com/root)",
				ToUsers:    []string{},
				ToChannels: []string{"channel1"},
				From:       "root",
			},
		},
		warnings: []string{},
	},
	{
		testTitle: "root assign manland to issue but no channel notification without matching subscription",
		fixture:   AssignIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{
			{
				Message:    "[root](http://my.gitlab.com/root) assigned you to issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			},
			{
				Message:    "[root](http://my.gitlab.com/root) unassigned you from issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"user"},
				ToChannels: []string{},
				From:       "root",
			},
		},
		warnings: []string{},
	},
	{
		testTitle: "manland reopen issue of root and display in channel with subscription label warning",
		fixture:   ReopenIssue,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues,label:1", Repository: "manland/webhook"},
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
		warnings: []string{"each label must be wrapped in quotes, e.g. label:\"bug\""},
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
			res, warnings, err := w.HandleIssue(context.Background(), issueEvent, gitlab.EventTypeIssue)
			assert.Empty(t, err)
			assert.Equal(t, len(test.res), len(res))
			assert.ElementsMatch(t, test.warnings, warnings)
			for index := range res {
				assert.Equal(t, test.res[index].Message, res[index].Message)
				assert.EqualValues(t, test.res[index].ToUsers, res[index].ToUsers)
				assert.ElementsMatch(t, test.res[index].ToChannels, res[index].ToChannels)
				assert.Equal(t, test.res[index].From, res[index].From)
			}
		})
	}
}

func TestConfidentialIssueWebhook(t *testing.T) {
	t.Parallel()
	testCases := []testDataIssueStr{
		{
			testTitle: "confidential issue on public project with confidential_issues subscription",
			fixture:   NewConfidentialIssue,
			gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
				{ChannelID: "channel1", CreatorID: "1", Features: "issues,confidential_issues", Repository: "manland/webhook"},
			}),
			res: []*HandleWebhook{{
				Message:    "[root](http://my.gitlab.com/root) assigned you to issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			}, {
				Message:    "#### confidential issue\n##### [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)\n###### new issue by [root](http://my.gitlab.com/root) on [2019-04-06 21:03:04 UTC](http://localhost:3000/manland/webhook/issues/1)\n\nconfidential details",
				ToUsers:    []string{},
				ToChannels: []string{"channel1"},
				From:       "root",
			}},
			warnings: []string{},
		},
		{
			testTitle: "confidential issue without confidential_issues feature does not notify channel",
			fixture:   NewConfidentialIssue,
			gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
				{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
			}),
			res: []*HandleWebhook{{
				Message:    "[root](http://my.gitlab.com/root) assigned you to issue [manland/webhook#1](http://localhost:3000/manland/webhook/issues/1)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			}},
			warnings: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			issueEvent := &gitlab.IssueEvent{}
			if err := json.Unmarshal([]byte(test.fixture), issueEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, warnings, err := w.HandleIssue(context.Background(), issueEvent, gitlab.EventConfidentialIssue)
			assert.Empty(t, err)
			assert.Equal(t, len(test.res), len(res))
			assert.ElementsMatch(t, test.warnings, warnings)
			for index := range res {
				assert.Equal(t, test.res[index].Message, res[index].Message)
				assert.EqualValues(t, test.res[index].ToUsers, res[index].ToUsers)
				assert.ElementsMatch(t, test.res[index].ToChannels, res[index].ToChannels)
				assert.Equal(t, test.res[index].From, res[index].From)
			}

			assert.True(t, test.gitlabRetreiver.gotIsConfidential,
				"confidential issues must be looked up with the confidential flag so the permission check is enforced")
		})
	}
}

func TestIssueWebhookPassesConfidentialFlag(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testTitle      string
		fixture        string
		expectedIsConf bool
	}{
		{
			testTitle:      "non-confidential issue does not force the permission check",
			fixture:        NewIssue,
			expectedIsConf: false,
		},
		{
			testTitle:      "confidential issue forces the permission check",
			fixture:        NewConfidentialIssue,
			expectedIsConf: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.testTitle, func(t *testing.T) {
			retreiver := newFakeWebhook([]*subscription.Subscription{
				{ChannelID: "channel1", CreatorID: "1", Features: "issues,confidential_issues", Repository: "manland/webhook"},
			})
			w := NewWebhook(retreiver)
			issueEvent := &gitlab.IssueEvent{}
			require.NoError(t, json.Unmarshal([]byte(test.fixture), issueEvent))

			_, _, err := w.HandleIssue(context.Background(), issueEvent, gitlab.EventTypeIssue)
			require.NoError(t, err)

			assert.Equal(t, test.expectedIsConf, retreiver.gotIsConfidential)
		})
	}
}

// A confidential issue delivered under the regular Issue Hook must still be
// gated on the confidential_issues feature, not just on the event type.
func TestConfidentialIssueUnderRegularEventTypeIsGated(t *testing.T) {
	t.Parallel()
	retreiver := newFakeWebhook([]*subscription.Subscription{
		{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
	})
	w := NewWebhook(retreiver)
	issueEvent := &gitlab.IssueEvent{}
	require.NoError(t, json.Unmarshal([]byte(NewConfidentialIssue), issueEvent))

	res, _, err := w.HandleIssue(context.Background(), issueEvent, gitlab.EventTypeIssue)
	require.NoError(t, err)

	for _, handler := range res {
		assert.Empty(t, handler.ToChannels,
			"confidential issue must not reach a channel lacking the confidential_issues feature")
	}
}

// Mentions are parsed from the issue description, which an assignee update does
// not touch, so reassigning must not re-notify everyone mentioned in it.
func TestIssueAssignDoesNotRepeatDescriptionMentions(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		testTitle     string
		fixture       string
		expectMention bool
	}{
		{
			testTitle:     "new issue notifies mentioned users",
			fixture:       NewIssue,
			expectMention: true,
		},
		{
			testTitle:     "assignee update does not notify mentioned users",
			fixture:       AssignIssue,
			expectMention: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.testTitle, func(t *testing.T) {
			retreiver := newFakeWebhook([]*subscription.Subscription{})
			retreiver.mentionedUsernames = []string{"user2"}
			w := NewWebhook(retreiver)
			issueEvent := &gitlab.IssueEvent{}
			require.NoError(t, json.Unmarshal([]byte(test.fixture), issueEvent))

			res, _, err := w.HandleIssue(context.Background(), issueEvent, gitlab.EventTypeIssue)
			require.NoError(t, err)

			mentioned := false
			for _, handler := range res {
				if strings.Contains(handler.Message, "mentioned you on") {
					mentioned = true
				}
			}
			assert.Equal(t, test.expectMention, mentioned)
		})
	}
}

// A single update can change labels and assignees at once, which runs the
// channel filter twice over the same subscriptions.
func TestIssueUpdateWithLabelAndAssigneeChangeWarnsOnce(t *testing.T) {
	t.Parallel()
	fixture := strings.ReplaceAll(AssignIssue, `"changes":{`,
		`"changes":{"labels":{"previous":[],"current":[{"id":1,"title":"bug"}]},`)

	retreiver := newFakeWebhook([]*subscription.Subscription{
		{ChannelID: "channel1", CreatorID: "1", Features: "issues,label:1", Repository: "manland/webhook"},
	})
	w := NewWebhook(retreiver)
	issueEvent := &gitlab.IssueEvent{}
	require.NoError(t, json.Unmarshal([]byte(fixture), issueEvent))

	_, warnings, err := w.HandleIssue(context.Background(), issueEvent, gitlab.EventTypeIssue)
	require.NoError(t, err)

	assert.Equal(t, []string{`each label must be wrapped in quotes, e.g. label:"bug"`}, warnings)
}

func TestConfidentialIssueAssignIsGated(t *testing.T) {
	t.Parallel()
	confidentialAssign := strings.ReplaceAll(AssignIssue, `"confidential":false`, `"confidential":true`)

	testCases := []struct {
		testTitle           string
		features            string
		expectChannelNotify bool
	}{
		{
			testTitle:           "issue_assigns without confidential_issues is gated",
			features:            "issue_assigns",
			expectChannelNotify: false,
		},
		{
			testTitle:           "issue_assigns with confidential_issues notifies the channel",
			features:            "issue_assigns,confidential_issues",
			expectChannelNotify: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.testTitle, func(t *testing.T) {
			retreiver := newFakeWebhook([]*subscription.Subscription{
				{ChannelID: "channel1", CreatorID: "1", Features: test.features, Repository: "manland/webhook"},
			})
			w := NewWebhook(retreiver)
			issueEvent := &gitlab.IssueEvent{}
			require.NoError(t, json.Unmarshal([]byte(confidentialAssign), issueEvent))

			res, _, err := w.HandleIssue(context.Background(), issueEvent, gitlab.EventTypeIssue)
			require.NoError(t, err)
			assert.True(t, retreiver.gotIsConfidential)

			notified := false
			for _, handler := range res {
				if len(handler.ToChannels) > 0 {
					notified = true
				}
			}
			assert.Equal(t, test.expectChannelNotify, notified)
		})
	}
}
