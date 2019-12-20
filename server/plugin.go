package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/manland/mattermost-plugin-gitlab/server/gitlab"
	"github.com/manland/mattermost-plugin-gitlab/server/webhook"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"

	"golang.org/x/oauth2"
)

const (
	GITLAB_TOKEN_KEY      = "_gitlabtoken"
	GITLAB_USERNAME_KEY   = "_gitlabusername"
	GITLAB_IDUSERNAME_KEY = "_gitlabidusername"
	WS_EVENT_CONNECT      = "gitlab_connect"
	WS_EVENT_DISCONNECT   = "gitlab_disconnect"
	WS_EVENT_REFRESH      = "gitlab_refresh"
	SETTING_NOTIFICATIONS = "notifications"
	SETTING_REMINDERS     = "reminders"
	SETTING_ON            = "on"
	SETTING_OFF           = "off"
)

type Plugin struct {
	plugin.MattermostPlugin

	BotUserID      string
	WebhookHandler webhook.Webhook
	GitlabClient   gitlab.Gitlab

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

func (p *Plugin) OnActivate() error {
	if err := p.API.RegisterCommand(getCommand()); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Unable to register command: %v", getCommand()))
	}

	botID, ensureBotError := p.Helpers.EnsureBot(&model.Bot{
		Username:    "gitlab",
		DisplayName: "GitLab Plugin",
		Description: "A bot account created by the plugin GitLab.",
	})
	if ensureBotError != nil {
		return errors.Wrap(ensureBotError, "can't ensure bot")
	}
	p.BotUserID = botID

	p.WebhookHandler = webhook.NewWebhook(&gitlabRetreiver{p: p})

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "can't retreive bundle path")
	}
	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "failed to read profile image")
	}
	if appErr := p.API.SetProfileImage(botID, profileImage); appErr != nil {
		return errors.Wrap(err, "failed to set profile image")
	}

	return nil
}

func (p *Plugin) getOAuthConfig() *oauth2.Config {
	config := p.getConfiguration()

	authURL, _ := url.Parse(config.GitlabURL)
	tokenURL, _ := url.Parse(config.GitlabURL)

	authURL.Path = path.Join(authURL.Path, "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "oauth", "token")

	return &oauth2.Config{
		ClientID:     config.GitlabOAuthClientID,
		ClientSecret: config.GitlabOAuthClientSecret,
		Scopes:       []string{"api", "read_user"},
		RedirectURL:  fmt.Sprintf("%s/plugins/%s/oauth/complete", *p.API.GetConfig().ServiceSettings.SiteURL, manifest.ID),
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL.String(),
			TokenURL: tokenURL.String(),
		},
	}
}

