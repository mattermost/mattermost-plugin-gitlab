// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	mocks "github.com/mattermost/mattermost-plugin-gitlab/server/gitlab/mocks"
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

func TestEffectiveAccess(t *testing.T) {
	tests := []struct {
		name             string
		perms            *gitlabLib.Permissions
		expectedLevel    gitlabLib.AccessLevelValue
		expectedAllowsMR bool
	}{
		{
			name:  "nil permissions",
			perms: nil,
		},
		{
			name:  "no project or group membership",
			perms: &gitlabLib.Permissions{},
		},
		{
			name: "guest on project only",
			perms: &gitlabLib.Permissions{
				ProjectAccess: &gitlabLib.ProjectAccess{AccessLevel: gitlabLib.GuestPermissions},
			},
			expectedLevel: gitlabLib.GuestPermissions,
		},
		{
			name: "reporter on project only",
			perms: &gitlabLib.Permissions{
				ProjectAccess: &gitlabLib.ProjectAccess{AccessLevel: gitlabLib.ReporterPermissions},
			},
			expectedLevel:    gitlabLib.ReporterPermissions,
			expectedAllowsMR: true,
		},
		{
			name: "reporter inherited from group only",
			perms: &gitlabLib.Permissions{
				GroupAccess: &gitlabLib.GroupAccess{AccessLevel: gitlabLib.ReporterPermissions},
			},
			expectedLevel:    gitlabLib.ReporterPermissions,
			expectedAllowsMR: true,
		},
		{
			name: "group access outranks guest project access",
			perms: &gitlabLib.Permissions{
				ProjectAccess: &gitlabLib.ProjectAccess{AccessLevel: gitlabLib.GuestPermissions},
				GroupAccess:   &gitlabLib.GroupAccess{AccessLevel: gitlabLib.MaintainerPermissions},
			},
			expectedLevel:    gitlabLib.MaintainerPermissions,
			expectedAllowsMR: true,
		},
		{
			name: "project access outranks guest group access",
			perms: &gitlabLib.Permissions{
				ProjectAccess: &gitlabLib.ProjectAccess{AccessLevel: gitlabLib.OwnerPermissions},
				GroupAccess:   &gitlabLib.GroupAccess{AccessLevel: gitlabLib.GuestPermissions},
			},
			expectedLevel:    gitlabLib.OwnerPermissions,
			expectedAllowsMR: true,
		},
		{
			name: "guest at both levels",
			perms: &gitlabLib.Permissions{
				ProjectAccess: &gitlabLib.ProjectAccess{AccessLevel: gitlabLib.GuestPermissions},
				GroupAccess:   &gitlabLib.GroupAccess{AccessLevel: gitlabLib.GuestPermissions},
			},
			expectedLevel: gitlabLib.GuestPermissions,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			level := effectiveAccess(test.perms)
			assert.Equal(t, test.expectedLevel, level)
			assert.Equal(t, test.expectedAllowsMR, level > gitlabLib.GuestPermissions)
		})
	}
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
	mock.On("LogDebug", "Can't parse webhook", "err", "unexpected event type: ", "header", "", "body_size", 2).Return(nil)
	p.SetAPI(mock)
	p.client = pluginapi.NewClient(mock, p.Driver)
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	w := httptest.NewRecorder()
	p.handleWebhook(w, req)
	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mock.AssertCalled(t, "LogDebug", "Can't parse webhook", "err", "unexpected event type: ", "header", "", "body_size", 2)
}

