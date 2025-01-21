// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
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

type testDataJobsStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataJobs = []testDataJobsStr{
	{
		testTitle: "root start a job in running",
		fixture:   JobRunning,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "jobs", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "### Pipeline Job Stage: **test**\n:rocket: **Status**: running\n**Repository**: [gitlab-org/gitlab-test](http://192.168.64.1:3005/gitlab-org/gitlab-test.git)\n**Triggered By**: User\n**Visit job [here](http://my.gitlab.com/gitlab-org/gitlab-test/-/jobs/1977)** \n",
			ToUsers:    []string{}, // No DM because user know he has launch a pipeline
			ToChannels: []string{"channel1"},
			From:       "User",
		}},
	},
	{
		testTitle: "root start a job in pending",
		fixture:   JobPending,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "jobs", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "### Pipeline Job Stage: **test**\n:clock1: **Status**: pending\n**Repository**: [gitlab-org/gitlab-test](http://192.168.64.1:3005/gitlab-org/gitlab-test.git)\n**Triggered By**: User\n**Visit job [here](http://my.gitlab.com/gitlab-org/gitlab-test/-/jobs/1977)** \n",
			ToUsers:    []string{}, // No DM because user know he has launch a pipeline
			ToChannels: []string{"channel1"},
			From:       "User",
		}},
	},
	{
		testTitle: "root start a job in running (subgroup)",
		fixture:   strings.ReplaceAll(JobRunning, "manland/webhook", "manland/subgroup/webhook"),
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "jobs", Repository: "manland/subgroup/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "### Pipeline Job Stage: **test**\n:rocket: **Status**: running\n**Repository**: [gitlab-org/gitlab-test](http://192.168.64.1:3005/gitlab-org/gitlab-test.git)\n**Triggered By**: User\n**Visit job [here](http://my.gitlab.com/gitlab-org/gitlab-test/-/jobs/1977)** \n",
			ToUsers:    []string{}, // No DM because user know he has launch a pipeline
			ToChannels: []string{"channel1"},
			From:       "User",
		}},
	},
	{
		testTitle: "root start a job in success",
		fixture:   JobSuccess,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "jobs", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "### Pipeline Job Stage: **test**\n:large_green_circle: **Status**: success\n**Repository**: [gitlab-org/gitlab-test](http://192.168.64.1:3005/gitlab-org/gitlab-test.git)\n**Triggered By**: User\n**Visit job [here](http://my.gitlab.com/gitlab-org/gitlab-test/-/jobs/1977)** \n",
			ToUsers:    []string{}, // No DM because user know he has launch a pipeline
			ToChannels: []string{"channel1"},
			From:       "User",
		}},
	},
	{
		testTitle: "root start a job in failed",
		fixture:   JobFailed,
		gitlabRetreiver: newFakeWebhook([]*subscription.Subscription{
			{ChannelID: "channel1", CreatorID: "1", Features: "jobs", Repository: "manland/webhook"},
		}),
		res: []*HandleWebhook{{
			Message:    "### Pipeline Job Stage: **test**\n:red_circle: **Status**: failed\n**Reason Failed**: script_failure\n**Repository**: [gitlab-org/gitlab-test](http://192.168.64.1:3005/gitlab-org/gitlab-test.git)\n**Triggered By**: User\n**Visit job [here](http://my.gitlab.com/gitlab-org/gitlab-test/-/jobs/1977)** \n",
			ToUsers:    []string{}, // No DM because user know he has launch a pipeline
			ToChannels: []string{"channel1"},
			From:       "User",
		}},
	},
}

func TestJobWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataJobs {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			jobEvent := &gitlab.JobEvent{}
			if err := json.Unmarshal([]byte(test.fixture), jobEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, err := w.HandleJobs(context.Background(), jobEvent)
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
