// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSetGitlabURL(t *testing.T) {
	config := &configuration{
		GitlabURL: "https://gitlab.com",
	}

	var savedConfig map[string]any
	api := &plugintest.API{}
	api.On("SavePluginConfig", mock.Anything).Run(func(args mock.Arguments) {
		savedConfig = args.Get(0).(map[string]any)
	}).Return(nil).Once()

	fm := &FlowManager{
		client:           pluginapi.NewClient(api, nil),
		getConfiguration: func() *configuration { return config },
	}

	err := fm.setGitlabURL("https://git.example.com")
	require.NoError(t, err)

	assert.Equal(t, "https://git.example.com", fm.gitlabURL)

	require.NotNil(t, savedConfig)
	assert.Equal(t, "https://git.example.com", savedConfig["gitlaburl"])

	assert.Equal(t, "https://gitlab.com", config.GitlabURL)

	api.AssertExpectations(t)
}

func TestSetGitlabURLSaveFailure(t *testing.T) {
	config := &configuration{
		GitlabURL: "https://gitlab.com",
	}

	api := &plugintest.API{}
	api.On("SavePluginConfig", mock.Anything).Return(model.NewAppError("SavePluginConfig", "app.plugin.config.app_error", nil, "", http.StatusInternalServerError)).Once()

	fm := &FlowManager{
		client:           pluginapi.NewClient(api, nil),
		getConfiguration: func() *configuration { return config },
	}

	err := fm.setGitlabURL("https://git.example.com")
	require.Error(t, err)

	assert.Empty(t, fm.gitlabURL)

	api.AssertExpectations(t)
}
