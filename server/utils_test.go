package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGitHubUsernameFromText(t *testing.T) {
	tcs := []struct {
		Text     string
		Expected []string
	}{
		{Text: "@jwilander", Expected: []string{"jwilander"}},
		{Text: "@jwilander.", Expected: []string{"jwilander"}},
		{Text: ".@jwilander", Expected: []string{"jwilander"}},
		{Text: " @jwilander ", Expected: []string{"jwilander"}},
		{Text: "@1jwilander", Expected: []string{"1jwilander"}},
		{Text: "@j", Expected: []string{"j"}},
		{Text: "@", Expected: []string{}},
		{Text: "", Expected: []string{}},
		{Text: "jwilander", Expected: []string{}},
		{Text: "@jwilander-", Expected: []string{}},
		{Text: "@-jwilander", Expected: []string{}},
		{Text: "@jwil--ander", Expected: []string{}},
		{Text: "@jwilander @jwilander2", Expected: []string{"jwilander", "jwilander2"}},
		{Text: "@jwilander2 @jwilander", Expected: []string{"jwilander2", "jwilander"}},
		{Text: "Hey @jwilander and @jwilander2!", Expected: []string{"jwilander", "jwilander2"}},
		{Text: "@jwilander @jwilan--der2", Expected: []string{"jwilander"}},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, parseGitlabUsernamesFromText(tc.Text))
	}
}

func TestParseOwnerAndRepo(t *testing.T) {
	tcs := []struct {
		Full          string
		BaseURL       string
		ExpectedOwner string
		ExpectedRepo  string
	}{
		{Full: "mattermost", BaseURL: "", ExpectedOwner: "mattermost", ExpectedRepo: ""},
		{Full: "mattermost", BaseURL: "https://gitlab.com/", ExpectedOwner: "mattermost", ExpectedRepo: ""},
		{Full: "https://gitlab.com/mattermost", BaseURL: "", ExpectedOwner: "mattermost", ExpectedRepo: ""},
		{Full: "https://gitlab.com/mattermost", BaseURL: "https://gitlab.com/", ExpectedOwner: "mattermost", ExpectedRepo: ""},
		{Full: "mattermost/mattermost-server", BaseURL: "", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost-server"},
		{Full: "mattermost/mattermost-server", BaseURL: "https://gitlab.com/", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost-server"},
		{Full: "https://gitlab.com/mattermost/mattermost-server", BaseURL: "", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost-server"},
		{Full: "https://gitlab.com/mattermost/mattermost-server", BaseURL: "https://gitlab.com/", ExpectedOwner: "mattermost", ExpectedRepo: "mattermost-server"},
		{Full: "", BaseURL: "", ExpectedOwner: "", ExpectedRepo: ""},
		{Full: "mattermost/mattermost/invalid_repo_url", BaseURL: "", ExpectedOwner: "", ExpectedRepo: ""},
		{Full: "https://gitlab.com/mattermost/mattermost/invalid_repo_url", BaseURL: "", ExpectedOwner: "", ExpectedRepo: ""},
		{Full: "http://127.0.0.1:3000/manland/personal", BaseURL: "http://127.0.0.1:3000", ExpectedOwner: "manland", ExpectedRepo: "personal"},
	}

	for _, tc := range tcs {
		_, owner, repo := parseOwnerAndRepo(tc.Full, tc.BaseURL)

		assert.Equal(t, owner, tc.ExpectedOwner)
		assert.Equal(t, repo, tc.ExpectedRepo)
	}
}
