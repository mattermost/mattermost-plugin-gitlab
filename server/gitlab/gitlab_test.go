// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestGitlabURL(t *testing.T) {
	t.Run("returns the URL the client was built for", func(t *testing.T) {
		assert.Equal(t, "https://gitlab.example.com", New("https://gitlab.example.com", "", nil).GitlabURL())
	})

	t.Run("defaults to gitlab.com", func(t *testing.T) {
		assert.Equal(t, Gitlabdotcom, New("", "", nil).GitlabURL())
	})
}

func TestGetGroupHooksPagination(t *testing.T) {
	var requestedPages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		requestedPages = append(requestedPages, page)
		w.Header().Set("Content-Type", "application/json")
		if page == "2" {
			_, _ = fmt.Fprint(w, `[{"id":2,"url":"https://example.com/second"}]`)
			return
		}
		w.Header().Set("X-Next-Page", "2")
		_, _ = fmt.Fprint(w, `[{"id":1,"url":"https://example.com/first"}]`)
	}))
	defer server.Close()

	hooks, err := New(server.URL, "", nil).GetGroupHooks(context.Background(), &UserInfo{}, &oauth2.Token{AccessToken: "token"}, "group")

	require.NoError(t, err)
	require.Len(t, hooks, 2)
	assert.Equal(t, "https://example.com/first", hooks[0].URL)
	assert.Equal(t, "https://example.com/second", hooks[1].URL)
	assert.Equal(t, Group, hooks[0].Scope)
	assert.Equal(t, []string{"1", "2"}, requestedPages)
}
