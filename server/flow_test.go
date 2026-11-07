// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/flow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestFlowManager(t *testing.T) (*FlowManager, *plugintest.API) {
	t.Helper()
	siteURL := "https://mattermost.example.com"
	mmConfig := &model.Config{}
	mmConfig.ServiceSettings.SiteURL = &siteURL

	api := &plugintest.API{}
	api.On("GetConfig").Return(mmConfig)
	api.On("LogDebug", mock.Anything).Return(nil).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("LogWarn", mock.Anything).Return(nil).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	var kvMu sync.RWMutex
	kvStore := make(map[string][]byte)

	api.On("KVSet", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		kvMu.Lock()
		defer kvMu.Unlock()
		key := args.Get(0).(string)
		val := args.Get(1).([]byte)
		kvStore[key] = val
	}).Return(true, nil).Maybe()

	api.On("KVSetWithOptions", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		kvMu.Lock()
		defer kvMu.Unlock()
		key := args.Get(0).(string)
		val := args.Get(1).([]byte)
		kvStore[key] = val
	}).Return(true, nil).Maybe()

	api.On("KVGet", mock.Anything).Return(func(key string) []byte {
		kvMu.RLock()
		defer kvMu.RUnlock()
		return kvStore[key]
	}, nil).Maybe()

	api.On("KVDelete", mock.Anything).Run(func(args mock.Arguments) {
		kvMu.Lock()
		defer kvMu.Unlock()
		delete(kvStore, args.Get(0).(string))
	}).Return(nil).Maybe()

	api.On("GetDirectChannel", mock.Anything, mock.Anything).Return(&model.Channel{Id: "dm-channel"}, nil).Maybe()
	api.On("CreatePost", mock.Anything).Return(&model.Post{Id: "post-id"}, nil).Maybe()
	api.On("UpdatePost", mock.Anything).Return(&model.Post{Id: "post-id"}, nil).Maybe()

	p := &Plugin{
		configuration: &configuration{
			GitlabURL: "https://gitlab.example.com",
		},
		BotUserID: "bot-user-id",
		router:    mux.NewRouter(),
	}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	fm, err := p.NewFlowManager()
	require.NoError(t, err)
	return fm, api
}

func TestStepAfterDefaultInstanceDeclined(t *testing.T) {
	t.Run("continues to OAuth when the configured instance already is the default", func(t *testing.T) {
		fm := &FlowManager{
			getConfiguration: func() *configuration {
				return &configuration{DefaultInstanceName: "production"}
			},
		}

		next, state, err := fm.stepAfterDefaultInstanceDeclined("production")

		require.NoError(t, err)
		assert.Equal(t, stepOAuthConnect, next)
		assert.Nil(t, state)
	})

	t.Run("stops the wizard when another instance remains the default", func(t *testing.T) {
		fm := &FlowManager{
			getConfiguration: func() *configuration {
				return &configuration{DefaultInstanceName: "production"}
			},
		}

		next, state, err := fm.stepAfterDefaultInstanceDeclined("staging")

		require.NoError(t, err)
		assert.Equal(t, stepDefaultInstanceDeclined, next)
		assert.Equal(t, flow.State{
			keyInstanceName:        "staging",
			keyDefaultInstanceName: "production",
		}, state)
	})
}

