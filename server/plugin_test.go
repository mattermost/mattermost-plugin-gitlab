// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

func TestIsNamespaceAllowed(t *testing.T) {
	tests := []struct {
		name            string
		gitlabGroup     string
		namespace       string
		expectAllowed   bool
		expectErrSubstr string
	}{
		{
			name:          "no group lock allows any namespace",
			gitlabGroup:   "",
			namespace:     "any/namespace",
			expectAllowed: true,
		},
		{
			name:          "exact match is allowed",
			gitlabGroup:   "dev-tool",
			namespace:     "dev-tool",
			expectAllowed: true,
		},
		{
			name:          "legitimate subgroup is allowed",
			gitlabGroup:   "dev-tool",
			namespace:     "dev-tool/team-a",
			expectAllowed: true,
		},
		{
			name:            "prefix without slash is rejected",
			gitlabGroup:     "dev-tool",
			namespace:       "dev-tool-foo",
			expectAllowed:   false,
			expectErrSubstr: "only repositories in the dev-tool namespace are allowed",
		},
		{
			name:            "unrelated namespace is rejected",
			gitlabGroup:     "dev-tool",
			namespace:       "other-group",
			expectAllowed:   false,
			expectErrSubstr: "only repositories in the dev-tool namespace are allowed",
		},
		{
			name:          "deeper subgroup is allowed",
			gitlabGroup:   "dev-tool",
			namespace:     "dev-tool/team-a/project",
			expectAllowed: true,
		},
		{
			name:            "group name as prefix of another group is rejected",
			gitlabGroup:     "dev",
			namespace:       "dev-tool",
			expectAllowed:   false,
			expectErrSubstr: "only repositories in the dev namespace are allowed",
		},
		{
			name:          "allowed group with surrounding spaces uses trimmed value",
			gitlabGroup:   "  dev-tool  ",
			namespace:     "dev-tool",
			expectAllowed: true,
		},
		{
			name:          "subgroup allowed when config has surrounding spaces",
			gitlabGroup:   "  dev-tool  ",
			namespace:     "dev-tool/subgroup",
			expectAllowed: true,
		},
		{
			name:            "prefix-with-dash bypass rejected when allowed has no trailing slash",
			gitlabGroup:     "mygroup",
			namespace:       "mygroup-wrong",
			expectAllowed:   false,
			expectErrSubstr: "only repositories in the mygroup namespace are allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{
				configuration: &configuration{GitlabGroup: tt.gitlabGroup},
			}
			err := p.isNamespaceAllowed(tt.namespace)
			if tt.expectAllowed {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tt.expectErrSubstr != "" {
				assert.Contains(t, err.Error(), tt.expectErrSubstr)
			}
		})
	}
}

