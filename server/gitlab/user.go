package gitlab

import (
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

func (g *gitlab) GetCurrentUser(userID string, token oauth2.Token) (*UserInfo, error) {
	client, err := g.gitlabConnect(token)
	if err != nil {
		return nil, err
	}

	gitUser, _, err := client.Users.CurrentUser()
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

func (g *gitlab) GetUserDetails(user *UserInfo) (*internGitlab.User, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	gitUser, _, err := client.Users.CurrentUser()
	return gitUser, err
}