func TestAdvanceOnOAuthConnect(t *testing.T) {
	t.Run("advances oauthFlow when at stepOAuthConnect", func(t *testing.T) {
		fm, _ := setupTestFlowManager(t)
		userID := "user-oauth-test"

		err := fm.StartOauthWizard(userID)
		require.NoError(t, err)

		err = fm.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		current, err := fm.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepOAuthConnect, current)

		advanced, err := fm.AdvanceOnOAuthConnect(userID)
		require.NoError(t, err)
		assert.True(t, advanced)

		current, err = fm.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepWebhookQuestion, current)
	})

	t.Run("advances setupFlow when at stepOAuthConnect", func(t *testing.T) {
		fm, _ := setupTestFlowManager(t)
		userID := "user-setup-test"

		err := fm.StartSetupWizard(userID, "")
		require.NoError(t, err)

		err = fm.setupFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		current, err := fm.setupFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepOAuthConnect, current)

		advanced, err := fm.AdvanceOnOAuthConnect(userID)
		require.NoError(t, err)
		assert.True(t, advanced)

		current, err = fm.setupFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepWebhookQuestion, current)
	})

	t.Run("prioritizes oauthFlow over setupFlow when both are at stepOAuthConnect", func(t *testing.T) {
		fm, _ := setupTestFlowManager(t)
		userID := "user-dual-flow-test"

		// Start setupFlow and move to stepOAuthConnect
		err := fm.StartSetupWizard(userID, "")
		require.NoError(t, err)
		err = fm.setupFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		// Start oauthFlow and move to stepOAuthConnect
		err = fm.StartOauthWizard(userID)
		require.NoError(t, err)
		err = fm.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		advanced, err := fm.AdvanceOnOAuthConnect(userID)
		require.NoError(t, err)
		assert.True(t, advanced)

		// oauthFlow must be advanced to stepWebhookQuestion
		oauthStep, err := fm.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepWebhookQuestion, oauthStep)

		// setupFlow remains at stepOAuthConnect
		setupStep, err := fm.setupFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepOAuthConnect, setupStep)
	})

	t.Run("returns false when no flow is active", func(t *testing.T) {
		fm, _ := setupTestFlowManager(t)

		advanced, err := fm.AdvanceOnOAuthConnect("unknown-user")
		require.NoError(t, err)
		assert.False(t, advanced)
	})

	t.Run("returns false when flow is at a different step", func(t *testing.T) {
		fm, _ := setupTestFlowManager(t)
		userID := "user-other-step"

		err := fm.StartOauthWizard(userID)
		require.NoError(t, err)

		current, err := fm.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepGitlabURL, current)

		advanced, err := fm.AdvanceOnOAuthConnect(userID)
		require.NoError(t, err)
		assert.False(t, advanced)

		current, err = fm.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepGitlabURL, current)
	})

	t.Run("advances specifically bound flow when specified", func(t *testing.T) {
		fm, _ := setupTestFlowManager(t)
		userID := "user-bound-flow-test"

		// Start setupFlow and move to stepOAuthConnect
		err := fm.StartSetupWizard(userID, "")
		require.NoError(t, err)
		err = fm.setupFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		// Start oauthFlow and move to stepOAuthConnect
		err = fm.StartOauthWizard(userID)
		require.NoError(t, err)
		err = fm.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		// Specifically advance setupFlow
		advanced, err := fm.AdvanceOnOAuthConnect(userID, "setup")
		require.NoError(t, err)
		assert.True(t, advanced)

		// setupFlow was advanced
		setupStep, err := fm.setupFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepWebhookQuestion, setupStep)

		// oauthFlow remained at stepOAuthConnect
		oauthStep, err := fm.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepOAuthConnect, oauthStep)
	})

	t.Run("preserves lookup errors and returns them", func(t *testing.T) {
		fm, api := setupTestFlowManager(t)
		userID := "user-lookup-err-test"

		// Start oauthFlow
		err := fm.StartOauthWizard(userID)
		require.NoError(t, err)

		// Simulate a storage failure on KVGet
		appErr := model.NewAppError("KVGet", "plugin.kv_get.error", nil, "database connection refused", http.StatusInternalServerError)
		api.ExpectedCalls = nil
		api.On("KVGet", mock.Anything).Return(nil, appErr)
		api.On("LogWarn", mock.Anything).Return(nil).Maybe()
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		advanced, err := fm.AdvanceOnOAuthConnect(userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current step")
		assert.False(t, advanced)
	})
}

