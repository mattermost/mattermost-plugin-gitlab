// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gitlabLib "github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/webhook"
)

type fakeWebhookHandler struct{}

func (fakeWebhookHandler) HandleIssue(_ context.Context, _ *gitlabLib.IssueEvent, _ gitlabLib.EventType) ([]*webhook.HandleWebhook, []string, error) {
	return []*webhook.HandleWebhook{{
		Message: "hello",
		From:    "test",
		ToUsers: []string{"unknown"},
	}}, []string{}, nil
}

func (fakeWebhookHandler) HandleMergeRequest(_ context.Context, _ *gitlabLib.MergeEvent) ([]*webhook.HandleWebhook, []string, error) {
	return []*webhook.HandleWebhook{{
		Message:    "hello",
		From:       "test",
		ToChannels: []string{"town-square"},
	}}, []string{}, nil
}

func (fakeWebhookHandler) HandleIssueComment(_ context.Context, _ *gitlabLib.IssueCommentEvent) ([]*webhook.HandleWebhook, []string, error) {
	return []*webhook.HandleWebhook{{
		Message: "hello",
		From:    "test",
		ToUsers: []string{"known"},
	}}, []string{}, nil
}

func (fakeWebhookHandler) HandleMergeRequestComment(_ context.Context, _ *gitlabLib.MergeCommentEvent) ([]*webhook.HandleWebhook, []string, error) {
	return nil, []string{}, nil
}

func (fakeWebhookHandler) HandlePipeline(_ context.Context, _ *gitlabLib.PipelineEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}

func (fakeWebhookHandler) HandleTag(_ context.Context, _ *gitlabLib.TagEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}

func (fakeWebhookHandler) HandlePush(_ context.Context, _ *gitlabLib.PushEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}

func (fakeWebhookHandler) HandleJobs(_ context.Context, _ *gitlabLib.JobEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}

func (fakeWebhookHandler) HandleDeployment(_ context.Context, _ *gitlabLib.DeploymentEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}

func (fakeWebhookHandler) HandleRelease(_ context.Context, _ *gitlabLib.ReleaseEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}

func TestHandleWebhookBadSecret(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}}
	req := httptest.NewRequest("POST", "http://example.com/foo", bytes.NewBufferString(""))
	req.Header.Add("X-Gitlab-Token", "bad_secret")
	w := httptest.NewRecorder()
	p.handleWebhook(w, req)
	resp := w.Result()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestHandleWebhookBadBody(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: fakeWebhookHandler{}}
	mock := &plugintest.API{}
	mock.On("LogDebug", "Can't parse webhook", "err", "unexpected event type: ", "header", "", "event", "{}").Return(nil)
	p.SetAPI(mock)
	p.client = pluginapi.NewClient(mock, p.Driver)
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	w := httptest.NewRecorder()
	p.handleWebhook(w, req)
	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mock.AssertCalled(t, "LogDebug", "Can't parse webhook", "err", "unexpected event type: ", "header", "", "event", "{}")
}

func TestHandleWebhookWithKnowAuthorButUnknowToUser(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: fakeWebhookHandler{}}

	mock := &plugintest.API{}
	mock.On("KVGet", "test_gitlabusername").Return([]byte("1"), nil).Once()
	mock.On("KVGet", "unknown_gitlabusername").Return(nil, nil).Once()
	mock.On("PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: "1"}).Return(nil).Once()
	mock.On("LogInfo", "new msg", "message", "hello", "from", "test").Return(nil)
	mock.On("LogInfo", "userFrom", "from", "1").Return(nil)
	p.SetAPI(mock)
	p.client = pluginapi.NewClient(mock, p.Driver)

	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"user": {"username":"test"}}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypeIssue))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mock.AssertCalled(t, "KVGet", "test_gitlabusername")
	mock.AssertCalled(t, "KVGet", "unknown_gitlabusername")
	mock.AssertNumberOfCalls(t, "KVGet", 2)
	mock.AssertCalled(t, "PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: "1"})
	mock.AssertNumberOfCalls(t, "PublishWebSocketEvent", 1)
}