func TestHandleWebhookRejectsBodyOverLimit(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: fakeWebhookHandler{}}
	body := `{"pad":"` + strings.Repeat("a", maxWebhookRequestBodySize) + `"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Add("X-Gitlab-Token", "secret")
	w := httptest.NewRecorder()
	p.handleWebhook(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Result().StatusCode)
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

const failedPipelineBody = `{
	"object_kind": "pipeline",
	"object_attributes": {"id": 3, "status": "failed", "source": "web"},
	"user": {"username": "author"},
	"project": {"path_with_namespace": "group/repo", "web_url": "https://gitlab.example.com/group/repo"},
	"commit": {"message": "Add CI configuration"}
}`

func TestHandleWebhookFailedPipelineDMsAuthor(t *testing.T) {
	const (
		authorMattermostID = "author-mm-id"
		wantDM             = "[group/repo](https://gitlab.example.com/group/repo) Your pipeline has failed for Add CI configuration [View Pipeline](https://gitlab.example.com/group/repo/-/pipelines/3)"
	)

	run := func(t *testing.T, notifications bool) *plugintest.API {
		infoJSON, err := json.Marshal(&gitlab.UserInfo{
			UserID:   authorMattermostID,
			Settings: &gitlab.UserSettings{Notifications: notifications},
		})
		require.NoError(t, err)

		p := &Plugin{configuration: &configuration{WebhookSecret: "secret", GitlabURL: "https://gitlab.example.com"}}
		p.WebhookHandler = webhook.NewWebhook(&gitlabRetreiver{p: p})

		api := &plugintest.API{}
		api.On("KVGet", "author"+GitlabUsernameKey).Return([]byte(authorMattermostID), nil).Once()
		api.On("KVGet", authorMattermostID+GitlabUserInfoKey).Return(infoJSON, nil)
		api.On("KVGet", SubscriptionsKey).Return(nil, nil)
		api.On("PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: authorMattermostID}).Return(nil).Once()
		api.On("LogInfo", "new msg", "message", mock.Anything, "from", mock.Anything).Return(nil)
		api.On("GetDirectChannel", authorMattermostID, p.BotUserID).Return(&model.Channel{Id: "author-dm"}, nil)
		api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool {
			return post.ChannelId == "author-dm" && post.Message == wantDM
		})).Return(&model.Post{}, nil)
		fakeDedupKV(api)
		p.SetAPI(api)
		p.client = pluginapi.NewClient(api, p.Driver)

		req := httptest.NewRequest("POST", "/", bytes.NewBufferString(failedPipelineBody))
		req.Header.Add("X-Gitlab-Token", "secret")
		req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypePipeline))
		w := httptest.NewRecorder()

		p.handleWebhook(w, req)

		require.Equal(t, http.StatusOK, w.Result().StatusCode)
		api.AssertNumberOfCalls(t, "PublishWebSocketEvent", 1)
		return api
	}

	t.Run("author with notifications on gets the DM", func(t *testing.T) {
		api := run(t, true)

		api.AssertNumberOfCalls(t, "CreatePost", 1)
	})

	t.Run("author with notifications off gets no DM", func(t *testing.T) {
		api := run(t, false)

		api.AssertNotCalled(t, "CreatePost", mock.Anything)
	})
}

func TestHandleWebhookFailedPipelineUnconnectedAuthor(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret", GitlabURL: "https://gitlab.example.com"}}
	p.WebhookHandler = webhook.NewWebhook(&gitlabRetreiver{p: p})

	api := &plugintest.API{}
	api.On("KVGet", "author"+GitlabUsernameKey).Return(nil, nil).Once()
	api.On("KVGet", SubscriptionsKey).Return(nil, nil)
	api.On("LogInfo", "new msg", "message", mock.Anything, "from", mock.Anything).Return(nil)
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(failedPipelineBody))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypePipeline))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)

	require.Equal(t, http.StatusOK, w.Result().StatusCode)
	api.AssertNumberOfCalls(t, "KVGet", 2) // username lookup once, plus subscriptions
	api.AssertNotCalled(t, "PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything)
	api.AssertNotCalled(t, "CreatePost", mock.Anything)
}

type twoHandlersForOneUserWebhookHandler struct {
	fakeWebhookHandler
}

func (twoHandlersForOneUserWebhookHandler) HandleIssueComment(_ context.Context, _ *gitlabLib.IssueCommentEvent) ([]*webhook.HandleWebhook, []string, error) {
	return []*webhook.HandleWebhook{
		{Message: "commented on your issue", From: "bob", ToUsers: []string{"alice"}},
		{Message: "mentioned you", From: "bob", ToUsers: []string{"alice"}},
	}, nil, nil
}

func TestHandleWebhookUserInMultipleHandlersGetsOneDM(t *testing.T) {
	const aliceMattermostID = "alice-mm-id"

	infoJSON, err := json.Marshal(&gitlab.UserInfo{
		UserID:   aliceMattermostID,
		Settings: &gitlab.UserSettings{Notifications: true},
	})
	require.NoError(t, err)

	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: twoHandlersForOneUserWebhookHandler{}}

	api := &plugintest.API{}
	api.On("KVGet", "alice"+GitlabUsernameKey).Return([]byte(aliceMattermostID), nil)
	api.On("KVGet", "bob"+GitlabUsernameKey).Return(nil, nil)
	api.On("KVGet", aliceMattermostID+GitlabUserInfoKey).Return(infoJSON, nil)
	api.On("PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: aliceMattermostID}).Return(nil)
	api.On("LogInfo", "new msg", "message", mock.Anything, "from", mock.Anything).Return(nil)
	api.On("GetDirectChannel", aliceMattermostID, p.BotUserID).Return(&model.Channel{Id: "alice-dm"}, nil)
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
	fakeDedupKV(api)
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"object_kind":"note","object_attributes":{"noteable_type":"Issue"},"user":{"username":"bob"},"project":{}}`)) //nolint:misspell // noteable_type is GitLab's field name
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypeNote))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)

	require.Equal(t, http.StatusOK, w.Result().StatusCode)
	api.AssertNumberOfCalls(t, "CreatePost", 1)
	api.AssertNumberOfCalls(t, "PublishWebSocketEvent", 1)
}

