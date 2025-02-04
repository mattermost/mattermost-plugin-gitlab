// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

type testDataDeploymentStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataDeployment = []testDataDeploymentStr{
	{
		testTitle:       "running deployment",
		fixture:         DeploymentEventRunning,
		gitlabRetreiver: newFakeWebhook(GetMockSubscriptions(DeploymentsKey)),
		res: []*HandleWebhook{{
			Message: "### Deployment Stage: **running**\n" +
				":rocket: **Status**: running\n" +
				"**Repository**: [myorg/myrepo](http://localhost:3000/myorg/myrepo.git)\n" +
				"**Triggered By**: testuser\n" +
				"**Visit deployment [here](http://localhost:3000/myorg/myrepo/deployment/123)** \n",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "testuser",
		}},
	},
	{
		testTitle:       "successful deployment",
		fixture:         DeploymentEventSuccessful,
		gitlabRetreiver: newFakeWebhook(GetMockSubscriptions(DeploymentsKey)),
		res: []*HandleWebhook{{
			Message: "### Deployment Stage: **success**\n" +
				":large_green_circle: **Status**: success\n" +
				"**Repository**: [myorg/myrepo](http://localhost:3000/myorg/myrepo.git)\n" +
				"**Triggered By**: testuser\n" +
				"**Visit deployment [here](http://localhost:3000/myorg/myrepo/deployment/456)** \n",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "testuser",
		}},
	},
	{
		testTitle:       "failed deployment",
		fixture:         DeploymentEventFailed,
		gitlabRetreiver: newFakeWebhook(GetMockSubscriptions(DeploymentsKey)),
		res: []*HandleWebhook{{
			Message: "### Deployment Stage: **failed**\n" +
				":red_circle: **Status**: failed\n" +
				"**Repository**: [myorg/myrepo](http://localhost:3000/myorg/myrepo.git)\n" +
				"**Triggered By**: testuser\n" +
				"**Visit deployment [here](http://localhost:3000/myorg/myrepo/deployment/789)** \n",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "testuser",
		}},
	},
	{
		testTitle:       "deployment with no action",
		fixture:         DeploymentEventWithoutAction,
		gitlabRetreiver: newFakeWebhook(GetMockSubscriptions(DeploymentsKey)),
		res:             nil,
	},
}

func TestDeploymentWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataDeployment {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			deploymentEvent := &gitlab.DeploymentEvent{}
			if err := json.Unmarshal([]byte(test.fixture), deploymentEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, err := w.HandleDeployment(context.Background(), deploymentEvent)
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