func TestHandleWebhookToChannel(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: fakeWebhookHandler{}}

	mock := &plugintest.API{}
	mock.On("KVGet", "test_gitlabusername").Return([]byte("1"), nil).Once()
	mock.On("PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: "1"}).Return(nil).Once()
	mock.On("LogInfo", "new msg", "message", "hello", "from", "test").Return(nil)
	mock.On("LogInfo", "userFrom", "from", "1").Return(nil)
	mock.On("CreatePost", &model.Post{Id: "", CreateAt: 0, UpdateAt: 0, EditAt: 0, DeleteAt: 0, IsPinned: false, UserId: "", ChannelId: "town-square", RootId: "", OriginalId: "", Message: "hello", MessageSource: "", Type: "", Hashtags: "", Filenames: model.StringArray(nil), FileIds: model.StringArray(nil), PendingPostId: "", HasReactions: false, Metadata: (*model.PostMetadata)(nil)}).Return(&model.Post{}, nil)
	fakeDedupKV(mock)
	p.SetAPI(mock)
	p.client = pluginapi.NewClient(mock, p.Driver)

	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"user": {"username":"test"}}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypeMergeRequest))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mock.AssertCalled(t, "KVGet", "test_gitlabusername")
	mock.AssertNumberOfCalls(t, "KVGet", 1)
	mock.AssertCalled(t, "PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: "1"})
	mock.AssertNumberOfCalls(t, "PublishWebSocketEvent", 1)
	mock.AssertNumberOfCalls(t, "CreatePost", 1)
}

func TestHandleWebhookForChildPipelineNotficationDisabled(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret", EnableChildPipelineNotifications: false}, WebhookHandler: fakeWebhookHandler{}}

	mock := &plugintest.API{}
	p.SetAPI(mock)
	p.client = pluginapi.NewClient(mock, p.Driver)

	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"user": {"username":"test"}, "object_attributes": {"source":"parent_pipeline"}}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypePipeline))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleWebhookForChildPipelineNotficationEnabled(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret", EnableChildPipelineNotifications: true}, WebhookHandler: fakeWebhookHandler{}}

	mock := &plugintest.API{}
	mock.On("KVGet", "test_gitlabusername").Return([]byte("1"), nil).Once()
	mock.On("PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: "1"}).Return(nil).Once()
	p.SetAPI(mock)
	p.client = pluginapi.NewClient(mock, p.Driver)

	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"user": {"username":"test"}, "object_attributes": {"source":"parent_pipeline"}}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypePipeline))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mock.AssertCalled(t, "KVGet", "test_gitlabusername")
	mock.AssertNumberOfCalls(t, "KVGet", 1)
	mock.AssertCalled(t, "PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: "1"})
	mock.AssertNumberOfCalls(t, "PublishWebSocketEvent", 1)
}

func TestNotificationDedupKey(t *testing.T) {
	t.Run("same recipient and message produce the same key", func(t *testing.T) {
		assert.Equal(t,
			notificationDedupKey("user-1", "hello"),
			notificationDedupKey("user-1", "hello"))
	})

	t.Run("different recipients produce different keys", func(t *testing.T) {
		assert.NotEqual(t,
			notificationDedupKey("user-1", "hello"),
			notificationDedupKey("user-2", "hello"))
	})

	t.Run("different messages produce different keys", func(t *testing.T) {
		assert.NotEqual(t,
			notificationDedupKey("user-1", "hello"),
			notificationDedupKey("user-1", "goodbye"))
	})

	t.Run("key has the expected prefix and length", func(t *testing.T) {
		key := notificationDedupKey("user-1", "hello")
		assert.True(t, strings.HasPrefix(key, "notif_dedup_"))
		assert.Len(t, key, len("notif_dedup_")+64) // sha256 hex digest is 64 chars
	})

	// Without the length prefix these concatenate to the same bytes.
	t.Run("shifting the recipient/message boundary does not collide", func(t *testing.T) {
		assert.NotEqual(t,
			notificationDedupKey("user-1", "hello"),
			notificationDedupKey("user-1h", "ello"))
	})
}