func TestHandleWebhookSkippedClaimDoesNotBlockLaterMessage(t *testing.T) {
	const aliceMattermostID = "alice-mm-id"

	infoJSON, err := json.Marshal(&gitlab.UserInfo{
		UserID:   aliceMattermostID,
		Settings: &gitlab.UserSettings{Notifications: true},
	})
	require.NoError(t, err)

	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: twoHandlersForOneUserWebhookHandler{}}

	api := &plugintest.API{}
	api.On("KVGet", "alice"+GitlabUsernameKey).Return([]byte(aliceMattermostID), nil)
	api.On("KVGet", "bob"+GitlabUsernameKey).Return(nil, nil)
	api.On("KVGet", aliceMattermostID+GitlabUserInfoKey).Return(infoJSON, nil)
	api.On("PublishWebSocketEvent", WsEventRefresh, map[string]any(nil), &model.WebsocketBroadcast{UserId: aliceMattermostID}).Return(nil)
	api.On("LogInfo", "new msg", "message", mock.Anything, "from", mock.Anything).Return(nil)
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	api.On("GetDirectChannel", aliceMattermostID, p.BotUserID).Return(&model.Channel{Id: "alice-dm"}, nil)
	api.On("CreatePost", mock.MatchedBy(func(post *model.Post) bool { return post.Message == "mentioned you" })).Return(&model.Post{}, nil)
	kv := fakeDedupKV(api)
	kv.claimed[notificationDedupKey(aliceMattermostID, "commented on your issue")] = "earlier-claim"
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"object_kind":"note","object_attributes":{"noteable_type":"Issue"},"user":{"username":"bob"},"project":{}}`)) //nolint:misspell // noteable_type is GitLab's field name
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypeNote))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)

	require.Equal(t, http.StatusOK, w.Result().StatusCode)
	api.AssertNumberOfCalls(t, "CreatePost", 1)
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

// dedupKVFake emulates the KV store's atomic-claim and compare-and-delete
// semantics keyed on the real dedup key, so tests assert on the dedup identity
// itself rather than on call counts.
type dedupKVFake struct {
	mu      sync.Mutex
	claimed map[string]string // dedup key -> claim token
}

func fakeDedupKV(api *plugintest.API) *dedupKVFake {
	kv := &dedupKVFake{claimed: map[string]string{}}

	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.MatchedBy(isDedupClaimOptions)).
		Return(func(key string, value []byte, _ model.PluginKVSetOptions) (bool, *model.AppError) {
			kv.mu.Lock()
			defer kv.mu.Unlock()
			if _, held := kv.claimed[key]; held {
				return false, nil
			}
			kv.claimed[key] = string(value)
			return true, nil
		})

	api.On("KVCompareAndDelete", mock.AnythingOfType("string"), mock.Anything).
		Return(func(key string, oldValue []byte) (bool, *model.AppError) {
			kv.mu.Lock()
			defer kv.mu.Unlock()
			if token, held := kv.claimed[key]; !held || token != string(oldValue) {
				return false, nil
			}
			delete(kv.claimed, key)
			return true, nil
		}).Maybe()

	return kv
}

func (kv *dedupKVFake) keys() []string {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	keys := make([]string, 0, len(kv.claimed))
	for k := range kv.claimed {
		keys = append(keys, k)
	}
	return keys
}

// expire drops a claim the way the KV store's TTL would.
func (kv *dedupKVFake) expire(key string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	delete(kv.claimed, key)
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
	kv := fakeDedupKV(api)

	for range 2 {
		w := httptest.NewRecorder()
		p.handleWebhook(w, newIssueCommentWebhookRequest())
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	}

	api.AssertNumberOfCalls(t, "CreatePost", 1)
	assert.Equal(t, []string{notificationDedupKey("1", "hello")}, kv.keys())
}

func TestHandleWebhookDeduplicatesConcurrentDeliveries(t *testing.T) {
	p, api := setupDedupTestPlugin(t)
	kv := fakeDedupKV(api)

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
	assert.Equal(t, []string{notificationDedupKey("1", "hello")}, kv.keys())
}

func TestSendDMNotificationFailsOpenOnKVError(t *testing.T) {
	api := &plugintest.API{}
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.MatchedBy(isDedupClaimOptions)).
		Return(false, model.NewAppError("KVSetWithOptions", "id", nil, "boom", http.StatusInternalServerError))
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p := newTestPluginWithAPI(api)
	api.On("GetDirectChannel", "user-1", p.BotUserID).Return(&model.Channel{Id: "dm-channel"}, nil)

	assert.True(t, p.sendDMNotification("user-1", "hello"))

	api.AssertNumberOfCalls(t, "CreatePost", 1)
}

func TestSendDMNotificationReleasesClaimOnDirectChannelFailure(t *testing.T) {
	api := &plugintest.API{}
	kv := fakeDedupKV(api)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p := newTestPluginWithAPI(api)
	api.On("GetDirectChannel", "user-1", p.BotUserID).
		Return(nil, model.NewAppError("GetDirectChannel", "id", nil, "boom", http.StatusInternalServerError))

	assert.False(t, p.sendDMNotification("user-1", "hello"))

	api.AssertNotCalled(t, "CreatePost", mock.Anything)
	assert.Empty(t, kv.keys(), "the claim should be released when no post was attempted")
}

func TestSendDMNotificationKeepsClaimOnCreatePostFailure(t *testing.T) {
	api := &plugintest.API{}
	kv := fakeDedupKV(api)
	api.On("CreatePost", mock.Anything).
		Return(nil, model.NewAppError("CreatePost", "id", nil, "boom", http.StatusInternalServerError))
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// CreateBotDMPost logs its own warning with 3 key/value pairs.
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p := newTestPluginWithAPI(api)
	api.On("GetDirectChannel", "user-1", p.BotUserID).Return(&model.Channel{Id: "dm-channel"}, nil)

	assert.True(t, p.sendDMNotification("user-1", "hello"), "a failed post may still have been persisted")

	assert.Equal(t, []string{notificationDedupKey("user-1", "hello")}, kv.keys())
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

	assert.False(t, p.sendDMNotification("user-1", "hello"), "no post was attempted")

	api.AssertNotCalled(t, "KVCompareAndDelete", mock.Anything, mock.Anything)
	api.AssertNumberOfCalls(t, "KVSetWithOptions", 1)
}

// A delivery slow enough for its claim to expire must not evict the claim of
// whichever delivery reacquired the key, or the event posts twice.
func TestSendDMNotificationReleaseDoesNotEvictReacquiredClaim(t *testing.T) {
	api := &plugintest.API{}
	kv := fakeDedupKV(api)
	api.On("LogDebug", mock.Anything, "dedup_key", mock.Anything).Return(nil)

	p := newTestPluginWithAPI(api)
	dedupKey := notificationDedupKey("user-1", "hello")

	deliverA, tokenA := p.claimDedupKey("test", dedupKey)
	require.True(t, deliverA)
	require.NotEmpty(t, tokenA)

	// A stalls in GetDirect long enough for its claim to time out.
	kv.expire(dedupKey)

	deliverB, tokenB := p.claimDedupKey("test", dedupKey)
	require.True(t, deliverB)
	require.NotEqual(t, tokenA, tokenB)

	// A finally fails and releases using its own, now-stale token.
	p.releaseDedupKey("test", dedupKey, tokenA)

	require.Equal(t, []string{dedupKey}, kv.keys(), "B's claim must survive A's release")
	deliverC, _ := p.claimDedupKey("test", dedupKey)
	require.False(t, deliverC, "a later delivery must still be suppressed by B's claim")
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
	kv := fakeDedupKV(api)

	for range 2 {
		w := httptest.NewRecorder()
		p.handleWebhook(w, newMergeRequestWebhookRequest())
		assert.Equal(t, http.StatusOK, w.Result().StatusCode)
	}

	api.AssertNumberOfCalls(t, "CreatePost", 1)
	assert.Equal(t, []string{channelPostDedupKey("town-square", "hello")}, kv.keys())
}

func TestHandleWebhookDeduplicatesConcurrentChannelDeliveries(t *testing.T) {
	p, api := setupChannelDedupTestPlugin(t)
	kv := fakeDedupKV(api)

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
	assert.Equal(t, []string{channelPostDedupKey("town-square", "hello")}, kv.keys())
}

func TestSendChannelNotification(t *testing.T) {
	t.Run("different channels each get their own post", func(t *testing.T) {
		api := &plugintest.API{}
		kv := fakeDedupKV(api)
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)

		p := newTestPluginWithAPI(api)
		p.sendChannelNotification("channel-1", "hello")
		p.sendChannelNotification("channel-2", "hello")

		// Two keys means the channel ID is part of the dedup identity.
		api.AssertNumberOfCalls(t, "CreatePost", 2)
		assert.Len(t, kv.keys(), 2)
	})

	t.Run("different messages to one channel each get their own post", func(t *testing.T) {
		api := &plugintest.API{}
		kv := fakeDedupKV(api)
		api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil)

		p := newTestPluginWithAPI(api)
		p.sendChannelNotification("channel-1", "hello")
		p.sendChannelNotification("channel-1", "goodbye")

		api.AssertNumberOfCalls(t, "CreatePost", 2)
		assert.Len(t, kv.keys(), 2)
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
		kv := fakeDedupKV(api)
		api.On("CreatePost", mock.Anything).
			Return(nil, model.NewAppError("CreatePost", "id", nil, "boom", http.StatusInternalServerError))
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		p := newTestPluginWithAPI(api)
		p.sendChannelNotification("channel-1", "hello")

		assert.Equal(t, []string{channelPostDedupKey("channel-1", "hello")}, kv.keys())
	})
}

//nolint:misspell // "noteable_type" is GitLab's actual webhook field name
const (
	mentionIssueNoteBody = `{"object_kind":"note","event_type":"note","user":{"username":"sender"},
		"project":{"path_with_namespace":"group/project","visibility":"private"},
		"object_attributes":{"noteable_type":"Issue","note":"secret plan, cc @%[1]s","description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/issues/1#note_1"},
		"issue":{"iid":1,"confidential":false}}`
	mentionConfidentialNoteBody = `{"object_kind":"note","event_type":"confidential_note","user":{"username":"sender"},
		"project":{"path_with_namespace":"group/project","visibility":"private"},
		"object_attributes":{"noteable_type":"Issue","note":"secret plan, cc @%[1]s","description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/issues/1#note_1"},
		"issue":{"iid":1,"confidential":false}}`
	mentionConfidentialIssueNoteBody = `{"object_kind":"note","event_type":"note","user":{"username":"sender"},
		"project":{"path_with_namespace":"group/project","visibility":"private"},
		"object_attributes":{"noteable_type":"Issue","note":"secret plan, cc @%[1]s","description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/issues/1#note_1"},
		"issue":{"iid":1,"confidential":true}}`
	mentionMergeRequestNoteBody = `{"object_kind":"note","event_type":"note","user":{"username":"sender"},
		"project":{"path_with_namespace":"group/project","visibility":"private"},
		"object_attributes":{"noteable_type":"MergeRequest","note":"secret plan, cc @%[1]s","description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/merge_requests/1#note_1"},
		"merge_request":{"iid":1}}`
	mentionPushBody = `{"object_kind":"push","user_name":"sender","user_username":"sender",
		"project":{"path_with_namespace":"group/project","visibility":"private","web_url":"https://gitlab.example.com/group/project"},
		"commits":[{"id":"abc123","message":"secret fix, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/commit/abc123"}],
		"total_commits_count":1}`
	mentionIssueBody = `{"object_kind":"issue","event_type":"issue","user":{"username":"sender"},
		"project":{"path_with_namespace":"group/project","visibility":"private"},
		"object_attributes":{"iid":1,"action":"close","confidential":false,"description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/issues/1"}}`
	mentionConfidentialIssueBody = `{"object_kind":"issue","event_type":"confidential_issue","user":{"username":"sender"},
		"project":{"path_with_namespace":"group/project","visibility":"private"},
		"object_attributes":{"iid":1,"action":"close","confidential":true,"description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/issues/1"}}`
)

func allowAnyLogs(api *plugintest.API) {
	for _, method := range []string{"LogDebug", "LogInfo", "LogWarn", "LogError"} {
		for n := 1; n <= 15; n += 2 {
			args := make([]any, n)
			for i := range args {
				args[i] = mock.Anything
			}
			api.On(method, args...).Return(nil).Maybe()
		}
	}
}

func projectWithAccess(level gitlabLib.AccessLevelValue) *gitlabLib.Project {
	return &gitlabLib.Project{
		PathWithNamespace: "group/project",
		Visibility:        gitlabLib.PrivateVisibility,
		Permissions: &gitlabLib.Permissions{
			ProjectAccess: &gitlabLib.ProjectAccess{AccessLevel: level},
		},
	}
}

// publicProjectForNonMember returns a public project as seen by a user who is
// not a member, with the given feature access levels.
func publicProjectForNonMember(issues, mergeRequests, repository gitlabLib.AccessControlValue) *gitlabLib.Project {
	return &gitlabLib.Project{
		PathWithNamespace:        "group/project",
		Visibility:               gitlabLib.PublicVisibility,
		IssuesAccessLevel:        issues,
		MergeRequestsAccessLevel: mergeRequests,
		RepositoryAccessLevel:    repository,
	}
}

func internalProjectForNonMember(issues, mergeRequests, repository gitlabLib.AccessControlValue) *gitlabLib.Project {
	project := publicProjectForNonMember(issues, mergeRequests, repository)
	project.Visibility = gitlabLib.InternalVisibility
	return project
}

var errProjectNotFound = errors.New("GET https://gitlab.example.com/api/v4/projects/group%2Fproject: 404 {message: 404 Project Not Found}")

// setupMentionWebhookPlugin connects each GitLab username to the Mattermost ID
// "<username>-mm-id" with notifications on. getProject answers GitLab project
// lookups made with that user's token. It returns the IDs of users DMed.
func setupMentionWebhookPlugin(t *testing.T, gitlabUsernames []string, getProject func(gitlabUsername string) (*gitlabLib.Project, error)) (*Plugin, func() []string) {
	t.Helper()

	encryptedToken, err := encrypt([]byte(testEncryptionKey), testGitlabToken)
	require.NoError(t, err)

	kv := map[string][]byte{}
	for _, username := range gitlabUsernames {
		mattermostID := username + "-mm-id"
		infoJSON, err := json.Marshal(&gitlab.UserInfo{
			UserID:         mattermostID,
			GitlabUsername: username,
			Settings:       &gitlab.UserSettings{Notifications: true},
		})
		require.NoError(t, err)
		kv[username+GitlabUsernameKey] = []byte(mattermostID)
		kv[mattermostID+GitlabUserInfoKey] = infoJSON
		kv[mattermostID+GitlabUserTokenKey] = []byte(encryptedToken)
	}

	mockedClient := mocks.NewMockGitlab(gomock.NewController(t))
	mockedClient.EXPECT().GetProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, info *gitlab.UserInfo, _ *oauth2.Token, _, _ string) (*gitlabLib.Project, error) {
			return getProject(info.GitlabUsername)
		}).AnyTimes()

	p := &Plugin{
		configuration: &configuration{
			WebhookSecret:     "secret",
			GitlabURL:         "https://gitlab.example.com",
			EnablePrivateRepo: true,
			EncryptionKey:     testEncryptionKey,
		},
		BotUserID:    "bot-id",
		gitlabClient: mockedClient,
	}
	p.WebhookHandler = webhook.NewWebhook(&gitlabRetreiver{p: p})

	var mu sync.Mutex
	var dmed []string

	api := &plugintest.API{}
	allowAnyLogs(api)
	fakeDedupKV(api)
	api.On("KVGet", mock.AnythingOfType("string")).Return(func(key string) ([]byte, *model.AppError) {
		return kv[key], nil
	}).Maybe()
	api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("GetDirectChannel", mock.AnythingOfType("string"), p.BotUserID).Return(func(userID, _ string) (*model.Channel, *model.AppError) {
		return &model.Channel{Id: userID + "-dm"}, nil
	}).Maybe()
	api.On("CreatePost", mock.Anything).Run(func(args mock.Arguments) {
		mu.Lock()
		defer mu.Unlock()
		dmed = append(dmed, strings.TrimSuffix(args.Get(0).(*model.Post).ChannelId, "-dm"))
	}).Return(&model.Post{}, nil).Maybe()
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	return p, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), dmed...)
	}
}

func postMentionWebhook(t *testing.T, p *Plugin, eventType gitlabLib.EventType, body string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(eventType))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)

	require.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandleWebhookMentionDMRequiresProjectAccess(t *testing.T) {
	const mentioned = "mallory"

	tests := []struct {
		name       string
		eventType  gitlabLib.EventType
		body       string
		project    *gitlabLib.Project
		projectErr error
		expectDM   bool
	}{
		{
			name:       "issue note, recipient cannot access project",
			eventType:  gitlabLib.EventTypeNote,
			body:       mentionIssueNoteBody,
			projectErr: errProjectNotFound,
		},
		{
			name:       "merge request note, recipient cannot access project",
			eventType:  gitlabLib.EventTypeNote,
			body:       mentionMergeRequestNoteBody,
			projectErr: errProjectNotFound,
		},
		{
			name:       "push commit message, recipient cannot access project",
			eventType:  gitlabLib.EventTypePush,
			body:       mentionPushBody,
			projectErr: errProjectNotFound,
		},
		{
			name:       "issue description, recipient cannot access project",
			eventType:  gitlabLib.EventTypeIssue,
			body:       mentionIssueBody,
			projectErr: errProjectNotFound,
		},
		{
			name:      "confidential note, recipient is only a guest",
			eventType: gitlabLib.EventConfidentialNote,
			body:      mentionConfidentialNoteBody,
			project:   projectWithAccess(gitlabLib.GuestPermissions),
		},
		{
			name:      "note on confidential issue, recipient is only a guest",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionConfidentialIssueNoteBody,
			project:   projectWithAccess(gitlabLib.GuestPermissions),
		},
		{
			name:      "confidential issue description, recipient is only a guest",
			eventType: gitlabLib.EventConfidentialIssue,
			body:      mentionConfidentialIssueBody,
			project:   projectWithAccess(gitlabLib.GuestPermissions),
		},
		{
			name:      "issue note, recipient is a developer",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionIssueNoteBody,
			project:   projectWithAccess(gitlabLib.DeveloperPermissions),
			expectDM:  true,
		},
		{
			name:      "merge request note, recipient is a developer",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionMergeRequestNoteBody,
			project:   projectWithAccess(gitlabLib.DeveloperPermissions),
			expectDM:  true,
		},
		{
			name:      "push commit message, recipient is a developer",
			eventType: gitlabLib.EventTypePush,
			body:      mentionPushBody,
			project:   projectWithAccess(gitlabLib.DeveloperPermissions),
			expectDM:  true,
		},
		{
			name:      "confidential note, recipient is a developer",
			eventType: gitlabLib.EventConfidentialNote,
			body:      mentionConfidentialNoteBody,
			project:   projectWithAccess(gitlabLib.DeveloperPermissions),
			expectDM:  true,
		},
		{
			name:      "public project with members-only merge requests, merge request note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionMergeRequestNoteBody,
			project:   publicProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.PrivateAccessControl, gitlabLib.EnabledAccessControl),
		},
		{
			name:      "public project with members-only issues, issue note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionIssueNoteBody,
			project:   publicProjectForNonMember(gitlabLib.PrivateAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
		},
		{
			name:      "public project with members-only repository, push commit message, recipient is not a member",
			eventType: gitlabLib.EventTypePush,
			body:      mentionPushBody,
			project:   publicProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.PrivateAccessControl),
		},
		{
			name:      "public project without feature access levels, issue note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionIssueNoteBody,
			project:   publicProjectForNonMember("", "", ""),
		},
		{
			name:      "public project with members-only merge requests, merge request note, recipient is a developer",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionMergeRequestNoteBody,
			project: func() *gitlabLib.Project {
				project := publicProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.PrivateAccessControl, gitlabLib.EnabledAccessControl)
				project.Permissions = projectWithAccess(gitlabLib.DeveloperPermissions).Permissions
				return project
			}(),
			expectDM: true,
		},
		{
			name:      "public project with members-only issues, merge request note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionMergeRequestNoteBody,
			project:   publicProjectForNonMember(gitlabLib.PrivateAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
			expectDM:  true,
		},
		{
			name:      "public project with all features enabled, issue note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionIssueNoteBody,
			project:   publicProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
			expectDM:  true,
		},
		{
			name:      "public project with all features enabled, merge request note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionMergeRequestNoteBody,
			project:   publicProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
			expectDM:  true,
		},
		{
			name:      "public project with all features enabled, push commit message, recipient is not a member",
			eventType: gitlabLib.EventTypePush,
			body:      mentionPushBody,
			project:   publicProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
			expectDM:  true,
		},
		{
			name:      "public project with all features enabled, confidential note, recipient is not a member",
			eventType: gitlabLib.EventConfidentialNote,
			body:      mentionConfidentialNoteBody,
			project:   publicProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
		},
		{
			name:      "internal project with all features enabled, issue note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionIssueNoteBody,
			project:   internalProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
			expectDM:  true,
		},
		{
			name:      "internal project with all features enabled, merge request note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionMergeRequestNoteBody,
			project:   internalProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
			expectDM:  true,
		},
		{
			name:      "internal project with all features enabled, push commit message, recipient is not a member",
			eventType: gitlabLib.EventTypePush,
			body:      mentionPushBody,
			project:   internalProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
			expectDM:  true,
		},
		{
			name:      "internal project with members-only issues, issue note, recipient is not a member",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionIssueNoteBody,
			project:   internalProjectForNonMember(gitlabLib.PrivateAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
		},
		{
			name:      "internal project with members-only repository, push commit message, recipient is not a member",
			eventType: gitlabLib.EventTypePush,
			body:      mentionPushBody,
			project:   internalProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.PrivateAccessControl),
		},
		{
			name:      "internal project with all features enabled, confidential note, recipient is not a member",
			eventType: gitlabLib.EventConfidentialNote,
			body:      mentionConfidentialNoteBody,
			project:   internalProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl),
		},
		{
			name:      "internal project with all features enabled, confidential issue note, recipient is only a guest",
			eventType: gitlabLib.EventTypeNote,
			body:      mentionConfidentialIssueNoteBody,
			project: func() *gitlabLib.Project {
				project := internalProjectForNonMember(gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl, gitlabLib.EnabledAccessControl)
				project.Permissions = projectWithAccess(gitlabLib.GuestPermissions).Permissions
				return project
			}(),
		},
		{
			name:       "internal project, recipient cannot load the project",
			eventType:  gitlabLib.EventTypeNote,
			body:       mentionIssueNoteBody,
			projectErr: errProjectNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, dmed := setupMentionWebhookPlugin(t, []string{mentioned}, func(string) (*gitlabLib.Project, error) {
				return test.project, test.projectErr
			})

			postMentionWebhook(t, p, test.eventType, fmt.Sprintf(test.body, mentioned))

			if test.expectDM {
				assert.Equal(t, []string{mentioned + "-mm-id"}, dmed())
			} else {
				assert.Empty(t, dmed(), "mention DM must not reach a user without access to the content")
			}
		})
	}
}

func TestHandleWebhookMentionDMGoesToFullUsername(t *testing.T) {
	tests := []struct {
		name      string
		mentioned string
		truncated string
	}{
		{name: "username with a dot", mentioned: "john.smith", truncated: "john"},
		{name: "username with an underscore", mentioned: "john_doe", truncated: "john"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, dmed := setupMentionWebhookPlugin(t, []string{test.mentioned, test.truncated}, func(string) (*gitlabLib.Project, error) {
				return projectWithAccess(gitlabLib.DeveloperPermissions), nil
			})

			postMentionWebhook(t, p, gitlabLib.EventTypeNote, fmt.Sprintf(mentionIssueNoteBody, test.mentioned))

			assert.Equal(t, []string{test.mentioned + "-mm-id"}, dmed())
		})
	}
}

// GitLab's project hook attributes carry only visibility_level, never the
// visibility string, so the private-repo gate must honor visibility_level.
func TestHandleWebhookPrivateRepoDisabledHonorsVisibilityLevel(t *testing.T) {
	const (
		recipient    = "mallory"
		privateLevel = webhook.PrivateVisibilityLevel
		publicLevel  = webhook.PublicVisibilityLevel
	)

	const (
		//nolint:misspell // noteable is how GitLab spells it in their API
		issueNoteBody = `{"object_kind":"note","event_type":"note","user":{"username":"sender"},
			"project":{"path_with_namespace":"group/project","visibility_level":%[2]d},
			"object_attributes":{"noteable_type":"Issue","note":"secret plan, cc @%[1]s","description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/issues/1#note_1"},
			"issue":{"iid":1,"confidential":false}}`
		//nolint:misspell // noteable is how GitLab spells it in their API
		mergeRequestNoteBody = `{"object_kind":"note","event_type":"note","user":{"username":"sender"},
			"project":{"path_with_namespace":"group/project","visibility_level":%[2]d},
			"object_attributes":{"noteable_type":"MergeRequest","note":"secret plan, cc @%[1]s","description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/merge_requests/1#note_1"},
			"merge_request":{"iid":1}}`
		issueBody = `{"object_kind":"issue","event_type":"issue","user":{"username":"sender"},
			"project":{"path_with_namespace":"group/project","visibility_level":%[2]d},
			"object_attributes":{"iid":1,"action":"close","confidential":false,"description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/issues/1"}}`
		mergeRequestBody = `{"object_kind":"merge_request","event_type":"merge_request","user":{"username":"sender"},
			"project":{"path_with_namespace":"group/project","visibility_level":%[2]d},
			"object_attributes":{"iid":1,"state":"opened","action":"open","description":"secret plan, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/merge_requests/1","target":{"path_with_namespace":"group/project","visibility_level":%[2]d}},
			"repository":{"homepage":"https://gitlab.example.com/group/project"}}`
		pushBody = `{"object_kind":"push","user_name":"sender","user_username":"sender",
			"project":{"path_with_namespace":"group/project","visibility_level":%[2]d,"web_url":"https://gitlab.example.com/group/project"},
			"commits":[{"id":"abc123","message":"secret fix, cc @%[1]s","url":"https://gitlab.example.com/group/project/-/commit/abc123"}],
			"total_commits_count":1}`
		tagBody = `{"object_kind":"tag_push","ref":"refs/tags/v1","user_name":"sender","user_username":"sender",
			"message":"secret release, cc @%[1]s",
			"project":{"path_with_namespace":"group/project","visibility_level":%[2]d,"web_url":"https://gitlab.example.com/group/project"}}`
		failedPipelineBody = `{"object_kind":"pipeline","user":{"username":"%[1]s"},
			"project":{"path_with_namespace":"group/project","visibility_level":%[2]d,"web_url":"https://gitlab.example.com/group/project"},
			"object_attributes":{"id":7,"status":"failed","source":"push"},
			"commit":{"message":"secret fix","url":"https://gitlab.example.com/group/project/-/commit/abc123"}}`
	)

	tests := []struct {
		name      string
		eventType gitlabLib.EventType
		body      string
	}{
		{name: "issue note", eventType: gitlabLib.EventTypeNote, body: issueNoteBody},
		{name: "merge request note", eventType: gitlabLib.EventTypeNote, body: mergeRequestNoteBody},
		{name: "issue", eventType: gitlabLib.EventTypeIssue, body: issueBody},
		{name: "merge request", eventType: gitlabLib.EventTypeMergeRequest, body: mergeRequestBody},
		{name: "push", eventType: gitlabLib.EventTypePush, body: pushBody},
		{name: "tag", eventType: gitlabLib.EventTypeTagPush, body: tagBody},
		{name: "failed pipeline", eventType: gitlabLib.EventTypePipeline, body: failedPipelineBody},
	}

	for _, test := range tests {
		for _, visibility := range []struct {
			level    int
			expectDM bool
		}{
			{level: privateLevel, expectDM: false},
			{level: publicLevel, expectDM: true},
		} {
			t.Run(fmt.Sprintf("%s, visibility_level %d", test.name, visibility.level), func(t *testing.T) {
				p, dmed := setupMentionWebhookPlugin(t, []string{recipient}, func(string) (*gitlabLib.Project, error) {
					return projectWithAccess(gitlabLib.DeveloperPermissions), nil
				})
				p.configuration.EnablePrivateRepo = false

				postMentionWebhook(t, p, test.eventType, fmt.Sprintf(test.body, recipient, visibility.level))

				if visibility.expectDM {
					assert.Equal(t, []string{recipient + "-mm-id"}, dmed())
				} else {
					assert.Empty(t, dmed(), "private project content must not be delivered while private repositories are disabled")
				}
			})
		}
	}
}
