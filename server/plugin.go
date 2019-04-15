package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/manland/go-gitlab"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"golang.org/x/oauth2"
)

const (
	GITLAB_TOKEN_KEY        = "_gitlabtoken"
	GITLAB_STATE_KEY        = "_gitlabstate"
	GITLAB_USERNAME_KEY     = "_gitlabusername"
	GITLAB_IDUSERNAME_KEY   = "_gitlabidusername"
	GITLAB_PRIVATE_REPO_KEY = "_gitlabprivate"
	WS_EVENT_CONNECT        = "gitlab_connect"
	WS_EVENT_DISCONNECT     = "gitlab_disconnect"
	WS_EVENT_REFRESH        = "gitlab_refresh"
	SETTING_BUTTONS_TEAM    = "team"
	SETTING_BUTTONS_CHANNEL = "channel"
	SETTING_BUTTONS_OFF     = "off"
	SETTING_NOTIFICATIONS   = "notifications"
	SETTING_REMINDERS       = "reminders"
	SETTING_ON              = "on"
	SETTING_OFF             = "off"
)

type Plugin struct {
	plugin.MattermostPlugin

	BotUserID string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

func (p *Plugin) gitlabConnect(token oauth2.Token) *gitlab.Client {
	config := p.getConfiguration()

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&token)
	tc := oauth2.NewClient(ctx, ts)

	if len(config.EnterpriseBaseURL) == 0 {
		return gitlab.NewOAuthClient(tc, token.AccessToken)
	}

	client := gitlab.NewOAuthClient(tc, token.AccessToken)
	if err := client.SetBaseURL(config.EnterpriseBaseURL); err != nil {
		mlog.Error(err.Error())
		return gitlab.NewOAuthClient(tc, token.AccessToken)
	}
	return client
}

func (p *Plugin) OnActivate() error {
	config := p.getConfiguration()

	if err := config.IsValid(); err != nil {
		return err
	}
	p.API.RegisterCommand(getCommand())
	user, err := p.API.GetUserByUsername(config.Username)
	if err != nil {
		mlog.Error(err.Error())
		return fmt.Errorf("Unable to find user with configured username: %v", config.Username)
	}

	p.BotUserID = user.Id

	return nil
}

func (p *Plugin) getOAuthConfig() *oauth2.Config {
	config := p.getConfiguration()

	authURL, _ := url.Parse("https://gitlab.com/")
	tokenURL, _ := url.Parse("https://gitlab.com/")
	if len(config.EnterpriseBaseURL) > 0 {
		authURL, _ = url.Parse(config.EnterpriseBaseURL)
		tokenURL, _ = url.Parse(config.EnterpriseBaseURL)
	}

	authURL.Path = path.Join(authURL.Path, "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "oauth", "token")

	return &oauth2.Config{
		ClientID:     config.GitlabOAuthClientID,
		ClientSecret: config.GitlabOAuthClientSecret,
		Scopes:       []string{"api", "read_user"},
		RedirectURL:  fmt.Sprintf("%s/plugins/%s/oauth/complete", *p.API.GetConfig().ServiceSettings.SiteURL, manifest.Id),
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL.String(),
			TokenURL: tokenURL.String(),
		},
	}
}

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

