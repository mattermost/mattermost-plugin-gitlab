package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			description: "invalid configuration: custom OAuth app without credentials",
			config: &configuration{
				GitlabURL:                   "https://gitlab.com",
				EncryptionKey:               "abcd",
				UsePreregisteredApplication: false,
			},
			errMsg: "must have a GitLab oauth client id",
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
