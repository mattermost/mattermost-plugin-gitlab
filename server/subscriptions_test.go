// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gitLabAPI "github.com/xanzy/go-gitlab"
	"go.uber.org/mock/gomock"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	mocks "github.com/mattermost/mattermost-plugin-gitlab/server/gitlab/mocks"
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

func TestGetSubscribedChannelsForProject(t *testing.T) {
	t.Parallel()

	subscriptionsData := &Subscriptions{
		Repositories: map[string][]*subscription.Subscription{
			"group/project": {
				{ChannelID: "channel1", CreatorID: "creator1", Features: "issues", Repository: "group/project"},
			},
		},
	}
	subscriptionsJSON, err := json.Marshal(subscriptionsData)
	require.NoError(t, err)

	userInfo := gitlab.UserInfo{
		UserID:         "creator1",
		GitlabUsername: "gitlab_user",
	}
	userInfoJSON, err := json.Marshal(userInfo)
	require.NoError(t, err)

	encryptedToken, err := encrypt([]byte(testEncryptionKey), testGitlabToken)
	require.NoError(t, err)

	testCases := []struct {
		name               string
		namespace          string
		project            string
		isPublicVisibility bool
		isConfidential     bool
		accessLevel        gitLabAPI.AccessLevelValue
		expectChannels     []string
		expectGetProject   bool
	}{
		{
			name:               "public non-confidential skips permission check",
			namespace:          "group",
			project:            "project",
			isPublicVisibility: true,
			isConfidential:     false,
			expectChannels:     []string{"channel1"},
			expectGetProject:   false,
		},
		{
			name:               "public confidential enforces permission check with access",
			namespace:          "group",
			project:            "project",
			isPublicVisibility: true,
			isConfidential:     true,
			accessLevel:        gitLabAPI.ReporterPermissions,
			expectChannels:     []string{"channel1"},
			expectGetProject:   true,
		},
		{
			name:               "public confidential excludes subscription without access",
			namespace:          "group",
			project:            "project",
			isPublicVisibility: true,
			isConfidential:     true,
			accessLevel:        gitLabAPI.GuestPermissions,
			expectChannels:     []string{},
			expectGetProject:   true,
		},
		{
			name:               "private project always enforces permission check",
			namespace:          "group",
			project:            "project",
			isPublicVisibility: false,
			isConfidential:     false,
			accessLevel:        gitLabAPI.ReporterPermissions,
			expectChannels:     []string{"channel1"},
			expectGetProject:   true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)

			mockedClient := mocks.NewMockGitlab(mockCtrl)
			if test.expectGetProject {
				mockedClient.EXPECT().GetProject(gomock.Any(), gomock.Any(), gomock.Any(), test.namespace, test.project).Return(&gitLabAPI.Project{
					Permissions: &gitLabAPI.Permissions{
						ProjectAccess: &gitLabAPI.ProjectAccess{
							AccessLevel: test.accessLevel,
						},
					},
				}, nil)
			}

			api := &plugintest.API{}
			api.On("KVGet", SubscriptionsKey).Return(subscriptionsJSON, nil).Once()
			api.On("KVGet", "creator1"+GitlabUserInfoKey).Return(userInfoJSON, nil).Once()
			api.On("KVGet", "creator1_usertoken").Return([]byte(encryptedToken), nil).Once()
			api.On("LogWarn",
				mock.AnythingOfType("string"),
				mock.AnythingOfType("string"),
				mock.AnythingOfType("string"),
				mock.AnythingOfType("string"),
				mock.AnythingOfType("string"))

			p := &Plugin{
				configuration: &configuration{
					EncryptionKey: testEncryptionKey,
				},
				GitlabClient: mockedClient,
			}
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)

			subs := p.GetSubscribedChannelsForProject(
				context.Background(),
				test.namespace,
				test.project,
				test.isPublicVisibility,
				test.isConfidential,
			)

			channelIDs := make([]string, len(subs))
			for i, sub := range subs {
				channelIDs[i] = sub.ChannelID
			}
			assert.ElementsMatch(t, test.expectChannels, channelIDs)
		})
	}
}
