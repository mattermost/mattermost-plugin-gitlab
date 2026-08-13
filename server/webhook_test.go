// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	gitlabLib "github.com/xanzy/go-gitlab"

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
	return nil, []string{}, nil
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
