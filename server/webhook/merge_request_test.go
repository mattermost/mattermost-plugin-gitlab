// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package webhook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

type testDataMergeRequestStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
	warnings        []string
}

var testDataMergeRequest = []testDataMergeRequestStr{
	{
		testTitle: "root open merge request for manland and display in channel1",
		fixture:   OpenMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) requested your review on [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}, {
			Message:    "#### Master\n##### [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4) new merge-request by [root](http://my.gitlab.com/root) on [2019-04-03 21:07:32 UTC](http://localhost:3000/manland/webhook/merge_requests/4)\n\ntest open merge request",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
		warnings: []string{},
	}, {
		testTitle: "root open merge request for manland and display in channel1 (subgroup)",
		fixture:   strings.ReplaceAll(OpenMergeRequest, "manland/webhook", "manland/subgroup/webhook"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/subgroup/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) requested your review on [#4](http://localhost:3000/manland/subgroup/webhook/merge_requests/4) in [manland/subgroup/webhook](http://localhost:3000/manland/subgroup/webhook)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}, {
			Message:    "#### Master\n##### [manland/subgroup/webhook!4](http://localhost:3000/manland/subgroup/webhook/merge_requests/4) new merge-request by [root](http://my.gitlab.com/root) on [2019-04-03 21:07:32 UTC](http://localhost:3000/manland/subgroup/webhook/merge_requests/4)\n\ntest open merge request",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
		warnings: []string{},
	}, {
		testTitle: "manland close merge request of root and display in channel1",
		fixture:   CloseMergeRequestByAssignee,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) closed your merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) was closed by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
		warnings: []string{},
	}, {
		testTitle: "manland reopened merge request of root and display in channel1",
		fixture:   ReopenMerge,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) reopened your merge request [#1](http://localhost:3000/manland/webhook/merge_requests/1) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!1 Update README.md](http://localhost:3000/manland/webhook/merge_requests/1) was reopened by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
		warnings: []string{},
	}, {
		testTitle:       "root assign manland to the merge-request",
		fixture:         RootUpdateAssigneeMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{}),
		res: []*HandleWebhook{
			{
				Message:    "[root](http://my.gitlab.com/root) assigned you to merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			},
		},
		warnings: []string{},
	}, {
		testTitle:       "root assign manland as reviewer to the merge-request",
		fixture:         RootUpdateReviewerMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{}),
		res: []*HandleWebhook{
			{
				Message:    "[root](http://my.gitlab.com/root) requested your review on merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "root",
			},
		},
		warnings: []string{},
	}, {
		testTitle:       "user assign manland as assignee to the merge-request",
		fixture:         UserUpdateAssigneeToManlandMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{}),
		res: []*HandleWebhook{
			{
				Message:    "[user](http://my.gitlab.com/user) assigned you to merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
				ToUsers:    []string{"manland"},
				ToChannels: []string{},
				From:       "user",
			},
		},
		warnings: []string{},
	}, {
		testTitle:       "user assign itself to the merge-request",
		fixture:         UserUpdateAssigneeToUserMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{}),
		res: []*HandleWebhook{
			{
				Message:    "[user](http://my.gitlab.com/user) assigned you to merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
				ToUsers:    []string{},
				ToChannels: []string{},
				From:       "user",
			},
		},
		warnings: []string{},
	}, {
		testTitle: "manland merge root merge-request and display in channel1",
		fixture:   MergeRequestMerged,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) merged your merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) was merged by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
		warnings: []string{},
	}, {
		testTitle: "manland approve root merge-request and display in channel1",
		fixture:   ApproveMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) approved your merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) was approved by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
		warnings: []string{},
	}, {
		testTitle: "manland unapprove root merge-request and display in channel1",
		fixture:   strings.ReplaceAll(ApproveMergeRequest, "approved", "unapproved"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) requested changes to your merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) changes were requested by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
		warnings: []string{},
	}, {
		testTitle: "root close its own MR without assignee and display in channel1",
		fixture:   CloseMergeRequestByCreator,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) closed your merge request [#1](http://localhost:3000/manland/webhook/merge_requests/1) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{}, // no assignee
			ToChannels: []string{},
			From:       "root",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!1 Update README.md](http://localhost:3000/manland/webhook/merge_requests/1) was closed by [root](http://my.gitlab.com/root)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
		warnings: []string{},
	}, {
		testTitle: "root open merge request for manland + channel but not subscription to merges",
		fixture:   OpenMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) requested your review on [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}},
		warnings: []string{},
	},
	{
		testTitle: "manland approve root merge-request with subscription label warning",
		fixture:   ApproveMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges,label:1", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) approved your merge request [#4](http://localhost:3000/manland/webhook/merge_requests/4) in [manland/webhook](http://localhost:3000/manland/webhook)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) was approved by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
		warnings: []string{"each label must be wrapped in quotes, e.g. label:\"bug\""},
	},
}

func TestMergeRequestWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataMergeRequest {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			mergeEvent := &gitlab.MergeEvent{}
			if err := json.Unmarshal([]byte(test.fixture), mergeEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, warnings, err := w.HandleMergeRequest(context.Background(), mergeEvent)
			assert.Empty(t, err)
			assert.Equal(t, len(test.res), len(res))
			assert.ElementsMatch(t, test.warnings, warnings)
			for index := range res {
				assert.Equal(t, test.res[index].Message, res[index].Message)
				assert.Equal(t, test.res[index].ToUsers, res[index].ToUsers)
				assert.Equal(t, test.res[index].ToChannels, res[index].ToChannels)
				assert.Equal(t, test.res[index].From, res[index].From)
			}
		})
	}
}
