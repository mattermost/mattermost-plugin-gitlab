package gitlab

import (
	"github.com/mattermost/mattermost-server/model"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

const SETTING_BUTTONS_TEAM = "team"

type GitlabUserInfo struct {
	UserID              string
	Token               *oauth2.Token
	GitlabUsername      string
	GitlabUserId        int
	LastToDoPostAt      int64
	Settings            *UserSettings
	AllowedPrivateRepos bool
}

type UserSettings struct {
	SidebarButtons string `json:"sidebar_buttons"`
	DailyReminder  bool   `json:"daily_reminder"`
	Notifications  bool   `json:"notifications"`
}

func (g *myGitlab) GetCurrentUser(userID string, token oauth2.Token) (*GitlabUserInfo, error) {
	client, err := g.connect(token)
	if err != nil {
		return nil, err
	}

	gitUser, _, err := client.Users.CurrentUser()
	if err != nil {
		return nil, err
	}

	return &GitlabUserInfo{
		UserID:         userID,
		GitlabUserId:   gitUser.ID,
		Token:          &token,
		GitlabUsername: gitUser.Username,
		LastToDoPostAt: model.GetMillis(),
		Settings: &UserSettings{
			SidebarButtons: SETTING_BUTTONS_TEAM,
			DailyReminder:  true,
			Notifications:  true,
		},
	}, nil
}

func (g *myGitlab) GetUserDetails(user *GitlabUserInfo) (*gitlab.User, error) {
	client, err := g.connect(*user.Token)
	if err != nil {
		return nil, err
	}

	gitUser, _, err := client.Users.CurrentUser()
	return gitUser, err
}
