// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

func TestIsValid(t *testing.T) {
	for _, testCase := range []struct {
		description string
		config      *configuration
		errMsg      string
	}{
		{
			description: "valid configuration: pre-registered app",
			config: &configuration{
				GitlabURL:                   "https://gitlab.com",
				EncryptionKey:               "abcd",
				UsePreregisteredApplication: true,
			},
		},
		{
			description: "valid configuration: custom OAuth app",
			config: &configuration{
				GitlabURL:                   "https://gitlab.com",
				GitlabOAuthClientID:         "client-id",
				GitlabOAuthClientSecret:     "client-secret",
				EncryptionKey:               "abcd",
				UsePreregisteredApplication: false,
			},
		},
		{
			// A KV-backed instance (or no instance at all) is valid at this layer; whether
			// OAuth credentials actually resolve is checked by Plugin.isConfigured, not
			// configuration.IsValid.
			description: "valid configuration: custom OAuth app without legacy credentials",
			config: &configuration{
				GitlabURL:                   "https://gitlab.com",
				EncryptionKey:               "abcd",
				UsePreregisteredApplication: false,
			},
		},
		{
			description: "invalid configuration: custom GitLab URL with pre-registered app",
			config: &configuration{
				GitlabURL:                   "https://my-company.gitlab.com",
				UsePreregisteredApplication: true,
				EncryptionKey:               "abcd",
			},
			errMsg: "pre-registered application can only be used with official public GitLab",
		},
		{
			description: "invalid configuration: missing encryption key",
			config: &configuration{
				GitlabURL: "https://gitlab.com",
			},
			errMsg: "must have an encryption key",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := testCase.config.IsValid()
			if testCase.errMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	for _, testCase := range []struct {
		description string
		isCloud     bool
		config      *configuration

		shouldChange bool
		outputCheck  func(*testing.T, *configuration)
		errMsg       string
	}{
		{
			description: "noop",
			config: &configuration{
				EncryptionKey: "abcd",
				WebhookSecret: "efgh",
			},
			shouldChange: false,
			outputCheck: func(t *testing.T, c *configuration) {
				assert.Equal(t, "abcd", c.EncryptionKey)
				assert.Equal(t, "efgh", c.WebhookSecret)
			},
		}, {
			description: "set encryption key",
			config: &configuration{
				EncryptionKey: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *configuration) {
				assert.Len(t, c.EncryptionKey, 32)
			},
		}, {
			description: "set webhook key",
			config: &configuration{
				WebhookSecret: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *configuration) {
				assert.Len(t, c.WebhookSecret, 32)
			},
		}, {
			description: "set webhook and encryption key",
			config: &configuration{
				EncryptionKey: "",
				WebhookSecret: "",
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *configuration) {
				assert.Len(t, c.EncryptionKey, 32)
				assert.Len(t, c.WebhookSecret, 32)
			},
		}, {
			description: "Should not set UsePreregisteredApplication in on-prem",
			isCloud:     false,
			config: &configuration{
				EncryptionKey:               "abcd",
				WebhookSecret:               "efgh",
				UsePreregisteredApplication: false,
			},
			shouldChange: false,
			outputCheck: func(t *testing.T, c *configuration) {
				assert.Equal(t, "abcd", c.EncryptionKey)
				assert.Equal(t, "efgh", c.WebhookSecret)
			},
		}, {
			description: "Should set UsePreregisteredApplication in cloud if no OAuth secret is configured",
			isCloud:     true,
			config: &configuration{
				EncryptionKey:               "abcd",
				WebhookSecret:               "efgh",
				UsePreregisteredApplication: false,
			},
			shouldChange: true,
			outputCheck: func(t *testing.T, c *configuration) {
				assert.Equal(t, "abcd", c.EncryptionKey)
				assert.Equal(t, "efgh", c.WebhookSecret)

				assert.True(t, c.UsePreregisteredApplication)
			},
		}, {
			description: "Should set not UsePreregisteredApplication in cloud if OAuth secret is configured",
			isCloud:     true,
			config: &configuration{
				EncryptionKey:               "abcd",
				WebhookSecret:               "efgh",
				UsePreregisteredApplication: false,
				GitlabOAuthClientID:         "some id",
				GitlabOAuthClientSecret:     "some secret",
			},
			shouldChange: false,
			outputCheck: func(t *testing.T, c *configuration) {
				assert.Equal(t, "abcd", c.EncryptionKey)
				assert.Equal(t, "efgh", c.WebhookSecret)

				assert.False(t, c.UsePreregisteredApplication)
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			changed, err := testCase.config.setDefaults(testCase.isCloud)

			assert.Equal(t, testCase.shouldChange, changed)
			testCase.outputCheck(t, testCase.config)

			if testCase.errMsg != "" {
				require.Error(t, err)
				assert.Equal(t, testCase.errMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func gitlabClientBaseURL(t *testing.T, client gitlab.Gitlab) string {
	t.Helper()
	require.NotNil(t, client)

	internalClient, err := client.GitlabConnect(oauth2.Token{AccessToken: "access-token"})
	require.NoError(t, err)

	return internalClient.BaseURL().String()
}

func TestRefreshGitlabClient(t *testing.T) {
	newPlugin := func(config *configuration, api *plugintest.API) *Plugin {
		p := &Plugin{configuration: config}
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		return p
	}

	t.Run("prefers the default instance over legacy plugin settings", func(t *testing.T) {
		instanceListJSON, err := json.Marshal([]string{"production"})
		require.NoError(t, err)
		instanceMapJSON, err := json.Marshal(map[string]InstanceConfiguration{
			"production": {
				GitlabURL:               "https://gitlab.instance.com",
				GitlabOAuthClientID:     "instance-client-id",
				GitlabOAuthClientSecret: "instance-client-secret",
			},
		})
		require.NoError(t, err)

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(instanceListJSON, nil)
		api.On("KVGet", instanceConfigMapKey).Return(instanceMapJSON, nil)

		p := newPlugin(&configuration{
			DefaultInstanceName:     "production",
			GitlabURL:               "https://gitlab.legacy.com",
			GitlabOAuthClientID:     "legacy-client-id",
			GitlabOAuthClientSecret: "legacy-client-secret",
		}, api)

		p.refreshGitlabClient()

		baseURL := gitlabClientBaseURL(t, p.gitlabClient)
		assert.Contains(t, baseURL, "gitlab.instance.com")
		assert.NotContains(t, baseURL, "gitlab.legacy.com")
	})

	t.Run("keeps the existing client but reports unconfigured when nothing is configured", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(nil, nil)

		p := newPlugin(&configuration{
			DefaultInstanceName: "production",
			GitlabURL:           "https://gitlab.legacy.com",
		}, api)
		existingClient := gitlab.New("https://gitlab.instance.com", "", nil)
		p.gitlabClient = existingClient

		p.refreshGitlabClient()

		assert.Same(t, existingClient, p.gitlabClient)

		_, err := p.getEffectiveConfig(p.getConfiguration())
		assert.ErrorIs(t, err, ErrNotConfigured)
	})

	t.Run("keeps the last resolved instance when the instance store is unreadable", func(t *testing.T) {
		instanceListJSON, err := json.Marshal([]string{"production"})
		require.NoError(t, err)
		instanceMapJSON, err := json.Marshal(map[string]InstanceConfiguration{
			"production": {
				GitlabURL:               "https://gitlab.instance.com",
				GitlabOAuthClientID:     "instance-client-id",
				GitlabOAuthClientSecret: "instance-client-secret",
			},
		})
		require.NoError(t, err)

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(instanceListJSON, nil).Once()
		api.On("KVGet", instanceConfigMapKey).Return(instanceMapJSON, nil).Once()
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		p := newPlugin(&configuration{DefaultInstanceName: "production"}, api)
		p.refreshGitlabClient()
		resolvedClient := p.gitlabClient

		// A storage failure leaves the effective instance unknown rather than absent, so the
		// last known good instance and client must survive it.
		api.On("KVGet", instanceConfigNameListKey).
			Return(nil, model.NewAppError("KVGet", "kv.read.error", nil, "boom", http.StatusInternalServerError))
		api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		p.refreshGitlabClient()

		assert.Same(t, resolvedClient, p.gitlabClient)

		effective, err := p.getEffectiveConfig(p.getConfiguration())
		require.NoError(t, err)
		assert.Equal(t, "https://gitlab.instance.com", effective.GitlabURL)
	})

	t.Run("uses plugin settings for pre-registered applications", func(t *testing.T) {
		api := &plugintest.API{}

		p := newPlugin(&configuration{
			GitlabURL:                   "https://gitlab.com",
			UsePreregisteredApplication: true,
		}, api)

		p.refreshGitlabClient()

		assert.Contains(t, gitlabClientBaseURL(t, p.gitlabClient), "gitlab.com")
		api.AssertNotCalled(t, "KVGet", instanceConfigNameListKey)
	})

	t.Run("cached instance serves later lookups without re-reading the instance store", func(t *testing.T) {
		instanceListJSON, err := json.Marshal([]string{"production"})
		require.NoError(t, err)
		instanceMapJSON, err := json.Marshal(map[string]InstanceConfiguration{
			"production": {
				GitlabURL:               "https://gitlab.instance.com",
				GitlabOAuthClientID:     "instance-client-id",
				GitlabOAuthClientSecret: "instance-client-secret",
			},
		})
		require.NoError(t, err)

		api := &plugintest.API{}
		api.On("KVGet", instanceConfigNameListKey).Return(instanceListJSON, nil).Once()
		api.On("KVGet", instanceConfigMapKey).Return(instanceMapJSON, nil).Once()

		p := newPlugin(&configuration{
			DefaultInstanceName: "production",
			EncryptionKey:       testEncryptionKey,
		}, api)
		p.refreshGitlabClient()

		for range 3 {
			require.NoError(t, p.isConfigured())
		}

		api.AssertNumberOfCalls(t, "KVGet", 2)
	})
}
