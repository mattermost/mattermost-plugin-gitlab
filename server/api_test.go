package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

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

		token := oauth2.Token{
			AccessToken: "access_token",
			Expiry:      time.Now().Add(1 * time.Hour),
		}
		info := gitlab.UserInfo{
			UserID:         "user_id",
			Token:          &token,
			GitlabUsername: "gitlab_username",
			GitlabUserID:   0,
		}
		encryptedToken, err := encrypt([]byte(config.EncryptionKey), info.Token.AccessToken)
		require.NoError(t, err)

		info.Token.AccessToken = encryptedToken

		jsonInfo, err := json.Marshal(info)
		require.NoError(t, err)

		mock := &plugintest.API{}
		plugin.SetAPI(mock)
		plugin.client = pluginapi.NewClient(plugin.API, plugin.Driver)

		mock.On("KVGet", "user_id_gitlabtoken").Return(jsonInfo, nil).Once()

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
		defer result.Body.Close()
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
		defer result.Body.Close()
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
		defer result.Body.Close()
		data, err := io.ReadAll(result.Body)
		require.NoError(t, err)

		assert.NotNil(t, result)
		assert.Equal(t, http.StatusOK, result.StatusCode)
		assert.Equal(t, `[{"repository_name":"repo2","repository_url":"https://example.com/repo2","features":["feature3","feature4"],"creator_id":"creator2"},{"repository_name":"repo3","repository_url":"https://example.com/repo3","features":["feature5"],"creator_id":"creator3"},{"repository_name":"repo4-empty","repository_url":"https://example.com/repo4-empty","features":[],"creator_id":""}]`, string(data))
	})
}