func TestChannelPostDedupKey(t *testing.T) {
	t.Run("same channel and message produce the same key", func(t *testing.T) {
		assert.Equal(t,
			channelPostDedupKey("channel-1", "hello"),
			channelPostDedupKey("channel-1", "hello"))
	})

	t.Run("different channels produce different keys", func(t *testing.T) {
		assert.NotEqual(t,
			channelPostDedupKey("channel-1", "hello"),
			channelPostDedupKey("channel-2", "hello"))
	})

	t.Run("different messages produce different keys", func(t *testing.T) {
		assert.NotEqual(t,
			channelPostDedupKey("channel-1", "hello"),
			channelPostDedupKey("channel-1", "goodbye"))
	})

	t.Run("key has the expected prefix and length", func(t *testing.T) {
		key := channelPostDedupKey("channel-1", "hello")
		assert.True(t, strings.HasPrefix(key, "chan_dedup_"))
		assert.Len(t, key, len("chan_dedup_")+64) // sha256 hex digest is 64 chars
	})

	// Without the length prefix these concatenate to the same bytes.
	t.Run("shifting the channel/message boundary does not collide", func(t *testing.T) {
		assert.NotEqual(t,
			channelPostDedupKey("channel-1", "hello"),
			channelPostDedupKey("channel-1h", "ello"))
	})

	t.Run("channel and DM keys never collide", func(t *testing.T) {
		assert.NotEqual(t,
			channelPostDedupKey("id-1", "hello"),
			notificationDedupKey("id-1", "hello"))
	})
}

func isDedupClaimOptions(opts model.PluginKVSetOptions) bool {
	return opts.Atomic && opts.OldValue == nil &&
		opts.ExpireInSeconds == int64(webhookDedupTTL/time.Second)
}

// isDedupReleaseOptions matches KV.Delete, which pluginapi implements as a
// plain non-atomic write of a nil value.
func isDedupReleaseOptions(opts model.PluginKVSetOptions) bool {
	return !opts.Atomic && opts.ExpireInSeconds == 0
}

// fakeDedupKV emulates the KV store's atomic-claim semantics keyed on the real
// dedup key, and returns an accessor for the currently held claims.
func fakeDedupKV(api *plugintest.API) func() []string {
	var mu sync.Mutex
	claimed := map[string]bool{}

	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.MatchedBy(isDedupClaimOptions)).
		Return(func(key string, _ []byte, _ model.PluginKVSetOptions) (bool, *model.AppError) {
			mu.Lock()
			defer mu.Unlock()
			if claimed[key] {
				return false, nil
			}
			claimed[key] = true
			return true, nil
		})

	api.On("KVSetWithOptions", mock.AnythingOfType("string"), isNilBytes, mock.MatchedBy(isDedupReleaseOptions)).
		Return(func(key string, _ []byte, _ model.PluginKVSetOptions) (bool, *model.AppError) {
			mu.Lock()
			defer mu.Unlock()
			delete(claimed, key)
			return true, nil
		}).Maybe()

	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		keys := make([]string, 0, len(claimed))
		for k := range claimed {
			keys = append(keys, k)
		}
		return keys
	}
}

func newTestPluginWithAPI(api *plugintest.API) *Plugin {
	p := &Plugin{}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	return p
}

// Minimal GitLab "Note Hook" payload that parses into an IssueCommentEvent.
const noteIssueCommentBody = `{"object_kind":"note","user":{"username":"test"},"object_attributes":{"noteable_type":"Issue"}}` //nolint:misspell // "noteable_type" is GitLab's actual webhook field name

func newIssueCommentWebhookRequest() *http.Request {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(noteIssueCommentBody))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypeNote))
	return req
}

// setupDedupTestPlugin readies a plugin for the "hello" DM that
// fakeWebhookHandler.HandleIssueComment sends to the "known" user.
func setupDedupTestPlugin(t *testing.T) (*Plugin, *plugintest.API) {
	t.Helper()

	const mattermostUserID = "1"

	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: fakeWebhookHandler{}}

	userInfo := &gitlab.UserInfo{
		UserID:   mattermostUserID,
		Settings: &gitlab.UserSettings{Notifications: true},
	}
	infoJSON, err := json.Marshal(userInfo)
	require.NoError(t, err)

	api := &plugintest.API{}
	api.On("KVGet", "test_gitlabusername").Return(nil, nil)
	api.On("KVGet", "known_gitlabusername").Return([]byte(mattermostUserID), nil)
	api.On("KVGet", mattermostUserID+GitlabUserInfoKey).Return(infoJSON, nil)
	api.On("PublishWebSocketEvent", WsEventRefresh, mock.Anything, mock.Anything).Return(nil)
	api.On("LogInfo", "new msg", "message", "hello", "from", "test").Return(nil)
	api.On("LogDebug", mock.Anything, "dedup_key", mock.Anything).Return(nil)
	api.On("GetDirectChannel", mattermostUserID, p.BotUserID).Return(&model.Channel{Id: "dm-channel"}, nil)
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == "dm-channel" && post.Message == "hello"
	})).Return(&model.Post{}, nil)

	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	return p, api
}

