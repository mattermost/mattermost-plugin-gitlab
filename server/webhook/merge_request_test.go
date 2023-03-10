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
}

var testDataMergeRequest = []testDataMergeRequestStr{
	{
		testTitle: "root open merge request for manland and display in channel1",
		fixture:   OpenMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) requested your review on [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}, {
			Message:    "#### Master\n##### [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4) new merge-request by [root](http://my.gitlab.com/root) on [2019-04-03 21:07:32 UTC](http://localhost:3000/manland/webhook/merge_requests/4)\n\ntest open merge request",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
	}, {
		testTitle: "root open merge request for manland and display in channel1 (subgroup)",
		fixture:   strings.ReplaceAll(OpenMergeRequest, "manland/webhook", "manland/subgroup/webhook"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/subgroup/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) requested your review on [manland/subgroup/webhook!4](http://localhost:3000/manland/subgroup/webhook/merge_requests/4)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}, {
			Message:    "#### Master\n##### [manland/subgroup/webhook!4](http://localhost:3000/manland/subgroup/webhook/merge_requests/4) new merge-request by [root](http://my.gitlab.com/root) on [2019-04-03 21:07:32 UTC](http://localhost:3000/manland/subgroup/webhook/merge_requests/4)\n\ntest open merge request",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
	}, {
		testTitle: "manland close merge request of root and display in channel1",
		fixture:   CloseMergeRequestByAssignee,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) closed your merge request [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) was closed by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle: "manland reopen merge request of root and display in channel1",
		fixture:   ReopenMerge,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) reopen your merge request [manland/webhook!1](http://localhost:3000/manland/webhook/merge_requests/1)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!1 Update README.md](http://localhost:3000/manland/webhook/merge_requests/1) was reopened by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle:       "root affect manland to merge-request",
		fixture:         AssigneeMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) assigned you to merge request [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}},
	}, {
		testTitle: "manland merge root merge-request and display in channel1",
		fixture:   MergeRequestMerged,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) merged your merge request [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) was merged by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle: "manland approve root merge-request and display in channel1",
		fixture:   ApproveMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) approved your merge request [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) was approved by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle: "manland unapprove root merge-request and display in channel1",
		fixture:   strings.ReplaceAll(ApproveMergeRequest, "approved", "unapproved"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland](http://my.gitlab.com/manland) requested changes to your merge request [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "manland",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!4 Master](http://localhost:3000/manland/webhook/merge_requests/4) changes were requested by [manland](http://my.gitlab.com/manland)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "manland",
		}},
	}, {
		testTitle: "root close its own MR without assignee and display in channel1",
		fixture:   CloseMergeRequestByCreator,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "merges", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) closed your merge request [manland/webhook!1](http://localhost:3000/manland/webhook/merge_requests/1)",
			ToUsers:    []string{}, // no assignee
			ToChannels: []string{},
			From:       "root",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Merge request [!1 Update README.md](http://localhost:3000/manland/webhook/merge_requests/1) was closed by [root](http://my.gitlab.com/root)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
	}, {
		testTitle: "root open merge request for manland + channel but not subscription to merges",
		fixture:   OpenMergeRequest,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "issues", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[root](http://my.gitlab.com/root) requested your review on [manland/webhook!4](http://localhost:3000/manland/webhook/merge_requests/4)",
			ToUsers:    []string{"manland"},
			ToChannels: []string{},
			From:       "root",
		}},
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
			res, err := w.HandleMergeRequest(context.Background(), mergeEvent)
			assert.Empty(t, err)
			assert.Equal(t, len(test.res), len(res))
			for index := range res {
				assert.Equal(t, test.res[index].Message, res[index].Message)
				assert.Equal(t, test.res[index].ToUsers, res[index].ToUsers)
				assert.Equal(t, test.res[index].ToChannels, res[index].ToChannels)
				assert.Equal(t, test.res[index].From, res[index].From)
			}
		})
	}
}
