// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitlabURL(t *testing.T) {
	t.Run("returns the URL the client was built for", func(t *testing.T) {
		assert.Equal(t, "https://gitlab.example.com", New("https://gitlab.example.com", "", nil).GitlabURL())
	})

	t.Run("defaults to gitlab.com", func(t *testing.T) {
		assert.Equal(t, Gitlabdotcom, New("", "", nil).GitlabURL())
	})
}