func setupPluginForOAuthFlowTest(t *testing.T, isSysAdmin bool) (*Plugin, *plugintest.API, *[]*model.Post, func(func(key string) *model.AppError)) {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "mock-token",
				"token_type":   "Bearer",
				"expires_in":   7200,
			})
		case "/api/v4/user":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       123,
				"username": "gitlabuser",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)

	siteURL := "https://mattermost.example.com"
	mmConfig := &model.Config{}
	mmConfig.ServiceSettings.SiteURL = &siteURL

	api := &plugintest.API{}
	api.On("GetConfig").Return(mmConfig)
	api.On("LogDebug", mock.Anything).Return(nil).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("LogWarn", mock.Anything).Return(nil).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything).Return().Maybe()

	var kvMu sync.RWMutex
	kvStore := make(map[string][]byte)

	var kvErrMu sync.RWMutex
	var injectKVGetErr func(key string) *model.AppError

	api.On("KVSet", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		kvMu.Lock()
		defer kvMu.Unlock()
		key := args.Get(0).(string)
		val := args.Get(1).([]byte)
		kvStore[key] = val
	}).Return(true, nil).Maybe()

	api.On("KVSetWithOptions", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		kvMu.Lock()
		defer kvMu.Unlock()
		key := args.Get(0).(string)
		val := args.Get(1).([]byte)
		kvStore[key] = val
	}).Return(true, nil).Maybe()

	api.On("KVGet", mock.Anything).Return(func(key string) []byte {
		kvMu.RLock()
		defer kvMu.RUnlock()
		return kvStore[key]
	}, func(key string) *model.AppError {
		kvErrMu.RLock()
		defer kvErrMu.RUnlock()
		if injectKVGetErr != nil {
			return injectKVGetErr(key)
		}
		return nil
	}).Maybe()

	api.On("KVDelete", mock.Anything).Run(func(args mock.Arguments) {
		kvMu.Lock()
		defer kvMu.Unlock()
		delete(kvStore, args.Get(0).(string))
	}).Return(nil).Maybe()

	roles := ""
	if isSysAdmin {
		roles = "system_admin"
	}
	api.On("GetUser", mock.Anything).Return(&model.User{Roles: roles}, nil).Maybe()
	api.On("GetDirectChannel", mock.Anything, mock.Anything).Return(&model.Channel{Id: "dm-channel"}, nil).Maybe()

	var createdPosts []*model.Post
	var postMu sync.Mutex
	api.On("CreatePost", mock.Anything).Run(func(args mock.Arguments) {
		post := args.Get(0).(*model.Post).Clone()
		postMu.Lock()
		createdPosts = append(createdPosts, post)
		postMu.Unlock()
	}).Return(&model.Post{Id: "post-id"}, nil).Maybe()

	api.On("UpdatePost", mock.Anything).Return(&model.Post{Id: "post-id"}, nil).Maybe()

	p := &Plugin{
		configuration: &configuration{
			GitlabURL:               ts.URL,
			GitlabOAuthClientID:     "client_id",
			GitlabOAuthClientSecret: "client_secret",
			EncryptionKey:           "aaaaaaaaaaaaaaaa",
		},
		BotUserID: "bot-user-id",
	}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	p.initializeAPI()
	p.oauthBroker = NewOAuthBroker(func(_ OAuthCompleteEvent) {})

	fm, err := p.NewFlowManager()
	require.NoError(t, err)
	p.flowManager = fm

	setErrFn := func(fn func(key string) *model.AppError) {
		kvErrMu.Lock()
		defer kvErrMu.Unlock()
		injectKVGetErr = fn
	}

	return p, api, &createdPosts, setErrFn
}

