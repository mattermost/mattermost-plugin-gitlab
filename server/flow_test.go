// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/flow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitLabAPI "github.com/xanzy/go-gitlab"
	"go.uber.org/mock/gomock"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	mocks "github.com/mattermost/mattermost-plugin-gitlab/server/gitlab/mocks"
)

func TestSubmitWebhook(t *testing.T) {
	const pluginWebhookURL = "https://example.com/" + inboundWebhookURL

	newFlowManager := func(mockedClient *mocks.MockGitlab) *FlowManager {
		p := new(Plugin)
		p.gitlabClient = mockedClient
		p.client = newAddWebhookTestClient(p, "https://example.com")
		return &FlowManager{
			client:           p.client,
			getConfiguration: p.getConfiguration,
			getGitlabUserInfoByMattermostID: func(string) (*gitlab.UserInfo, *APIErrorResponse) {
				return &gitlab.UserInfo{}, nil
			},
			getGitlabClient: p.getGitlabClient,
			useGitlabClient: p.useGitlabClient,
			createHook:      p.createHook,
		}
	}

	t.Run("succeeds without creating a webhook when the project already has one", func(t *testing.T) {
		mockedClient := mocks.NewMockGitlab(gomock.NewController(t))
		mockedClient.EXPECT().ResolveNamespaceAndProject(gomock.Any(), gomock.Any(), gomock.Any(), "group/project", gomock.Any()).Return("group", "project", nil)
		mockedClient.EXPECT().GetProjectHooks(gomock.Any(), gomock.Any(), gomock.Any(), "group", "project").Return([]*gitlab.WebhookInfo{{URL: pluginWebhookURL}}, nil)
		mockedClient.EXPECT().GetGroupHooks(gomock.Any(), gomock.Any(), gomock.Any(), "group").Return(nil, nil)
		fm := newFlowManager(mockedClient)

		next, state, errs, err := fm.submitWebhook(&flow.Flow{UserID: "user-id"}, map[string]any{"namespace": "group/project"})

		require.NoError(t, err)
		assert.Equal(t, stepWebhookConfirmation, next)
		assert.Nil(t, state)
		assert.Nil(t, errs)
	})

	t.Run("creates a webhook when the project has none", func(t *testing.T) {
		mockedClient := mocks.NewMockGitlab(gomock.NewController(t))
		mockedClient.EXPECT().ResolveNamespaceAndProject(gomock.Any(), gomock.Any(), gomock.Any(), "group/project", gomock.Any()).Return("group", "project", nil)
		mockedClient.EXPECT().GetProjectHooks(gomock.Any(), gomock.Any(), gomock.Any(), "group", "project").Return(nil, nil)
		mockedClient.EXPECT().GetGroupHooks(gomock.Any(), gomock.Any(), gomock.Any(), "group").Return(nil, nil)
		mockedClient.EXPECT().GetProject(gomock.Any(), gomock.Any(), gomock.Any(), "group", "project").Return(&gitLabAPI.Project{ID: 4}, nil)
		mockedClient.EXPECT().NewProjectHook(gomock.Any(), gomock.Any(), gomock.Any(), 4, gomock.Any()).Return(&gitlab.WebhookInfo{URL: pluginWebhookURL}, nil)
		fm := newFlowManager(mockedClient)

		next, _, _, err := fm.submitWebhook(&flow.Flow{UserID: "user-id"}, map[string]any{"namespace": "group/project"})

		require.NoError(t, err)
		assert.Equal(t, stepWebhookConfirmation, next)
	})
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
