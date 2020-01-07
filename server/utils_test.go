package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGitlabUsernamesFromText(t *testing.T) {
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

func TestNormalizePath(t *testing.T) {
	tcs := []struct {
		Full     string
		BaseURL  string
		Expected string
	}{
		{Full: "mattermost", BaseURL: "", Expected: "mattermost"},
		{Full: "mattermost", BaseURL: "https://gitlab.com/", Expected: "mattermost"},
		{Full: "https://gitlab.com/mattermost", BaseURL: "", Expected: "mattermost"},
		{Full: "https://gitlab.com/mattermost", BaseURL: "https://gitlab.com/", Expected: "mattermost"},
		{Full: "mattermost/mattermost-server", BaseURL: "", Expected: "mattermost/mattermost-server"},
		{Full: "mattermost/mattermost-server", BaseURL: "https://gitlab.com/", Expected: "mattermost/mattermost-server"},
		{Full: "https://gitlab.com/mattermost/mattermost-server", BaseURL: "", Expected: "mattermost/mattermost-server"},
		{Full: "https://gitlab.com/mattermost/mattermost-server", BaseURL: "https://gitlab.com/", Expected: "mattermost/mattermost-server"},
		{Full: "", BaseURL: "", Expected: ""},
		{Full: "group/subgroup/project", BaseURL: "", Expected: "group/subgroup/project"},
		{Full: "https://gitlab.com/group/subgroup/project", BaseURL: "", Expected: "group/subgroup/project"},
		{Full: "http://127.0.0.1:3000/manland/personal", BaseURL: "http://127.0.0.1:3000", Expected: "manland/personal"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, normalizePath(tc.Full, tc.BaseURL))
	}
}