func (p *Plugin) storeGitlabUserInfo(info *gitlab.GitlabUserInfo) error {
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

func (p *Plugin) getGitlabUserInfoByMattermostID(userID string) (*gitlab.GitlabUserInfo, *APIErrorResponse) {
	config := p.getConfiguration()

	var userInfo gitlab.GitlabUserInfo

	if infoBytes, err := p.API.KVGet(userID + GITLAB_TOKEN_KEY); err != nil || infoBytes == nil {
		return nil, &APIErrorResponse{ID: API_ERROR_ID_NOT_CONNECTED, Message: "Must connect user account to GitLab first.", StatusCode: http.StatusBadRequest}
	} else if err := json.Unmarshal(infoBytes, &userInfo); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse token.", StatusCode: http.StatusInternalServerError}
	}

	unencryptedToken, err := decrypt([]byte(config.EncryptionKey), userInfo.Token.AccessToken)
	if err != nil {
		p.API.LogError("can't decrypt token", "err", err.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	userInfo.Token.AccessToken = unencryptedToken

	return &userInfo, nil
}

func (p *Plugin) storeGitlabToUserIDMapping(gitlabUsername, userID string) error {
	if err := p.API.KVSet(gitlabUsername+GITLAB_USERNAME_KEY, []byte(userID)); err != nil {
		return fmt.Errorf("Encountered error saving GitLab username mapping")
	}
	if err := p.API.KVSet(userID+GITLAB_IDUSERNAME_KEY, []byte(gitlabUsername)); err != nil {
		return fmt.Errorf("Encountered error saving GitLab id mapping")
	}
	return nil
}

func (p *Plugin) getGitlabToUserIDMapping(gitlabUsername string) string {
	userID, err := p.API.KVGet(gitlabUsername + GITLAB_USERNAME_KEY)
	if err != nil {
		p.API.LogError("can't get userId from store with username", "err", err.DetailedError, "username", gitlabUsername)
	}
	return string(userID)
}

func (p *Plugin) getGitlabIDToUsernameMapping(gitlabUserID string) string {
	gitlabUsername, err := p.API.KVGet(gitlabUserID + GITLAB_IDUSERNAME_KEY)
	if err != nil {
		p.API.LogError("can't get user id by login", "err", err.DetailedError)
	}
	return string(gitlabUsername)
}

func (p *Plugin) disconnectGitlabAccount(userID string) {
	userInfo, err := p.getGitlabUserInfoByMattermostID(userID)
	if err != nil {
		p.API.LogError("can't get GitLab user info from mattermost id", "err", err.Message)
		return
	}
	if userInfo == nil {
		return
	}

	if err := p.API.KVDelete(userID + GITLAB_TOKEN_KEY); err != nil {
		p.API.LogError("can't delete token in store", "err", err.DetailedError, "userId", userID)
	}
	if err := p.API.KVDelete(userInfo.GitlabUsername + GITLAB_USERNAME_KEY); err != nil {
		p.API.LogError("can't delete username in store", "err", err.DetailedError, "username", userInfo.GitlabUsername)
	}
	if err := p.API.KVDelete(fmt.Sprintf("%d%s", userInfo.GitlabUserId, GITLAB_IDUSERNAME_KEY)); err != nil {
		p.API.LogError("can't delete user id in sotre", "err", err.DetailedError, "id", userInfo.GitlabUserId)
	}

	if user, err := p.API.GetUser(userID); err == nil && user.Props != nil && len(user.Props["git_user"]) > 0 {
		delete(user.Props, "git_user")
		if _, err := p.API.UpdateUser(user); err != nil {
			p.API.LogError("can't update user after delete git account", "err", err.DetailedError)
		}
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
		p.API.LogError("Couldn't get bot's DM channel", "user_id", userID)
		return err
	}

	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogError("can't post DM", "err", err.DetailedError)
		return err
	}

	return nil
}

func (p *Plugin) PostToDo(info *gitlab.GitlabUserInfo) {
	text, err := p.GetToDo(info)
	if err != nil {
		p.API.LogError("can't post todo", "err", err.Error())
		return
	}

	if err := p.CreateBotDMPost(info.UserID, text, "custom_git_todo"); err != nil {
		p.API.LogError("can't create dm post in post todo", "err", err.DetailedError)
	}
}

func (p *Plugin) GetToDo(user *gitlab.GitlabUserInfo) (string, error) {
	unreads, err := p.GitlabClient.GetUnreads(user)
	if err != nil {
		return "", err
	}

	yourAssignments, err := p.GitlabClient.GetYourAssignments(user)
	if err != nil {
		return "", err
	}

	yourMergeRequests, err := p.GitlabClient.GetYourPrs(user)
	if err != nil {
		return "", err
	}

	reviews, err := p.GitlabClient.GetReviews(user)
	if err != nil {
		return "", err
	}

	text := "##### Unread Messages\n"

	notificationCount := 0
	notificationContent := ""
	for _, n := range unreads {
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

	if len(reviews) == 0 {
		text += "You don't have any merge requests awaiting your review.\n"
	} else {
		text += fmt.Sprintf("You have %v merge requests awaiting your review:\n", len(reviews))

		for _, pr := range reviews {
			text += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
		}
	}

	text += "##### Assignments\n"

	if len(yourAssignments) == 0 {
		text += "You don't have any issues awaiting your dev.\n"
	} else {
		text += fmt.Sprintf("You have %v issues awaiting dev:\n", len(yourAssignments))

		for _, pr := range yourAssignments {
			text += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
		}
	}

	text += "##### Your Open Merge Requests\n"

	if len(yourMergeRequests) == 0 {
		text += "You don't have any open merge requests.\n"
	} else {
		text += fmt.Sprintf("You have %v open merge requests:\n", len(yourMergeRequests))

		for _, pr := range yourMergeRequests {
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
