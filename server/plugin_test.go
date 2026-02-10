// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
