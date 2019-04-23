package webhook

import (
	"encoding/json"
	"testing"

	"github.com/manland/go-gitlab"
	"github.com/manland/mattermost-plugin-gitlab/server/subscription"
	"github.com/stretchr/testify/assert"
)

type testDataPipelineStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataPipeline = []testDataPipelineStr{
	{
		testTitle:       "root start a pipeline",
		fixture:         PipelineStart,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{subscription.New("channel1", "1", "pipeline", "manland/webhook")}),
		res: []*HandleWebhook{{
			Message:    "[manland/webhook](http://localhost:3000/manland/webhook) New pipeline by [root](http://my.gitlab.com/root) for [Start gitlab-ci](http://localhost:3000/manland/webhook/commit/ec0a1bcd4580bfec3495674e412f4834ee2c2550)",
			ToUsers:    []string{}, // No DM because user know he has launch a pipeline
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
