// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gitlab

import (
	"context"
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

func TestGetGroupLabelsSearch(t *testing.T) {
	t.Run("passes search query parameter when search is non-empty", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"name":"bug"}]`))
		}))
		defer srv.Close()

		client := New(srv.URL, "", func(string) error { return nil })
		labels, err := client.GetGroupLabels(context.Background(), &UserInfo{}, "my-group", "bug", &oauth2.Token{AccessToken: "tok"})
		require.NoError(t, err)
		require.Len(t, labels, 1)
		assert.Equal(t, "bug", labels[0].Name)
		assert.Contains(t, gotQuery, "search=bug")
	})

	t.Run("omits search query parameter when search is empty", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"name":"bug"}]`))
		}))
		defer srv.Close()

		client := New(srv.URL, "", func(string) error { return nil })
		labels, err := client.GetGroupLabels(context.Background(), &UserInfo{}, "my-group", "", &oauth2.Token{AccessToken: "tok"})
		require.NoError(t, err)
		require.Len(t, labels, 1)
		assert.NotContains(t, gotQuery, "search=")
	})
}