func TestNotifyUsersOfDisallowedSubscriptions(t *testing.T) {
	const botUserID = "bot-user-id"

	makePlugin := func(t *testing.T, api *plugintest.API, gitlabGroup string) *Plugin {
		t.Helper()
		p := &Plugin{
			configuration: &configuration{GitlabGroup: gitlabGroup},
			BotUserID:     botUserID,
		}
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)
		return p
	}

	t.Run("empty subscriptions", func(t *testing.T) {
		emptySubs := &Subscriptions{Repositories: map[string][]*subscription.Subscription{}}
		payload, err := json.Marshal(emptySubs)
		require.NoError(t, err)

		api := &plugintest.API{}
		api.On("KVGet", SubscriptionsKey).Return(payload, nil).Once()

		p := makePlugin(t, api, "dev-tool")
		p.notifyUsersOfDisallowedSubscriptions()

		api.AssertCalled(t, "KVGet", SubscriptionsKey)
		api.AssertNotCalled(t, "GetDirectChannel", mock.Anything, mock.Anything)
		api.AssertNotCalled(t, "CreatePost", mock.Anything)
	})

	t.Run("existing subscriptions with only allowed namespaces", func(t *testing.T) {
		// dev-tool and dev-tool/team-a are allowed when GitlabGroup is "dev-tool"
		subs := &Subscriptions{
			Repositories: map[string][]*subscription.Subscription{
				"dev-tool":             {{ChannelID: "ch1", CreatorID: "user1", Features: "merges", Repository: "dev-tool"}},
				"dev-tool/team-a/repo": {{ChannelID: "ch2", CreatorID: "user2", Features: "issues", Repository: "dev-tool/team-a/repo"}},
			},
		}
		payload, err := json.Marshal(subs)
		require.NoError(t, err)

		api := &plugintest.API{}
		api.On("KVGet", SubscriptionsKey).Return(payload, nil).Once()

		p := makePlugin(t, api, "dev-tool")
		p.notifyUsersOfDisallowedSubscriptions()

		api.AssertCalled(t, "KVGet", SubscriptionsKey)
		api.AssertNotCalled(t, "GetDirectChannel", mock.Anything, mock.Anything)
		api.AssertNotCalled(t, "CreatePost", mock.Anything)
	})

	t.Run("existing subscriptions with mix of allowed and disallowed namespaces", func(t *testing.T) {
		// dev-tool allowed; other-group and dev-tool-foo disallowed
		subs := &Subscriptions{
			Repositories: map[string][]*subscription.Subscription{
				"dev-tool":     {{ChannelID: "ch1", CreatorID: "user1", Features: "merges", Repository: "dev-tool"}},
				"other-group":  {{ChannelID: "ch2", CreatorID: "user2", Features: "issues", Repository: "other-group"}},
				"dev-tool-foo": {{ChannelID: "ch3", CreatorID: "user2", Features: "pushes", Repository: "dev-tool-foo"}},
			},
		}
		payload, err := json.Marshal(subs)
		require.NoError(t, err)

		api := &plugintest.API{}
		api.On("KVGet", SubscriptionsKey).Return(payload, nil).Once()
		api.On("GetChannel", "ch2").Return(&model.Channel{Id: "ch2", Name: "other-team", Type: model.ChannelTypeOpen}, nil).Once()
		api.On("GetChannel", "ch3").Return(&model.Channel{Id: "ch3", Name: "dev-channel", Type: model.ChannelTypeOpen}, nil).Once()
		api.On("GetDirectChannel", "user2", botUserID).Return(&model.Channel{Id: "dm-user2"}, nil).Once()
		api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			msg := post.Message
			// DM must list only disallowed repos grouped by channel (other-group, dev-tool-foo), not the allowed one (dev-tool)
			return strings.Contains(msg, "##### ~") && strings.Contains(msg, "other-group") &&
				strings.Contains(msg, "dev-tool-foo") && !strings.Contains(msg, "* dev-tool\n")
		})).Return(&model.Post{}, nil).Once()

		p := makePlugin(t, api, "dev-tool")
		p.notifyUsersOfDisallowedSubscriptions()

		api.AssertExpectations(t)
		api.AssertNumberOfCalls(t, "CreatePost", 1)
	})

	t.Run("failure to send DM", func(t *testing.T) {
		subs := &Subscriptions{
			Repositories: map[string][]*subscription.Subscription{
				"other-group": {{ChannelID: "ch1", CreatorID: "user1", Features: "merges", Repository: "other-group"}},
			},
		}
		payload, err := json.Marshal(subs)
		require.NoError(t, err)

		api := &plugintest.API{}
		api.On("KVGet", SubscriptionsKey).Return(payload, nil).Once()
		api.On("GetChannel", "ch1").Return(&model.Channel{Id: "ch1", Name: "town-square", Type: model.ChannelTypeOpen}, nil).Once()
		api.On("GetDirectChannel", "user1", botUserID).Return(&model.Channel{Id: "dm-user1"}, nil).Once()
		api.On("CreatePost", mock.Anything).Return(nil, &model.AppError{Message: "Unable to save the Post"}).Once()
		// CreateBotDMPost logs first, then notifyUsersOfDisallowedSubscriptions logs
		api.On("LogWarn", "CreateBotDMPost failed", "user_id", "user1", "post_type", "custom_git_group_lock", "err", "Unable to save the Post").Return(nil).Once()
		api.On("LogWarn", "Failed to send group lock change DM to user", "user_id", "user1", "err", "Unable to save the Post").Return(nil).Once()

		p := makePlugin(t, api, "dev-tool")
		p.notifyUsersOfDisallowedSubscriptions()

		api.AssertCalled(t, "KVGet", SubscriptionsKey)
		api.AssertCalled(t, "GetDirectChannel", "user1", botUserID)
		api.AssertCalled(t, "CreatePost", mock.Anything)
	})
}

