// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

func TestGetChannelSubscriptions(t *testing.T) {
	setupPlugin := func(t *testing.T) (*Plugin, *plugintest.API) {
		t.Helper()

		config := configuration{
			GitlabURL:               "https://example.com",
			GitlabOAuthClientID:     "client_id",
			GitlabOAuthClientSecret: "secret",
			EncryptionKey:           "aaaaaaaaaaaaaaaa",
		}

		plugin := Plugin{configuration: &config}
		plugin.initializeAPI()

		info := gitlab.UserInfo{
			UserID:         "user_id",
			GitlabUsername: "gitlab_username",
			GitlabUserID:   0,
		}

		jsonInfo, err := json.Marshal(info)
		require.NoError(t, err)

		mock := &plugintest.API{}
		plugin.SetAPI(mock)
		plugin.client = pluginapi.NewClient(plugin.API, plugin.Driver)

		mock.On("KVGet", "user_id_userinfo").Return(jsonInfo, nil).Once()

		return &plugin, mock
	}

	t.Run("no permission to channel", func(t *testing.T) {
		plugin, mock := setupPlugin(t)

		mock.On("HasPermissionToChannel", "user_id", "id", model.PermissionReadChannel).Return(false, nil).Once()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/channel/id/subscriptions", nil)
		r.Header.Set("Mattermost-User-ID", "user_id")

		plugin.ServeHTTP(nil, w, r)

		result := w.Result()
		assert.NotNil(t, result)
		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
	})

	t.Run("no subscriptions", func(t *testing.T) {
		plugin, mock := setupPlugin(t)

		mock.On("HasPermissionToChannel", "user_id", "id", model.PermissionReadChannel).Return(true, nil).Once()
		mock.On("KVGet", SubscriptionsKey).Return([]byte(`{"Repositories":{"repo1":[]}}`), nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/channel/id/subscriptions", nil)
		r.Header.Set("Mattermost-User-ID", "user_id")

		plugin.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		data, err := io.ReadAll(result.Body)
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "[]", string(data))
	})

	t.Run("no subscriptions for channel", func(t *testing.T) {
		plugin, mock := setupPlugin(t)

		mock.On("HasPermissionToChannel", "user_id", "id", model.PermissionReadChannel).Return(true, nil).Once()
		mock.On("KVGet", SubscriptionsKey).Return([]byte(`{"Repositories":{"namespace":[{"ChannelID":"other","Repository":"repo1"}]}}`), nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/channel/id/subscriptions", nil)
		r.Header.Set("Mattermost-User-ID", "user_id")

		plugin.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		data, err := io.ReadAll(result.Body)
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, "[]", string(data))
	})

	t.Run("subscriptions for channel", func(t *testing.T) {
		plugin, mock := setupPlugin(t)

		mock.On("HasPermissionToChannel", "user_id", "id", model.PermissionReadChannel).Return(true, nil).Once()
		mock.On("KVGet", SubscriptionsKey).Return([]byte(`{"Repositories":{"namespace":[{"ChannelID":"other","Repository":"repo1","Features":"feature1,feature2","CreatorID":"creator1"},{"ChannelID":"id","Repository":"repo2","Features":"feature3,feature4","CreatorID":"creator2"},{"ChannelID":"id", "Repository":"repo3","Features":"feature5","CreatorID":"creator3"},{"ChannelID":"id","Repository":"repo4-empty"}]}}`), nil)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/channel/id/subscriptions", nil)
		r.Header.Set("Mattermost-User-ID", "user_id")

		plugin.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		data, err := io.ReadAll(result.Body)
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, `[{"repository_name":"repo2","repository_url":"https://example.com/repo2","features":["feature3","feature4"],"creator_id":"creator2"},{"repository_name":"repo3","repository_url":"https://example.com/repo3","features":["feature5"],"creator_id":"creator3"},{"repository_name":"repo4-empty","repository_url":"https://example.com/repo4-empty","features":[],"creator_id":""}]`, string(data))
	})
}

const testEncryptionKeyForAPI = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const testGitlabTokenForAPI = `{"access_token":"token","token_type":"Bearer","refresh_token":"refresh","expiry":"3022-10-23T15:14:43.623638795-05:00"}`

