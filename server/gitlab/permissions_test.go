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
	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

const testGitlabUserID = 7

func newTestClient(t *testing.T, handler http.HandlerFunc) (Gitlab, *oauth2.Token) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := New(server.URL, "", func(string) error { return nil })
	return client, &oauth2.Token{AccessToken: "token"}
}

func TestGetProjectAccessLevel(t *testing.T) {
	user := &UserInfo{UserID: "user_id", GitlabUserID: testGitlabUserID}

	t.Run("returns effective access level", func(t *testing.T) {
		client, token := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v4/projects/group%2Fproject/members/all/7", r.URL.EscapedPath())
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":7,"access_level":40}`))
		})

		level, err := client.GetProjectAccessLevel(context.Background(), user, token, "group", "project")
		require.NoError(t, err)
		assert.Equal(t, internGitlab.MaintainerPermissions, level)
	})

	t.Run("returns ErrNotFound when user has no membership", func(t *testing.T) {
		client, token := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"404 Not found"}`))
		})

		level, err := client.GetProjectAccessLevel(context.Background(), user, token, "group", "project")
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Equal(t, internGitlab.NoPermissions, level)
	})

	t.Run("returns error on server failure", func(t *testing.T) {
		client, token := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		_, err := client.GetProjectAccessLevel(context.Background(), user, token, "group", "project")
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrNotFound)
	})
}

func TestGetGroupAccessLevel(t *testing.T) {
	user := &UserInfo{UserID: "user_id", GitlabUserID: testGitlabUserID}

	t.Run("returns effective access level", func(t *testing.T) {
		client, token := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v4/groups/group/members/all/7", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":7,"access_level":50}`))
		})

		level, err := client.GetGroupAccessLevel(context.Background(), user, token, "group")
		require.NoError(t, err)
		assert.Equal(t, internGitlab.OwnerPermissions, level)
	})

	t.Run("returns ErrNotFound when user has no membership", func(t *testing.T) {
		client, token := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"404 Not found"}`))
		})

		level, err := client.GetGroupAccessLevel(context.Background(), user, token, "group")
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Equal(t, internGitlab.NoPermissions, level)
	})

	t.Run("returns error on server failure", func(t *testing.T) {
		client, token := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		_, err := client.GetGroupAccessLevel(context.Background(), user, token, "group")
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrNotFound)
	})
}