func TestOAuthCompleteCallbackFlowAdvancement(t *testing.T) {
	t.Run("advances bound oauthFlow and suppresses welcome message", func(t *testing.T) {
		p, _, createdPosts, _ := setupPluginForOAuthFlowTest(t, true)
		userID := "user1111111111111111111111"

		err := p.flowManager.StartOauthWizard(userID)
		require.NoError(t, err)
		err = p.flowManager.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		state := "abcdefghijklmno_" + userID
		_, err = p.client.KV.Set(state, []byte(state+"#oauth"))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=testcode&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", userID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusOK, result.StatusCode)
		body, _ := io.ReadAll(result.Body)
		assert.Contains(t, string(body), "Completed connecting to GitLab")

		currentStep, err := p.flowManager.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepWebhookQuestion, currentStep)

		for _, post := range *createdPosts {
			assert.NotEqual(t, "custom_git_welcome", post.Type)
			assert.NotContains(t, post.Message, "Welcome to the Mattermost GitLab Plugin")
		}
	})

	t.Run("advances bound setupFlow and preserves oauthFlow state", func(t *testing.T) {
		p, _, createdPosts, _ := setupPluginForOAuthFlowTest(t, true)
		userID := "user2222222222222222222222"

		err := p.flowManager.StartSetupWizard(userID, "")
		require.NoError(t, err)
		err = p.flowManager.setupFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		err = p.flowManager.StartOauthWizard(userID)
		require.NoError(t, err)
		err = p.flowManager.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		state := "abcdefghijklmno_" + userID
		_, err = p.client.KV.Set(state, []byte(state+"#setup"))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=testcode&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", userID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusOK, result.StatusCode)

		setupStep, err := p.flowManager.setupFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepWebhookQuestion, setupStep)

		oauthStep, err := p.flowManager.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepOAuthConnect, oauthStep)

		for _, post := range *createdPosts {
			assert.NotEqual(t, "custom_git_welcome", post.Type)
		}
	})

	t.Run("advances setupFlow on legacy parameterless callback", func(t *testing.T) {
		p, _, createdPosts, _ := setupPluginForOAuthFlowTest(t, true)
		userID := "user3333333333333333333333"

		err := p.flowManager.StartSetupWizard(userID, "")
		require.NoError(t, err)
		err = p.flowManager.setupFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		state := "abcdefghijklmno_" + userID
		_, err = p.client.KV.Set(state, []byte(state))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=testcode&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", userID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusOK, result.StatusCode)

		setupStep, err := p.flowManager.setupFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Equal(t, stepWebhookQuestion, setupStep)

		for _, post := range *createdPosts {
			assert.NotEqual(t, "custom_git_welcome", post.Type)
		}
	})

	t.Run("posts welcome message when no setup wizard is active", func(t *testing.T) {
		p, _, createdPosts, _ := setupPluginForOAuthFlowTest(t, false)
		userID := "user4444444444444444444444"

		state := "abcdefghijklmno_" + userID
		_, err := p.client.KV.Set(state, []byte(state))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=testcode&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", userID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusOK, result.StatusCode)

		var welcomePost *model.Post
		for _, post := range *createdPosts {
			if post.Type == "custom_git_welcome" {
				welcomePost = post
				break
			}
		}
		require.NotNil(t, welcomePost, "expected welcome message post when no setup wizard was active")
		assert.Contains(t, welcomePost.Message, "Welcome to the Mattermost GitLab Plugin")
	})

	t.Run("handles advancement error for sysadmin with setup webhook command", func(t *testing.T) {
		p, _, createdPosts, setKVGetErr := setupPluginForOAuthFlowTest(t, true)
		userID := "user5555555555555555555555"

		err := p.flowManager.StartOauthWizard(userID)
		require.NoError(t, err)
		err = p.flowManager.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		setKVGetErr(func(key string) *model.AppError {
			if strings.Contains(key, "_flow-") {
				return model.NewAppError("KVGet", "plugin.kv_get.error", nil, "database connection refused", http.StatusInternalServerError)
			}
			return nil
		})

		state := "abcdefghijklmno_" + userID
		_, err = p.client.KV.Set(state, []byte(state+"#oauth"))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=testcode&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", userID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusOK, result.StatusCode)

		for _, post := range *createdPosts {
			assert.NotEqual(t, "custom_git_welcome", post.Type)
		}

		var recoveryPost *model.Post
		for _, post := range *createdPosts {
			if strings.Contains(post.Message, "setup could not continue automatically") {
				recoveryPost = post
				break
			}
		}
		require.NotNil(t, recoveryPost, "expected recovery DM post")
		assert.Contains(t, recoveryPost.Message, "To complete webhook configuration, run `/gitlab setup webhook`.")
		assert.NotContains(t, recoveryPost.Message, "Please contact a System Admin")
	})

	t.Run("handles advancement error for delegate with contact admin instruction", func(t *testing.T) {
		p, _, createdPosts, setKVGetErr := setupPluginForOAuthFlowTest(t, false)
		userID := "user6666666666666666666666"

		err := p.flowManager.StartOauthWizard(userID)
		require.NoError(t, err)
		err = p.flowManager.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		setKVGetErr(func(key string) *model.AppError {
			if strings.Contains(key, "_flow-") {
				return model.NewAppError("KVGet", "plugin.kv_get.error", nil, "database connection refused", http.StatusInternalServerError)
			}
			return nil
		})

		state := "abcdefghijklmno_" + userID
		_, err = p.client.KV.Set(state, []byte(state+"#oauth"))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/oauth/complete?code=testcode&state="+state, nil)
		r.Header.Set("Mattermost-User-ID", userID)

		p.ServeHTTP(nil, w, r)

		result := w.Result()
		defer func() { _ = result.Body.Close() }()
		assert.Equal(t, http.StatusOK, result.StatusCode)

		for _, post := range *createdPosts {
			assert.NotEqual(t, "custom_git_welcome", post.Type)
		}

		var recoveryPost *model.Post
		for _, post := range *createdPosts {
			if strings.Contains(post.Message, "setup could not continue automatically") {
				recoveryPost = post
				break
			}
		}
		require.NotNil(t, recoveryPost, "expected recovery DM post")
		assert.Contains(t, recoveryPost.Message, "Please contact a System Admin with the required GitLab permissions to complete webhook configuration.")
		assert.NotContains(t, recoveryPost.Message, "/gitlab setup webhook")
	})
}

