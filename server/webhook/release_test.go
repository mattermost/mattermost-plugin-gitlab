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

type testDataReleaseStr struct {
	testTitle       string
	fixture         string
	gitlabRetreiver *fakeWebhook
	res             []*HandleWebhook
}

var testDataRelease = []testDataReleaseStr{
	{
		testTitle:       "create release",
		fixture:         ReleaseEventCreate,
		gitlabRetreiver: newFakeWebhook(GetMockSubscriptions(ReleasesKey)),
		res: []*HandleWebhook{{
			Message: "### Release: **create**\n" +
				":new: **Status**: create\n" +
				"**Repository**: [myorg/myrepo](http://localhost:3000/myorg/myrepo.git)\n" +
				"**Release**: [v1.0.0](http://localhost:3000/myorg/myrepo/releases/v1.0.0)\n",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "",
		}},
	},
	{
		testTitle:       "update release",
		fixture:         ReleaseEventUpdate,
		gitlabRetreiver: newFakeWebhook(GetMockSubscriptions(ReleasesKey)),
		res: []*HandleWebhook{{
			Message: "### Release: **update**\n" +
				":arrows_counterclockwise: **Status**: update\n" +
				"**Repository**: [myorg/myrepo](http://localhost:3000/myorg/myrepo.git)\n" +
				"**Release**: [v1.1.0](http://localhost:3000/myorg/myrepo/releases/v1.1.0)\n",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "",
		}},
	},
	{
		testTitle:       "delete release",
		fixture:         ReleaseEventDelete,
		gitlabRetreiver: newFakeWebhook(GetMockSubscriptions(ReleasesKey)),
		res: []*HandleWebhook{{
			Message: "### Release: **delete**\n" +
				":red_circle: **Status**: delete\n" +
				"**Repository**: [myorg/myrepo](http://localhost:3000/myorg/myrepo.git)\n" +
				"**Release**: v1.2.0\n",
			ToUsers:    []string{},
			ToChannels: []string{"channel1"},
			From:       "",
		}},
	},
	{
		testTitle:       "release with no action",
		fixture:         ReleaseEventWithoutAction,
		gitlabRetreiver: newFakeWebhook(GetMockSubscriptions(ReleasesKey)),
		res:             nil,
	},
}

func TestReleaseWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range testDataRelease {
		t.Run(test.testTitle, func(t *testing.T) {
			w := NewWebhook(test.gitlabRetreiver)
			releaseEvent := &gitlab.ReleaseEvent{}
			if err := json.Unmarshal([]byte(test.fixture), releaseEvent); err != nil {
				assert.Fail(t, "can't unmarshal fixture")
			}
			res, err := w.HandleRelease(context.Background(), releaseEvent)
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
