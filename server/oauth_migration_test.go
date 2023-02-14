package main

import (
	"testing"

	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
)

func TestNotifyAllConnectedUsersToReconnect(t *testing.T) {
	p := new(Plugin)

	testAPI := &plugintest.API{}

	p.client = pluginapi.NewClient(testAPI, nil)
	p.poster = poster.NewPoster(&p.client.Post, "botid")

	userID := "someuser"
	botID := "botid"
	dmChannelID := "dm_channel_id"

	tokenKey := userID + "_gitlabtoken"
	usernameKey := "myusername_gitlabusername"
	gitlabIDKey := "myid_gitlabidusername"

	keys := []string{
		"invalid",
		tokenKey,
	}

	siteURL := "https://myserver.com"
	testAPI.On("GetConfig").Return(
		&model.Config{ServiceSettings: model.ServiceSettings{SiteURL: &siteURL}},
		nil,
	)

	testAPI.On("KVList", 0, 100).Return(keys, nil)
	testAPI.On("KVList", 1, 100).Return([]string{}, nil)

	testAPI.On("GetDirectChannel", botID, userID).Return(&model.Channel{Id: dmChannelID}, nil)

	expected := "An update for this integration requires you to reconnect your account. [Click here to link your GitLab account.](https://myserver.com/plugins/com.github.manland.mattermost-plugin-gitlab/oauth/connect)"
	testAPI.On("CreatePost", &model.Post{
		ChannelId: dmChannelID,
		UserId:    botID,
		Message:   expected,
	}).Return(&model.Post{}, nil)

	testAPI.On("KVSetWithOptions", tokenKey, []byte(nil), mock.Anything).Return(true, nil)
	testAPI.On("KVSetWithOptions", usernameKey, []byte(nil), mock.Anything).Return(true, nil)
	testAPI.On("KVSetWithOptions", gitlabIDKey, []byte(nil), mock.Anything).Return(true, nil)

	p.SetAPI(testAPI)

	err := p.notifyAllConnectedUsersToReconnect()
	require.NoError(t, err)
}
