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

const (
	testEncryptionKeyForAPI = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testGitlabTokenForAPI   = `{"access_token":"token","token_type":"Bearer","refresh_token":"refresh","expiry":"3022-10-23T15:14:43.623638795-05:00"}`
)

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
			project := map[string]any{
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
			issue := map[string]any{
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
	defer func() { _ = result.Body.Close() }()
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
	defer func() { _ = result.Body.Close() }()
	assert.NotEqual(t, http.StatusForbidden, result.StatusCode)
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestCompleteConnectUserToGitlab_StateValidation(t *testing.T) {
	validUserID := "abcdefghijklmnopqrstuvwxyz"

	setupPlugin := func(t *testing.T) *Plugin {
		t.Helper()

		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		config := configuration{
			GitlabURL:               "https://gitlab.example.com",
			GitlabOAuthClientID:     "client_id",
			GitlabOAuthClientSecret: "client_secret",
			EncryptionKey:           "aaaaaaaaaaaaaaaa",
		}

		p := &Plugin{configuration: &config}
		p.initializeAPI()

		api := &plugintest.API{}
		api.On("GetConfig").Return(mmConfig)
		api.On("KVGet", mock.Anything).Return(nil, nil).Maybe()
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)
		p.oauthBroker = NewOAuthBroker(func(_ OAuthCompleteEvent) {})
		return p
	}

	t.Run("rejects state with arbitrary KV key name", func(t *testing.T) {
		p := setupPlugin(t)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=test&state=Gitlab_Instance_Configuration_Map", nil)
		r.Header.Set("Mattermost-User-ID", validUserID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
		data, _ := io.ReadAll(result.Body)
		assert.Contains(t, string(data), "invalid state format")
	})

	t.Run("rejects state targeting user token key", func(t *testing.T) {
		p := setupPlugin(t)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=test&state="+validUserID+"_usertoken", nil)
		r.Header.Set("Mattermost-User-ID", validUserID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
		data, _ := io.ReadAll(result.Body)
		assert.Contains(t, string(data), "invalid state format")
	})

	t.Run("rejects empty state", func(t *testing.T) {
		p := setupPlugin(t)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=test&state=", nil)
		r.Header.Set("Mattermost-User-ID", validUserID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	})

	t.Run("passes validation with correctly formatted state and matching user", func(t *testing.T) {
		state := "abcdefghijklmno_" + validUserID

		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		config := configuration{
			GitlabURL:               "https://gitlab.example.com",
			GitlabOAuthClientID:     "client_id",
			GitlabOAuthClientSecret: "client_secret",
			EncryptionKey:           "aaaaaaaaaaaaaaaa",
		}

		p := &Plugin{configuration: &config}
		p.initializeAPI()

		api := &plugintest.API{}
		api.On("GetConfig").Return(mmConfig)
		api.On("KVGet", state).Return([]byte(state), nil)
		api.On("KVSetWithOptions", state, []byte(nil), mock.Anything).Return(true, nil)
		api.On("KVGet", instanceConfigNameListKey).Return(nil, nil)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)
		p.oauthBroker = NewOAuthBroker(func(_ OAuthCompleteEvent) {})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=test&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", validUserID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()

		// The request passes state validation and proceeds to the token exchange,
		// which fails because there's no real GitLab server — that's an Internal
		// Server Error, not a Bad Request or Unauthorized from the validation gates.
		assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
		data, _ := io.ReadAll(result.Body)
		assert.NotContains(t, string(data), "invalid state")
		assert.NotContains(t, string(data), "not authorized, incorrect user")

		api.AssertCalled(t, "KVGet", state)
		api.AssertCalled(t, "KVSetWithOptions", state, []byte(nil), mock.Anything)
	})

	t.Run("returns error when KV delete of state token fails", func(t *testing.T) {
		state := "abcdefghijklmno_" + validUserID

		siteURL := "https://mattermost.example.com"
		mmConfig := &model.Config{}
		mmConfig.ServiceSettings.SiteURL = &siteURL

		config := configuration{
			GitlabURL:               "https://gitlab.example.com",
			GitlabOAuthClientID:     "client_id",
			GitlabOAuthClientSecret: "client_secret",
			EncryptionKey:           "aaaaaaaaaaaaaaaa",
		}

		p := &Plugin{configuration: &config}
		p.initializeAPI()

		kvDeleteErr := model.NewAppError("KVDelete", "plugin.kv_delete.error", nil, "storage failure", http.StatusInternalServerError)

		api := &plugintest.API{}
		api.On("GetConfig").Return(mmConfig)
		api.On("KVGet", state).Return([]byte(state), nil)
		api.On("KVSetWithOptions", state, []byte(nil), mock.Anything).Return(false, kvDeleteErr)
		api.On("KVGet", instanceConfigNameListKey).Return(nil, nil)
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)
		p.oauthBroker = NewOAuthBroker(func(_ OAuthCompleteEvent) {})

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=test&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", validUserID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, result.StatusCode)
		data, _ := io.ReadAll(result.Body)
		assert.Contains(t, string(data), "error deleting stored state")

		api.AssertCalled(t, "KVSetWithOptions", state, []byte(nil), mock.Anything)
		api.AssertNotCalled(t, "KVGet", "user_id_usertoken")
	})

	t.Run("rejects state with wrong user ID", func(t *testing.T) {
		p := setupPlugin(t)

		differentUserID := "zyxwvutsrqponmlkjihgfedcba"
		state := "abcdefghijklmno_" + differentUserID

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=test&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", validUserID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusUnauthorized, result.StatusCode)
		data, _ := io.ReadAll(result.Body)
		assert.Contains(t, string(data), "not authorized, incorrect user")
	})
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
	defer func() { _ = result.Body.Close() }()
	data, _ := io.ReadAll(result.Body)
	assert.Equal(t, http.StatusForbidden, result.StatusCode)
	assert.Contains(t, string(data), "only repositories in the mygroup namespace are allowed")
}
