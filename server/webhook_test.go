package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	gitlabLib "github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/webhook"
)

type fakeWebhookHandler struct{}

func (fakeWebhookHandler) HandleIssue(event *gitlabLib.IssueEvent) ([]*webhook.HandleWebhook, error) {
	return []*webhook.HandleWebhook{{
		Message: "hello",
		From:    "test",
		ToUsers: []string{"unknown"},
	}}, nil
}
func (fakeWebhookHandler) HandleMergeRequest(event *gitlabLib.MergeEvent) ([]*webhook.HandleWebhook, error) {
	return []*webhook.HandleWebhook{{
		Message:    "hello",
		From:       "test",
		ToChannels: []string{"town-square"},
	}}, nil
}
func (fakeWebhookHandler) HandleIssueComment(event *gitlabLib.IssueCommentEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}
func (fakeWebhookHandler) HandleMergeRequestComment(event *gitlabLib.MergeCommentEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}
func (fakeWebhookHandler) HandlePipeline(event *gitlabLib.PipelineEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}
func (fakeWebhookHandler) HandleTag(event *gitlabLib.TagEvent) ([]*webhook.HandleWebhook, error) {
	return nil, nil
}
func (fakeWebhookHandler) HandlePush(event *gitlabLib.PushEvent) ([]*webhook.HandleWebhook, error) {
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
	mock.On("LogError", "can't parse webhook", "err", "unexpected event type: ", "header", "", "event", "{}").Return(nil)
	p.SetAPI(mock)
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	w := httptest.NewRecorder()
	p.handleWebhook(w, req)
	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mock.AssertCalled(t, "LogError", "can't parse webhook", "err", "unexpected event type: ", "header", "", "event", "{}")
}

func TestHandleWebhookWithKnowAuthorButUnknowToUser(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: fakeWebhookHandler{}}

	mock := &plugintest.API{}
	mock.On("KVGet", "test_gitlabusername").Return([]byte("1"), nil).Once()
	mock.On("KVGet", "unknown_gitlabusername").Return(nil, nil).Once()
	mock.On("PublishWebSocketEvent", WsEventRefresh, map[string]interface{}(nil), &model.WebsocketBroadcast{UserId: "1"}).Return(nil).Once()
	mock.On("LogInfo", "new msg", "message", "hello", "from", "test").Return(nil)
	mock.On("LogInfo", "userFrom", "from", "1").Return(nil)
	p.SetAPI(mock)

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
	mock.AssertCalled(t, "PublishWebSocketEvent", WsEventRefresh, map[string]interface{}(nil), &model.WebsocketBroadcast{UserId: "1"})
	mock.AssertNumberOfCalls(t, "PublishWebSocketEvent", 1)
}

func TestHandleWebhookToChannel(t *testing.T) {
	p := &Plugin{configuration: &configuration{WebhookSecret: "secret"}, WebhookHandler: fakeWebhookHandler{}}

	mock := &plugintest.API{}
	mock.On("KVGet", "test_gitlabusername").Return([]byte("1"), nil).Once()
	mock.On("PublishWebSocketEvent", WsEventRefresh, map[string]interface{}(nil), &model.WebsocketBroadcast{UserId: "1"}).Return(nil).Once()
	mock.On("LogInfo", "new msg", "message", "hello", "from", "test").Return(nil)
	mock.On("LogInfo", "userFrom", "from", "1").Return(nil)
	mock.On("CreatePost", &model.Post{Id: "", CreateAt: 0, UpdateAt: 0, EditAt: 0, DeleteAt: 0, IsPinned: false, UserId: "", ChannelId: "town-square", RootId: "", OriginalId: "", Message: "hello", MessageSource: "", Type: "", Hashtags: "", Filenames: model.StringArray(nil), FileIds: model.StringArray(nil), PendingPostId: "", HasReactions: false, Metadata: (*model.PostMetadata)(nil)}).Return(nil, nil)
	p.SetAPI(mock)

	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"user": {"username":"test"}}`))
	req.Header.Add("X-Gitlab-Token", "secret")
	req.Header.Add("X-Gitlab-Event", string(gitlabLib.EventTypeMergeRequest))
	w := httptest.NewRecorder()

	p.handleWebhook(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mock.AssertCalled(t, "KVGet", "test_gitlabusername")
	mock.AssertNumberOfCalls(t, "KVGet", 1)
	mock.AssertCalled(t, "PublishWebSocketEvent", WsEventRefresh, map[string]interface{}(nil), &model.WebsocketBroadcast{UserId: "1"})
	mock.AssertNumberOfCalls(t, "PublishWebSocketEvent", 1)
	mock.AssertNumberOfCalls(t, "CreatePost", 1)
}
