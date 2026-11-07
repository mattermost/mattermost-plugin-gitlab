// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
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