func TestHandleWebhookDeduplicatesDuplicateDelivery(t *testing.T) {
	p, api := setupDedupTestPlugin(t)
	claimedKeys := fakeDedupKV(api)

	for range 2 {
		w := httptest.NewRecorder()
		p.handleWebhook(w, newIssueCommentWebhookRequest())
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	}

	api.AssertNumberOfCalls(t, "CreatePost", 1)
	assert.Equal(t, []string{notificationDedupKey("1", "hello")}, claimedKeys())
}

func TestHandleWebhookDeduplicatesConcurrentDeliveries(t *testing.T) {
	p, api := setupDedupTestPlugin(t)
	claimedKeys := fakeDedupKV(api)

	const deliveries = 25
	var wg sync.WaitGroup
	wg.Add(deliveries)
	start := make(chan struct{})
	for range deliveries {
		go func() {
			defer wg.Done()
			<-start
			w := httptest.NewRecorder()
			p.handleWebhook(w, newIssueCommentWebhookRequest())
		}()
	}
	close(start)
	wg.Wait()

	api.AssertNumberOfCalls(t, "CreatePost", 1)
	assert.Equal(t, []string{notificationDedupKey("1", "hello")}, claimedKeys())
}

func TestSendDMNotificationFailsOpenOnKVError(t *testing.T) {
	api := &plugintest.API{}
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.MatchedBy(isDedupClaimOptions)).
		Return(false, model.NewAppError("KVSetWithOptions", "id", nil, "boom", http.StatusInternalServerError))
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p := newTestPluginWithAPI(api)
	api.On("GetDirectChannel", "user-1", p.BotUserID).Return(&model.Channel{Id: "dm-channel"}, nil)

	p.sendDMNotification("user-1", "hello")

	api.AssertNumberOfCalls(t, "CreatePost", 1)
}

func TestSendDMNotificationReleasesClaimOnDirectChannelFailure(t *testing.T) {
	api := &plugintest.API{}
	claimedKeys := fakeDedupKV(api)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p := newTestPluginWithAPI(api)
	api.On("GetDirectChannel", "user-1", p.BotUserID).
		Return(nil, model.NewAppError("GetDirectChannel", "id", nil, "boom", http.StatusInternalServerError))

	p.sendDMNotification("user-1", "hello")

	api.AssertNotCalled(t, "CreatePost", mock.Anything)
	assert.Empty(t, claimedKeys(), "the claim should be released when no post was attempted")
}

func TestSendDMNotificationKeepsClaimOnCreatePostFailure(t *testing.T) {
	api := &plugintest.API{}
	claimedKeys := fakeDedupKV(api)
	api.On("CreatePost", mock.Anything).
		Return(nil, model.NewAppError("CreatePost", "id", nil, "boom", http.StatusInternalServerError))
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// CreateBotDMPost logs its own warning with 3 key/value pairs.
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p := newTestPluginWithAPI(api)
	api.On("GetDirectChannel", "user-1", p.BotUserID).Return(&model.Channel{Id: "dm-channel"}, nil)

	p.sendDMNotification("user-1", "hello")

	assert.Equal(t, []string{notificationDedupKey("user-1", "hello")}, claimedKeys())
}

func TestSendDMNotificationSkipsDeleteWhenClaimWasNeverMade(t *testing.T) {
	api := &plugintest.API{}
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.MatchedBy(isDedupClaimOptions)).
		Return(false, model.NewAppError("KVSetWithOptions", "id", nil, "boom", http.StatusInternalServerError)).Once()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p := newTestPluginWithAPI(api)
	api.On("GetDirectChannel", "user-1", p.BotUserID).
		Return(nil, model.NewAppError("GetDirectChannel", "id", nil, "boom", http.StatusInternalServerError))

	p.sendDMNotification("user-1", "hello")

	api.AssertNotCalled(t, "KVSetWithOptions", mock.Anything, isNilBytes, mock.Anything)
	api.AssertNumberOfCalls(t, "KVSetWithOptions", 1)
}

