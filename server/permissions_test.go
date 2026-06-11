// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gitLabAPI "github.com/xanzy/go-gitlab"
	"go.uber.org/mock/gomock"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	mocks "github.com/mattermost/mattermost-plugin-gitlab/server/gitlab/mocks"
)

func newPermissionTestPlugin(t *testing.T, mockedClient *mocks.MockGitlab, user *model.User) *Plugin {
	t.Helper()
	p := new(Plugin)
	p.configuration = &configuration{EncryptionKey: testEncryptionKey}
	p.GitlabClient = mockedClient

	encryptedToken, _ := encrypt([]byte(testEncryptionKey), testGitlabToken)

	api := &plugintest.API{}
	api.On("KVGet", "user_id_usertoken").Return([]byte(encryptedToken), nil)
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	if user != nil {
		api.On("GetUser", "user_id").Return(user, nil)
	}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	return p
}

func TestHasGitlabPermission(t *testing.T) {
	info := &gitlab.UserInfo{UserID: "user_id", GitlabUserID: 7}

	t.Run("project: sufficient access", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockedClient := mocks.NewMockGitlab(mockCtrl)
		mockedClient.EXPECT().GetProjectAccessLevel(gomock.Any(), gomock.Any(), gomock.Any(), "group", "project").Return(gitLabAPI.MaintainerPermissions, nil)
		p := newPermissionTestPlugin(t, mockedClient, nil)

		allowed, err := p.hasGitlabPermission(context.Background(), info, "group", "project", actionManageWebhook)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("project: insufficient access", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockedClient := mocks.NewMockGitlab(mockCtrl)
		mockedClient.EXPECT().GetProjectAccessLevel(gomock.Any(), gomock.Any(), gomock.Any(), "group", "project").Return(gitLabAPI.DeveloperPermissions, nil)
		p := newPermissionTestPlugin(t, mockedClient, nil)

		allowed, err := p.hasGitlabPermission(context.Background(), info, "group", "project", actionManageWebhook)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("not a member is a definitive deny, not an error", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockedClient := mocks.NewMockGitlab(mockCtrl)
		mockedClient.EXPECT().GetProjectAccessLevel(gomock.Any(), gomock.Any(), gomock.Any(), "group", "project").Return(gitLabAPI.NoPermissions, gitlab.ErrNotFound)
		p := newPermissionTestPlugin(t, mockedClient, nil)

		allowed, err := p.hasGitlabPermission(context.Background(), info, "group", "project", actionManageWebhook)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("API failure falls back to System Admin: admin allowed", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockedClient := mocks.NewMockGitlab(mockCtrl)
		mockedClient.EXPECT().GetProjectAccessLevel(gomock.Any(), gomock.Any(), gomock.Any(), "group", "project").Return(gitLabAPI.NoPermissions, errors.New("gitlab unavailable"))
		p := newPermissionTestPlugin(t, mockedClient, &model.User{Id: "user_id", Roles: "system_admin system_user"})

		allowed, err := p.hasGitlabPermission(context.Background(), info, "group", "project", actionManageWebhook)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("API failure falls back to System Admin: non-admin denied", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockedClient := mocks.NewMockGitlab(mockCtrl)
		mockedClient.EXPECT().GetProjectAccessLevel(gomock.Any(), gomock.Any(), gomock.Any(), "group", "project").Return(gitLabAPI.NoPermissions, errors.New("gitlab unavailable"))
		p := newPermissionTestPlugin(t, mockedClient, &model.User{Id: "user_id", Roles: "system_user"})

		allowed, err := p.hasGitlabPermission(context.Background(), info, "group", "project", actionManageWebhook)
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("group: uses group access level lookup", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		mockedClient := mocks.NewMockGitlab(mockCtrl)
		mockedClient.EXPECT().GetGroupAccessLevel(gomock.Any(), gomock.Any(), gomock.Any(), "group").Return(gitLabAPI.OwnerPermissions, nil)
		p := newPermissionTestPlugin(t, mockedClient, nil)

		allowed, err := p.hasGitlabPermission(context.Background(), info, "group", "", actionManageWebhook)
		assert.NoError(t, err)
		assert.True(t, allowed)
	})
}
