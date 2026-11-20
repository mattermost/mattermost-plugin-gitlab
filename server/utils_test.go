// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

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
		{Text: "@john.smith", Expected: []string{"john.smith"}},
		{Text: "@john_doe", Expected: []string{"john_doe"}},
		{Text: "@john_", Expected: []string{"john_"}},
		{Text: "@john.doe-smith", Expected: []string{"john.doe-smith"}},
		{Text: "@john.smith.", Expected: []string{"john.smith"}},
		{Text: "Ping @john.smith...", Expected: []string{"john.smith"}},
		{Text: "thanks @john.", Expected: []string{"john"}},
		{Text: "@john,", Expected: []string{"john"}},
		{Text: "(@john.smith)", Expected: []string{"john.smith"}},
		{Text: "@john.smith's change", Expected: []string{"john.smith"}},
		{Text: "cc @john.smith, @jane_doe!", Expected: []string{"john.smith", "jane_doe"}},
		{Text: "@john @john.smith @john_doe", Expected: []string{"john", "john.smith", "john_doe"}},
		{Text: "@john..smith", Expected: []string{}},
		{Text: "john.smith@example.com", Expected: []string{}},
		{Text: "Please ask _@john_.", Expected: []string{"john"}},
		{Text: "__@john__", Expected: []string{"john"}},
		{Text: "_@john.smith_", Expected: []string{"john.smith"}},
		{Text: "a_@john", Expected: []string{}},
		{Text: "first._@john.smith", Expected: []string{}},
		{Text: "_@john._", Expected: []string{"john"}},
		{Text: "__@john...__", Expected: []string{"john"}},
		{Text: "_@john__", Expected: []string{"john_"}},
		{Text: "Hi .@john", Expected: []string{"john"}},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, parseGitlabUsernamesFromText(tc.Text), tc.Text)
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

func TestLastN(t *testing.T) {
	tcs := []struct {
		Text     string
		N        int
		Expected string
	}{
		{Text: "", N: -99, Expected: ""},
		{Text: "", N: -1, Expected: ""},
		{Text: "", N: 0, Expected: ""},
		{Text: "", N: 1, Expected: ""},
		{Text: "", N: 99, Expected: ""},
		{Text: "abcdef", N: 4, Expected: "**cdef"},
		{Text: "abcdefghi", N: 2, Expected: "***hi"},
		{Text: "abcdefghi", N: 0, Expected: "***"},
		{Text: "abcdefghi", N: 99, Expected: "abcdefghi"},
	}

	for _, tc := range tcs {
		assert.Equal(t, tc.Expected, lastN(tc.Text, tc.N))
	}
}
