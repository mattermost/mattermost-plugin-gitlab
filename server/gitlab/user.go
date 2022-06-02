package gitlab

import (
	"context"

	"github.com/mattermost/mattermost-server/v6/model"
	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

const SettingButtonsTeam = "team"

type UserInfo struct {
	UserID              string
	Token               *oauth2.Token
	GitlabUsername      string
	GitlabUserID        int
	LastToDoPostAt      int64
	Settings            *UserSettings
	AllowedPrivateRepos bool
}

type UserSettings struct {
	SidebarButtons string `json:"sidebar_buttons"`
	DailyReminder  bool   `json:"daily_reminder"`
	Notifications  bool   `json:"notifications"`
}

func (g *gitlab) GetCurrentUser(ctx context.Context, userID string, token oauth2.Token) (*UserInfo, error) {
	client, err := g.GitlabConnect(token)
	if err != nil {
		return nil, err
	}

	gitUser, resp, err := client.Users.CurrentUser(internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		UserID:         userID,
		GitlabUserID:   gitUser.ID,
		Token:          &token,
		GitlabUsername: gitUser.Username,
		LastToDoPostAt: model.GetMillis(),
		Settings: &UserSettings{
			SidebarButtons: SettingButtonsTeam,
			DailyReminder:  true,
			Notifications:  true,
		},
	}, nil
}

func (g *gitlab) GetUserDetails(ctx context.Context, user *UserInfo) (*internGitlab.User, error) {
	client, err := g.GitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	gitUser, resp, err := client.Users.CurrentUser(internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	return gitUser, nil
}
