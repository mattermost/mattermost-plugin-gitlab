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

func TestGetLineNumbers(t *testing.T) {
	tcs := []struct {
		input      string
		start, end int
	}{
		{
			input: "L19",
			start: 16,
			end:   22,
		}, {
			input: "L19-L23",
			start: 19,
			end:   23,
		}, {
			input: "L23-L19",
			start: -1,
			end:   -1,
		}, {
			input: "L",
			start: -1,
			end:   -1,
		}, {
			input: "bad",
			start: -1,
			end:   -1,
		}, {
			input: "L99-",
			start: 99,
			end:   -1,
		}, {
			input: "L2",
			start: 0,
			end:   5,
		},
	}
	for _, tc := range tcs {
		start, end := getLineNumbers(tc.input)
		assert.Equalf(t, tc.start, start, "unexpected start index for getLineNumbers(%q)", tc.input)
		assert.Equalf(t, tc.end, end, "unexpected end index for getLineNumbers(%q)", tc.input)
	}
}

func TestInsideLink(t *testing.T) {
	tcs := []struct {
		input    string
		index    int
		expected bool
	}{
		{
			input:    "[text](link)",
			index:    7,
			expected: true,
		}, {
			input:    "[text]( link space)",
			index:    8,
			expected: true,
		}, {
			input:    "text](link",
			index:    6,
			expected: true,
		}, {
			input:    "text] (link)",
			index:    7,
			expected: false,
		}, {
			input:    "text](link)",
			index:    6,
			expected: true,
		}, {
			input:    "link",
			index:    0,
			expected: false,
		}, {
			input:    " (link)",
			index:    2,
			expected: false,
		},
	}

	for _, tc := range tcs {
		assert.Equalf(t, tc.expected, isInsideLink(tc.input, tc.index), "unexpected result for isInsideLink(%q, %d)", tc.input, tc.index)
	}
}