func TestOAuthFlowWebhookSteps(t *testing.T) {
	t.Run("stepWebhookWarning terminates oauthFlow cleanly", func(t *testing.T) {
		fm, _ := setupTestFlowManager(t)
		userID := "user-warning-test"

		err := fm.StartOauthWizard(userID)
		require.NoError(t, err)

		err = fm.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		advanced, err := fm.AdvanceOnOAuthConnect(userID)
		require.NoError(t, err)
		assert.True(t, advanced)

		err = fm.oauthFlow.ForUser(userID).Go(stepWebhookWarning)
		require.NoError(t, err)

		// Terminal step finishes the flow
		current, err := fm.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Empty(t, current)
	})

	t.Run("stepWebhookConfirmation terminates oauthFlow cleanly", func(t *testing.T) {
		fm, _ := setupTestFlowManager(t)
		userID := "user-confirm-test"

		err := fm.StartOauthWizard(userID)
		require.NoError(t, err)

		err = fm.oauthFlow.ForUser(userID).Go(stepOAuthConnect)
		require.NoError(t, err)

		advanced, err := fm.AdvanceOnOAuthConnect(userID)
		require.NoError(t, err)
		assert.True(t, advanced)

		err = fm.oauthFlow.ForUser(userID).Go(stepWebhookConfirmation)
		require.NoError(t, err)

		// Terminal step finishes the flow
		current, err := fm.oauthFlow.ForUser(userID).GetCurrentStep()
		require.NoError(t, err)
		assert.Empty(t, current)
	})
}
