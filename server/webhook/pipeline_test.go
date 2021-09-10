package webhook

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

type testDataPipelineStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataPipeline = []testDataPipelineStr{
	{
		testTitle: "root start a pipeline in pending",
		fixture:   PipelinePending,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "pipeline", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{}, // we don't care about pending pipeline
	}, {
		testTitle: "root start a pipeline in running",
		fixture:   PipelineRun,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "pipeline", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) New pipeline from merge_request_event by [root](http://my.gitlab.com/root) for Start gitlab-ci\n [View Pipeline](http://my.gitlab.com/manland/webhook/-/pipelines/62)",
			ToUsers:    []string{}, // No DM because user know he has launch a pipeline
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
	}, {
		testTitle: "root start a pipeline in running (subgroup)",
		fixture:   strings.ReplaceAll(PipelineRun, "manland/webhook", "manland/subgroup/webhook"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "pipeline", Repository: "manland/subgroup/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland/subgroup/webhook](http://localhost:3000/manland/subgroup/webhook) New pipeline from merge_request_event by [root](http://my.gitlab.com/root) for Start gitlab-ci\n [View Pipeline](http://my.gitlab.com/manland/subgroup/webhook/-/pipelines/62)",
			ToUsers:    []string{}, // No DM because user know he has launch a pipeline
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
	}, {
		testTitle: "root fail a pipeline",
		fixture:   PipelineFail,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "pipeline", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Your pipeline has failed for Start gitlab-ci\n [View Pipeline](http://my.gitlab.com/manland/webhook/-/pipelines/62)",
			ToUsers:    []string{"root"},
			ToChannels: []string{},
			From:       "",
		}, {
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Pipeline by [root](http://my.gitlab.com/root) fail for Start gitlab-ci\n [View Pipeline](http://my.gitlab.com/manland/webhook/-/pipelines/62)",
			ToUsers:    []string{},
			ToChannels: []string{},
			From:       "root",
		}},
	}, {
		testTitle: "root success a pipeline",
		fixture:   PipelineSuccess,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "pipeline", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) Pipeline by [root](http://my.gitlab.com/root) success for Start gitlab-ci\n [View Pipeline](http://my.gitlab.com/manland/webhook/-/pipelines/62)",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "root",
		}},
	},
}

func TestPipelineWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataPipeline {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			pipelineEvent := &gitlab.PipelineEvent{}
			if err := json.Unmarshal([]byte(test.fixture), pipelineEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, err := w.HandlePipeline(pipelineEvent)
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
