package main

import (
	"encoding/json"
	"testing"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

func TestSubscribe(t *testing.T) {
	testCases := []struct {
		name                 string
		info                 *gitlab.UserInfo
		namespace            string
		project              string
		channelID            string
		features             string
		initialSubscriptions *Subscriptions

		initMock                     func() *plugintest.API
		expectedError                error
		expectedUpdatedSubscriptions *Subscriptions
	}{
		{
			name:                 "should add new subscription",
			info:                 &gitlab.UserInfo{UserID: "user_id"},
			namespace:            "namespace",
			project:              "project",
			channelID:            "channelID",
			features:             "merges",
			initialSubscriptions: &Subscriptions{Repositories: map[string][]*subscription.Subscription{}},

			expectedError: nil,
			expectedUpdatedSubscriptions: &Subscriptions{
				Repositories: map[string][]*subscription.Subscription{
					"namespace/project": {
						{ChannelID: "channelID", CreatorID: "user_id", Features: "merges", Repository: "namespace/project"},
					},
				},
			},
		}, {
			name:      "should keep existing subscriptions",
			info:      &gitlab.UserInfo{UserID: "user_id"},
			namespace: "namespace",
			project:   "project",
			channelID: "channelID2",
			features:  "merges",
			initialSubscriptions: &Subscriptions{
				Repositories: map[string][]*subscription.Subscription{
					"namespace/project": {
						{ChannelID: "channelID", CreatorID: "user_id", Features: "merges", Repository: "namespace/project"},
					},
				},
			},

			expectedError: nil,
			expectedUpdatedSubscriptions: &Subscriptions{
				Repositories: map[string][]*subscription.Subscription{
					"namespace/project": {
						{ChannelID: "channelID", CreatorID: "user_id", Features: "merges", Repository: "namespace/project"},
						{ChannelID: "channelID2", CreatorID: "user_id", Features: "merges", Repository: "namespace/project"},
					},
				},
			},
		}, {
			name:      "should error on invalid features",
			info:      &gitlab.UserInfo{UserID: "user_id"},
			namespace: "namespace",
			project:   "project",
			channelID: "channelID2",
			features:  "invalid",
			initialSubscriptions: &Subscriptions{
				Repositories: map[string][]*subscription.Subscription{
					"namespace/project": {
						{ChannelID: "channelID", CreatorID: "user_id", Features: "merges", Repository: "namespace/project"},
					},
				},
			},

			expectedError:                errors.New("unknown features invalid"),
			expectedUpdatedSubscriptions: nil,
		},
	}

	t.Parallel()
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			m := &plugintest.API{}
			if test.expectedError == nil {
				initialSubscriptions, err := json.Marshal(test.initialSubscriptions)
				require.NoError(t, err)

				expectedSubscriptions, err := json.Marshal(test.expectedUpdatedSubscriptions)
				require.NoError(t, err)

				m.On("KVGet", SubscriptionsKey).Return(initialSubscriptions, nil).Once()
				m.On("KVSetWithOptions", SubscriptionsKey, expectedSubscriptions, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
			}

			p := &Plugin{configuration: &configuration{}}
			p.SetAPI(m)
			p.client = pluginapi.NewClient(m, p.Driver)

			updatedSubscriptions, err := p.Subscribe(test.info, test.namespace, test.project, test.channelID, test.features)
			if test.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, test.expectedError, err)
			}
			assert.Equal(t, test.expectedUpdatedSubscriptions, updatedSubscriptions)
			m.AssertExpectations(t)
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	testCases := []struct {
		name                         string
		channelID                    string
		repoName                     string
		initMock                     func() *plugintest.API
		shouldDelete                 bool
		shouldError                  bool
		expectedUpdatedSubscriptions *Subscriptions
	}{
		{
			name:      "should delete existing subscription",
			channelID: "1",
			repoName:  "owner/project",
			initMock: func() *plugintest.API {
				m := &plugintest.API{}
				kvget := `{"Repositories":{"owner/project":[{"ChannelID":"1","CreatorID":"1","Features":"all","Repository":"owner/project"}]}}`
				kvset := `{"Repositories":{}}`
				m.On("KVGet", SubscriptionsKey).Return([]byte(kvget), nil).Once()
				m.On("KVSetWithOptions", SubscriptionsKey, []byte(kvset), mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
				return m
			},
			shouldDelete: true,
			shouldError:  false,
			expectedUpdatedSubscriptions: &Subscriptions{
				Repositories: map[string][]*subscription.Subscription{},
			},
		}, {
			name:      "should keep other channel",
			channelID: "1",
			repoName:  "owner/project",
			initMock: func() *plugintest.API {
				m := &plugintest.API{}
				kvget := `{"Repositories":{"owner/project":[{"ChannelID":"1","CreatorID":"1","Features":"all","Repository":"owner/project"},{"ChannelID":"2","CreatorID":"1","Features":"all","Repository":"owner/project"}]}}`
				kvset := `{"Repositories":{"owner/project":[{"ChannelID":"2","CreatorID":"1","Features":"all","Repository":"owner/project"}]}}`
				m.On("KVGet", SubscriptionsKey).Return([]byte(kvget), nil).Once()
				m.On("KVSetWithOptions", SubscriptionsKey, []byte(kvset), mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
				return m
			},
			shouldDelete: true,
			shouldError:  false,
			expectedUpdatedSubscriptions: &Subscriptions{
				Repositories: map[string][]*subscription.Subscription{
					"owner/project": {
						{ChannelID: "2", CreatorID: "1", Features: "all", Repository: "owner/project"},
					},
				},
			},
		}, {
			name:      "should not delete if not exist",
			channelID: "2",
			repoName:  "owner/project",
			initMock: func() *plugintest.API {
				m := &plugintest.API{}
				kvget := `{"Repositories":{"owner/project":[{"ChannelID":"1","CreatorID":"1","Features":"all","Repository":"owner/project"}]}}`
				m.On("KVGet", SubscriptionsKey).Return([]byte(kvget), nil).Once()
				return m
			},
			shouldDelete: false,
			shouldError:  false,
			expectedUpdatedSubscriptions: &Subscriptions{
				Repositories: map[string][]*subscription.Subscription{
					"owner/project": {
						{ChannelID: "1", CreatorID: "1", Features: "all", Repository: "owner/project"},
					},
				},
			},
		}, {
			name:      "should refuse empty repo",
			channelID: "1",
			repoName:  "",
			initMock: func() *plugintest.API {
				return &plugintest.API{}
			},
			shouldDelete:                 false,
			shouldError:                  true,
			expectedUpdatedSubscriptions: nil,
		}, {
			name:      "should delete organization",
			channelID: "1",
			repoName:  "owner",
			initMock: func() *plugintest.API {
				m := &plugintest.API{}
				kvget := `{"Repositories":{"owner/":[{"ChannelID":"1","CreatorID":"1","Features":"all","Repository":"owner/"}]}}`
				kvset := `{"Repositories":{}}`
				m.On("KVGet", SubscriptionsKey).Return([]byte(kvget), nil).Once()
				m.On("KVSetWithOptions", SubscriptionsKey, []byte(kvset), mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
				return m
			},
			shouldDelete: true,
			shouldError:  false,
			expectedUpdatedSubscriptions: &Subscriptions{
				Repositories: map[string][]*subscription.Subscription{},
			},
		},
	}

	t.Parallel()
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			m := test.initMock()
			p := &Plugin{configuration: &configuration{}}
			p.SetAPI(m)
			p.client = pluginapi.NewClient(m, p.Driver)
			res, updatedSubscriptions, err := p.Unsubscribe(test.channelID, test.repoName)
			assert.Equal(t, test.shouldDelete, res)
			assert.Equal(t, test.shouldError, err != nil)
			assert.Equal(t, test.expectedUpdatedSubscriptions, updatedSubscriptions)
			m.AssertExpectations(t)
		})
	}
}
