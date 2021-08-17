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