// fakeGitLabServer returns an httptest.Server that stubs GitLab API endpoints
// used by the namespace-enforcement tests:
//   - GET  /api/v4/projects/:id         → returns a project with the given path_with_namespace
//   - POST /api/v4/projects/:id/issues  → returns a minimal created issue
func fakeGitLabServer(t *testing.T, projectPathWithNamespace string) *httptest.Server {
	t.Helper()
	repoPath := "repo"
	if idx := strings.LastIndex(projectPathWithNamespace, "/"); idx >= 0 {
		repoPath = projectPathWithNamespace[idx+1:]
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v4/projects/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// GET /api/v4/projects/:id — project metadata (used by ensureProjectInAllowedGroup)
		if r.Method == http.MethodGet {
			project := map[string]interface{}{
				"id":                  123,
				"path":                repoPath,
				"path_with_namespace": projectPathWithNamespace,
				"web_url":             "https://example.com/" + projectPathWithNamespace,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(project)
			return
		}

		// POST /api/v4/projects/:id/issues — issue creation
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues") {
			issue := map[string]interface{}{
				"id": 1, "iid": 1, "project_id": 123, "title": "Test",
				"web_url": "https://example.com/" + projectPathWithNamespace + "/-/issues/1",
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(issue)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
}

// setupNamespaceTestPlugin wires up a Plugin backed by the given fake GitLab URL
// and gitlabGroup, with all common mocks pre-configured.
// Use extraMocks to register additional expectations (e.g. SendEphemeralPost, GetPost).
func setupNamespaceTestPlugin(t *testing.T, gitlabURL, gitlabGroup string, extraMocks func(*plugintest.API)) *Plugin {
	t.Helper()
	config := configuration{
		GitlabURL:               gitlabURL,
		GitlabGroup:             gitlabGroup,
		EncryptionKey:           testEncryptionKeyForAPI,
		GitlabOAuthClientID:     "client_id",
		GitlabOAuthClientSecret: "client_secret",
	}
	info := gitlab.UserInfo{UserID: "user_id", GitlabUsername: "gitlab_username", GitlabUserID: 0}
	jsonInfo, err := json.Marshal(info)
	require.NoError(t, err)
	encryptedToken, err := encrypt([]byte(testEncryptionKeyForAPI), testGitlabTokenForAPI)
	require.NoError(t, err)

	siteURL := "https://example.com"
	conf := &model.Config{ServiceSettings: model.ServiceSettings{SiteURL: &siteURL}}
	mockAPI := &plugintest.API{}
	mockAPI.On("GetConfig", mock.Anything).Return(conf)
	mockAPI.On("KVGet", "user_id_userinfo").Return(jsonInfo, nil)
	mockAPI.On("KVGet", "user_id_usertoken").Return([]byte(encryptedToken), nil)
	mockAPI.On("LogAuditRec", mock.Anything).Maybe()
	mockAPI.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	mockAPI.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	if extraMocks != nil {
		extraMocks(mockAPI)
	}

	p := &Plugin{configuration: &config}
	p.initializeAPI()
	p.SetAPI(mockAPI)
	p.client = pluginapi.NewClient(mockAPI, p.Driver)
	p.GitlabClient = gitlab.New(config.GitlabURL, config.GitlabGroup, p.isNamespaceAllowed)
	return p
}

func TestCreateIssueReturns403WhenNamespaceNotAllowed(t *testing.T) {
	fakeGitLab := fakeGitLabServer(t, "othergroup/repo")
	defer fakeGitLab.Close()

	p := setupNamespaceTestPlugin(t, fakeGitLab.URL, "mygroup", nil)

	body := `{"project_id":123,"title":"Test","description":""}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/issue", bytes.NewReader([]byte(body)))
	r.Header.Set("Mattermost-User-ID", "user_id")

	p.ServeHTTP(nil, w, r)

	result := w.Result()
	defer result.Body.Close()
	data, _ := io.ReadAll(result.Body)
	assert.Equal(t, http.StatusForbidden, result.StatusCode)
	assert.Contains(t, string(data), "only repositories in the mygroup namespace are allowed")
}

func TestCreateIssueAllowsWhenGitlabGroupEmpty(t *testing.T) {
	fakeGitLab := fakeGitLabServer(t, "anygroup/repo")
	defer fakeGitLab.Close()

	p := setupNamespaceTestPlugin(t, fakeGitLab.URL, "", func(m *plugintest.API) {
		m.On("SendEphemeralPost", mock.Anything, mock.Anything).Return(&model.Post{})
	})

	body := `{"project_id":123,"title":"Test","description":""}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/issue", bytes.NewReader([]byte(body)))
	r.Header.Set("Mattermost-User-ID", "user_id")

	p.ServeHTTP(nil, w, r)

	result := w.Result()
	defer result.Body.Close()
	assert.NotEqual(t, http.StatusForbidden, result.StatusCode)
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestAttachCommentToIssueReturns403WhenNamespaceNotAllowed(t *testing.T) {
	fakeGitLab := fakeGitLabServer(t, "othergroup/repo")
	defer fakeGitLab.Close()

	post := &model.Post{Id: "post_id", ChannelId: "channel_id", Message: "msg", UserId: "user_id"}
	p := setupNamespaceTestPlugin(t, fakeGitLab.URL, "mygroup", func(m *plugintest.API) {
		m.On("GetPost", "post_id").Return(post, nil)
		m.On("GetUser", "user_id").Return(&model.User{Username: "testuser"}, nil)
	})

	body := `{"project_id":123,"iid":1,"post_id":"post_id","comment":"a comment","web_url":"https://gitlab.com/group/repo/-/issues/1"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/attachcommenttoissue", bytes.NewReader([]byte(body)))
	r.Header.Set("Mattermost-User-ID", "user_id")

	p.ServeHTTP(nil, w, r)

	result := w.Result()
	defer result.Body.Close()
	data, _ := io.ReadAll(result.Body)
	assert.Equal(t, http.StatusForbidden, result.StatusCode)
	assert.Contains(t, string(data), "only repositories in the mygroup namespace are allowed")
}