// setupChannelDedupTestPlugin readies a plugin for the "hello" post that
// fakeWebhookHandler.HandleMergeRequest sends to "town-square".
func setupChannelDedupTestPlugin(t *testing.T) (*Plugin, *plugintest.API) {
	t.Helper()

	api := &plugintest.API{}
	api.On("KVGet", "test_gitlabusername").Return(nil, nil)
	api.On("PublishWebSocketEvent", WsEventRefresh, mock.Anything, mock.Anything).Return(nil)
	api.On("LogInfo", "new msg", "message", "hello", "from", "test").Return(nil)
	api.On("LogDebug", mock.Anything, "dedup_key", mock.Anything).Return(nil)
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
		return post.ChannelId == "town-square" && post.Message == "hello"
	})).Return(&model.Post{}, nil)

	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: fakeWebhookHandler{}}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	return p, api
}

func newMergeRequestWebhookRequest() *http.Request {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"user": {"username":"test"}}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypeMergeRequest))
	return req
}

func TestHandleWebhookDeduplicatesDuplicateChannelDelivery(t *testing.T) {
	p, api := setupChannelDedupTestPlugin(t)
	claimedKeys := fakeDedupKV(api)

	for range 2 {
		w := httptest.NewRecorder()
		p.handleWebhook(w, newMergeRequestWebhookRequest())
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	}

	api.AssertNumberOfCalls(t, "CreatePost", 1)
	assert.Equal(t, []string{channelPostDedupKey("town-square", "hello")}, claimedKeys())
}

func TestHandleWebhookDeduplicatesConcurrentChannelDeliveries(t *testing.T) {
	p, api := setupChannelDedupTestPlugin(t)
	claimedKeys := fakeDedupKV(api)

	const deliveries = 25
	var wg sync.WaitGroup
	wg.Add(deliveries)
	start := make(chan struct{})
	for range deliveries {
		go func() {
			defer wg.Done()
			<-start
			w := httptest.NewRecorder()
			p.handleWebhook(w, newMergeRequestWebhookRequest())
		}()
	}
	close(start)
	wg.Wait()

	api.AssertNumberOfCalls(t, "CreatePost", 1)
	assert.Equal(t, []string{channelPostDedupKey("town-square", "hello")}, claimedKeys())
}

func TestSendChannelNotification(t *testing.T) {
	t.Run("different channels each get their own post", func(t *testing.T) {
		api := &plugintest.API{}
		claimedKeys := fakeDedupKV(api)
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)

		p := newTestPluginWithAPI(api)
		p.sendChannelNotification("channel-1", "hello")
		p.sendChannelNotification("channel-2", "hello")

		// Two keys means the channel ID is part of the dedup identity.
		api.AssertNumberOfCalls(t, "CreatePost", 2)
		assert.Len(t, claimedKeys(), 2)
	})

	t.Run("different messages to one channel each get their own post", func(t *testing.T) {
		api := &plugintest.API{}
		claimedKeys := fakeDedupKV(api)
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)

		p := newTestPluginWithAPI(api)
		p.sendChannelNotification("channel-1", "hello")
		p.sendChannelNotification("channel-1", "goodbye")

		api.AssertNumberOfCalls(t, "CreatePost", 2)
		assert.Len(t, claimedKeys(), 2)
	})

	t.Run("fails open when the KV claim errors", func(t *testing.T) {
		api := &plugintest.API{}
		api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.MatchedBy(isDedupClaimOptions)).
			Return(false, model.NewAppError("KVSetWithOptions", "id", nil, "boom", http.StatusInternalServerError))
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		p := newTestPluginWithAPI(api)
		p.sendChannelNotification("channel-1", "hello")

		api.AssertNumberOfCalls(t, "CreatePost", 1)
	})

	t.Run("keeps the claim when the post fails", func(t *testing.T) {
		api := &plugintest.API{}
		claimedKeys := fakeDedupKV(api)
		api.On("CreatePost", mock.Anything).
			Return(nil, model.NewAppError("CreatePost", "id", nil, "boom", http.StatusInternalServerError))
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		p := newTestPluginWithAPI(api)
		p.sendChannelNotification("channel-1", "hello")

		assert.Equal(t, []string{channelPostDedupKey("channel-1", "hello")}, claimedKeys())
	})
}