func TestGetOAuthConfig(t *testing.T) {
	t.Run("returns error when no instance and no legacy credentials", func(t *testing.T) {
		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		p := &Plugin{
			configuration: &configuration{DefaultInstanceName: ""},
		}

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(nil, nil)
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		api.On("GetConfig").Return(mmConfig)
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		conf, err := p.getOAuthConfig()
		assert.Nil(t, conf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no OAuth credentials available")
	})

	t.Run("falls back to legacy plugin settings when instance not found", func(t *testing.T) {
		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		p := &Plugin{
			configuration: &configuration{
				DefaultInstanceName:     "",
				GitlabURL:               "https://gitlab.example.com",
				GitlabOAuthClientID:     "legacy-client-id",
				GitlabOAuthClientSecret: "legacy-client-secret",
			},
		}

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(nil, nil)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		api.On("GetConfig").Return(mmConfig)
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		conf, err := p.getOAuthConfig()
		require.NoError(t, err)
		require.NotNil(t, conf)
		assert.Equal(t, "legacy-client-id", conf.ClientID)
		assert.Equal(t, "legacy-client-secret", conf.ClientSecret)
		assert.Contains(t, conf.Endpoint.AuthURL, "gitlab.example.com")
	})

	t.Run("returns error when instance does not exist and no legacy credentials", func(t *testing.T) {
		instanceList := []string{"production"}
		instanceListJSON, _ := json.Marshal(instanceList)

		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		p := &Plugin{
			configuration: &configuration{DefaultInstanceName: "nonexistent"},
		}

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(instanceListJSON, nil)
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		api.On("GetConfig").Return(mmConfig)
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		conf, err := p.getOAuthConfig()
		assert.Nil(t, conf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no OAuth credentials available")
	})

	t.Run("returns error when GitLab URL is malformed", func(t *testing.T) {
		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		p := &Plugin{
			configuration: &configuration{
				DefaultInstanceName:     "",
				GitlabURL:               "ht\x7ftp://invalid",
				GitlabOAuthClientID:     "legacy-client-id",
				GitlabOAuthClientSecret: "legacy-client-secret",
			},
		}

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(nil, nil)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		api.On("GetConfig").Return(mmConfig)
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		conf, err := p.getOAuthConfig()
		assert.Nil(t, conf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse GitLab URL")
	})

	t.Run("returns valid config when instance exists in KV store", func(t *testing.T) {
		instanceList := []string{"production"}
		instanceListJSON, _ := json.Marshal(instanceList)

		instanceConfigMap := map[string]InstanceConfiguration{
			"production": {
				GitlabURL:               "https://gitlab.example.com",
				GitlabOAuthClientID:     "instance-client-id",
				GitlabOAuthClientSecret: "instance-client-secret",
			},
		}
		instanceConfigJSON, _ := json.Marshal(instanceConfigMap)

		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		p := &Plugin{
			configuration: &configuration{
				DefaultInstanceName: "production",
				GitlabURL:           "https://gitlab.example.com",
			},
		}

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(instanceListJSON, nil)
		api.On("KVGet", instanceConfigMapKey).Return(instanceConfigJSON, nil)
		api.On("GetConfig").Return(mmConfig)
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		conf, err := p.getOAuthConfig()
		require.NoError(t, err)
		require.NotNil(t, conf)
		assert.Equal(t, "instance-client-id", conf.ClientID)
		assert.Equal(t, "instance-client-secret", conf.ClientSecret)
	})

	t.Run("KV store instance takes precedence over legacy credentials", func(t *testing.T) {
		instanceList := []string{"production"}
		instanceListJSON, _ := json.Marshal(instanceList)

		instanceConfigMap := map[string]InstanceConfiguration{
			"production": {
				GitlabURL:               "https://gitlab.example.com",
				GitlabOAuthClientID:     "instance-client-id",
				GitlabOAuthClientSecret: "instance-client-secret",
			},
		}
		instanceConfigJSON, _ := json.Marshal(instanceConfigMap)

		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		p := &Plugin{
			configuration: &configuration{
				DefaultInstanceName:     "production",
				GitlabURL:               "https://gitlab.example.com",
				GitlabOAuthClientID:     "legacy-client-id",
				GitlabOAuthClientSecret: "legacy-client-secret",
			},
		}

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(instanceListJSON, nil)
		api.On("KVGet", instanceConfigMapKey).Return(instanceConfigJSON, nil)
		api.On("GetConfig").Return(mmConfig)
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		conf, err := p.getOAuthConfig()
		require.NoError(t, err)
		require.NotNil(t, conf)
		assert.Equal(t, "instance-client-id", conf.ClientID)
		assert.Equal(t, "instance-client-secret", conf.ClientSecret)
	})
}

func TestRefreshTokenReturnsErrorWhenOAuthConfigFails(t *testing.T) {
	siteURL := "https://mattermost.example.com"
	mmConfig := &model.Config{}
	mmConfig.ServiceSettings.SiteURL = &siteURL

	p := &Plugin{
		configuration: &configuration{DefaultInstanceName: ""},
	}

	api := &plugintest.API{}
	api.On("KVGet", instanceConfigNameListKey).Return(nil, nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	api.On("GetConfig").Return(mmConfig)
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	userInfo := &gitlab.UserInfo{UserID: "test-user"}
	token := &oauth2.Token{
		AccessToken:  "old-access-token",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(-1 * time.Hour),
	}

	newToken, err := p.refreshToken(userInfo, token)
	assert.Nil(t, newToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to get OAuth config for token refresh")
}
