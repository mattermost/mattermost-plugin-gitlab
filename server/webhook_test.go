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
	"sync/atomic"
	"testing"

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
}

// noteIssueCommentBody is a minimal GitLab "Note Hook" payload that parses
// into an *gitlabLib.IssueCommentEvent.
const noteIssueCommentBody = `{"object_kind":"note","user":{"username":"test"},"object_attributes":{"noteable_type":"Issue"}}` //nolint:misspell // "noteable_type" is GitLab's actual webhook field name

func newIssueCommentWebhookRequest() *http.Request {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(noteIssueCommentBody))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypeNote))
	return req
}

// setupDedupTestPlugin wires up a Plugin and mock API for a single known
// recipient ("known" GitLab username, mapped to mattermostUserID) with
// notifications enabled, ready to receive the "hello" DM from
// fakeWebhookHandler.HandleIssueComment.
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
	api.On("LogDebug", "notification already claimed, skipping", "dedup_key", mock.Anything).Return(nil)
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

	var claims atomic.Int32
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		func(string, []byte, model.PluginKVSetOptions) (bool, *model.AppError) {
			return claims.Add(1) == 1, nil
		},
	)

	// Simulate the same GitLab event being delivered twice, e.g. via
	// overlapping group/project hooks or a GitLab retry.
	for range 2 {
		w := httptest.NewRecorder()
		p.handleWebhook(w, newIssueCommentWebhookRequest())
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	}

	api.AssertNumberOfCalls(t, "CreatePost", 1)
}

func TestHandleWebhookDeduplicatesConcurrentDeliveries(t *testing.T) {
	p, api := setupDedupTestPlugin(t)

	var claims atomic.Int32
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(
		func(string, []byte, model.PluginKVSetOptions) (bool, *model.AppError) {
			return claims.Add(1) == 1, nil
		},
	)

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
}

func TestSendDMNotificationFailsOpenOnKVError(t *testing.T) {
	p := &Plugin{}

	api := &plugintest.API{}
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.Anything).
		Return(false, model.NewAppError("KVSetWithOptions", "id", nil, "boom", http.StatusInternalServerError))
	api.On("GetDirectChannel", "user-1", p.BotUserID).Return(&model.Channel{Id: "dm-channel"}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	p.sendDMNotification("user-1", "hello")

	api.AssertNumberOfCalls(t, "CreatePost", 1)
}

// TestSendDMNotificationReleasesClaimOnDirectChannelFailure verifies that a
// failure resolving the bot's DM channel (before any post is attempted)
// releases the dedup claim, so a subsequent retry isn't suppressed for the
// rest of the TTL.
func TestSendDMNotificationReleasesClaimOnDirectChannelFailure(t *testing.T) {
	p := &Plugin{}
	dedupKey := notificationDedupKey("user-1", "hello")

	api := &plugintest.API{}
	api.On("KVSetWithOptions", dedupKey, []byte("true"), mock.Anything).Return(true, nil).Once()
	api.On("GetDirectChannel", "user-1", p.BotUserID).
		Return(nil, model.NewAppError("GetDirectChannel", "id", nil, "boom", http.StatusInternalServerError))
	api.On("KVSetWithOptions", dedupKey, isNilBytes, mock.Anything).Return(true, nil).Once()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	p.sendDMNotification("user-1", "hello")

	api.AssertNotCalled(t, "CreatePost", mock.Anything)
	api.AssertCalled(t, "KVSetWithOptions", dedupKey, isNilBytes, mock.Anything)
}

// TestSendDMNotificationKeepsClaimOnCreatePostFailure verifies that a
// CreatePost failure (which may have persisted the post despite the error)
// does not release the dedup claim, so a retry can't post a duplicate.
func TestSendDMNotificationKeepsClaimOnCreatePostFailure(t *testing.T) {
	p := &Plugin{}
	dedupKey := notificationDedupKey("user-1", "hello")

	api := &plugintest.API{}
	api.On("KVSetWithOptions", dedupKey, []byte("true"), mock.Anything).Return(true, nil).Once()
	api.On("GetDirectChannel", "user-1", p.BotUserID).Return(&model.Channel{Id: "dm-channel"}, nil)
	api.On("CreatePost", mock.Anything).
		Return(nil, model.NewAppError("CreatePost", "id", nil, "boom", http.StatusInternalServerError))
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// CreateBotDMPost logs its own "CreatePost failed" warning with 3 key/value pairs.
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	p.sendDMNotification("user-1", "hello")

	api.AssertNotCalled(t, "KVSetWithOptions", dedupKey, isNilBytes, mock.Anything)
}

// TestSendDMNotificationSkipsDeleteWhenClaimWasNeverMade verifies that when
// the initial KV.Set claim fails (fail-open path) and CreateBotDMPost then
// fails with a DM-channel lookup error, no KV.Delete is attempted, since no
// claim was ever written for this recipient/message.
func TestSendDMNotificationSkipsDeleteWhenClaimWasNeverMade(t *testing.T) {
	p := &Plugin{}
	dedupKey := notificationDedupKey("user-1", "hello")

	api := &plugintest.API{}
	api.On("KVSetWithOptions", dedupKey, []byte("true"), mock.Anything).
		Return(false, model.NewAppError("KVSetWithOptions", "id", nil, "boom", http.StatusInternalServerError)).Once()
	api.On("GetDirectChannel", "user-1", p.BotUserID).
		Return(nil, model.NewAppError("GetDirectChannel", "id", nil, "boom", http.StatusInternalServerError))
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	p.sendDMNotification("user-1", "hello")

	api.AssertNotCalled(t, "KVSetWithOptions", dedupKey, isNilBytes, mock.Anything)
	api.AssertNumberOfCalls(t, "KVSetWithOptions", 1)
}