func (p *Plugin) storeGitlabUserInfo(info *GitlabUserInfo) error {
	config := p.getConfiguration()

	encryptedToken, err := encrypt([]byte(config.EncryptionKey), info.Token.AccessToken)
	if err != nil {
		return err
	}

	info.Token.AccessToken = encryptedToken

	jsonInfo, err := json.Marshal(info)
	if err != nil {
		return err
	}

	if err := p.API.KVSet(info.UserID+GITLAB_TOKEN_KEY, jsonInfo); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) getGitlabUserInfoByMattermostID(userID string) (*GitlabUserInfo, *APIErrorResponse) {
	config := p.getConfiguration()

	var userInfo GitlabUserInfo

	if infoBytes, err := p.API.KVGet(userID + GITLAB_TOKEN_KEY); err != nil || infoBytes == nil {
		return nil, &APIErrorResponse{ID: API_ERROR_ID_NOT_CONNECTED, Message: "Must connect user account to Gitlab first.", StatusCode: http.StatusBadRequest}
	} else if err := json.Unmarshal(infoBytes, &userInfo); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse token.", StatusCode: http.StatusInternalServerError}
	}

	unencryptedToken, err := decrypt([]byte(config.EncryptionKey), userInfo.Token.AccessToken)
	if err != nil {
		mlog.Error(err.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	userInfo.Token.AccessToken = unencryptedToken

	return &userInfo, nil
}

func (p *Plugin) storeGitlabToUserIDMapping(gitlabUsername, userID string) error {
	if err := p.API.KVSet(gitlabUsername+GITLAB_USERNAME_KEY, []byte(userID)); err != nil {
		return fmt.Errorf("Encountered error saving gitlab username mapping")
	}
	if err := p.API.KVSet(userID+GITLAB_IDUSERNAME_KEY, []byte(gitlabUsername)); err != nil {
		return fmt.Errorf("Encountered error saving gitlab id mapping")
	}
	return nil
}

func (p *Plugin) getGitlabToUserIDMapping(gitlabUsername string) string {
	userID, _ := p.API.KVGet(gitlabUsername + GITLAB_USERNAME_KEY)
	return string(userID)
}

func (p *Plugin) getGitlabIDToUsernameMapping(gitlabUserID string) string {
	gitlabUsername, err := p.API.KVGet(gitlabUserID + GITLAB_IDUSERNAME_KEY)
	if err != nil {
		p.API.LogError("can't get user id by login", "err", err)
		return ""
	}
	return string(gitlabUsername)
}

func (p *Plugin) disconnectGitlabAccount(userID string) {
	userInfo, _ := p.getGitlabUserInfoByMattermostID(userID)
	if userInfo == nil {
		return
	}

	p.API.KVDelete(userID + GITLAB_TOKEN_KEY)
	p.API.KVDelete(userInfo.GitlabUsername + GITLAB_USERNAME_KEY)

	if user, err := p.API.GetUser(userID); err == nil && user.Props != nil && len(user.Props["git_user"]) > 0 {
		delete(user.Props, "git_user")
		p.API.UpdateUser(user)
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_DISCONNECT,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) CreateBotDMPost(userID, message, postType string) *model.AppError {
	channel, err := p.API.GetDirectChannel(userID, p.BotUserID)
	if err != nil {
		mlog.Error("Couldn't get bot's DM channel", mlog.String("user_id", userID))
		return err
	}

	config := p.getConfiguration()

	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
		Props: map[string]interface{}{
			"from_webhook":      "true",
			"override_username": GITLAB_USERNAME,
			"override_icon_url": config.ProfileImageURL,
		},
	}

	if _, err := p.API.CreatePost(post); err != nil {
		mlog.Error(err.Error())
		return err
	}

	return nil
}

func (p *Plugin) PostToDo(info *GitlabUserInfo) {
	text, err := p.GetToDo(info, p.gitlabConnect(*info.Token))
	if err != nil {
		mlog.Error(err.Error())
		return
	}

	p.CreateBotDMPost(info.UserID, text, "custom_git_todo")
}

func (p *Plugin) GetToDo(user *GitlabUserInfo, client *gitlab.Client) (string, error) {
	notifications, _, err := client.Todos.ListTodos(&gitlab.ListTodosOptions{})
	if err != nil {
		return "", err
	}

	opened := "opened"
	scope := "all"

	issueResults, issueResponse, err := client.Issues.ListIssues(&gitlab.ListIssuesOptions{
		AssigneeID: &user.GitlabUserId,
		State:      &opened,
	})
	if err != nil {
		return "", err
	}

	yourPrs, yourPrsResponse, err := client.MergeRequests.ListMergeRequests(&gitlab.ListMergeRequestsOptions{
		AuthorID: &user.GitlabUserId,
		State:    &opened,
	})
	if err != nil {
		return "", err
	}

	yourAssignments, yourAssignmentsResponse, err := client.MergeRequests.ListMergeRequests(&gitlab.ListMergeRequestsOptions{
		AssigneeID: &user.GitlabUserId,
		State:      &opened,
		Scope:      &scope,
	})
	if err != nil {
		return "", err
	}

	text := "##### Unread Messages\n"

	notificationCount := 0
	notificationContent := ""
	for _, n := range notifications {
		if p.checkGroup(n.Project.NameWithNamespace) != nil {
			continue
		}
		notificationCount++
		notificationContent += fmt.Sprintf("* %v : [%v](%v)\n", n.ActionName, n.Target.Title, n.TargetURL)
	}

	if notificationCount == 0 {
		text += "You don't have any unread messages.\n"
	} else {
		text += fmt.Sprintf("You have %v unread messages:\n", notificationCount)
		text += notificationContent
	}

	text += "##### Review Requests\n"

	if yourAssignmentsResponse.TotalItems == 0 {
		text += "You don't have any pull requests awaiting your review.\n"
	} else {
		text += fmt.Sprintf("You have %v pull requests awaiting your review:\n", yourAssignmentsResponse.TotalItems)

		for _, pr := range yourAssignments {
			text += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
		}
	}

	text += "##### Assignments\n"

	if issueResponse.TotalItems == 0 {
		text += "You don't have any issues awaiting your dev.\n"
	} else {
		text += fmt.Sprintf("You have %v issues awaiting dev:\n", issueResponse.TotalItems)

		for _, pr := range issueResults {
			text += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
		}
	}

	text += "##### Your Open Pull Requests\n"

	if yourPrsResponse.TotalItems == 0 {
		text += "You don't have any open pull requests.\n"
	} else {
		text += fmt.Sprintf("You have %v open pull requests:\n", yourPrsResponse.TotalItems)

		for _, pr := range yourPrs {
			text += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
		}
	}

	return text, nil
}

func (p *Plugin) checkGroup(projectNameWithGroup string) error {
	config := p.getConfiguration()

	group := strings.TrimSpace(config.GitlabGroup)
	if group != "" && group != strings.Split(projectNameWithGroup, "/")[0] {
		return fmt.Errorf("Only repositories in the %v group are supported", config.GitlabGroup)
	}

	return nil
}

func (p *Plugin) sendRefreshEvent(userID string) {
	p.API.PublishWebSocketEvent(
		WS_EVENT_REFRESH,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}
