package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/webhook"
)

const (
	GitlabTokenKey       = "_gitlabtoken"
	GitlabUsernameKey    = "_gitlabusername"
	GitlabIDUsernameKey  = "_gitlabidusername"
	WsEventConnect       = "gitlab_connect"
	WsEventDisconnect    = "gitlab_disconnect"
	WsEventRefresh       = "gitlab_refresh"
	SettingNotifications = "notifications"
	SettingReminders     = "reminders"
	SettingOn            = "on"
	SettingOff           = "off"

	chimeraGitLabAppIdentifier = "plugin-gitlab"
)

var errEmptySiteURL = errors.New("siteURL is not set. Please set it and restart the plugin")

type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	BotUserID      string
	WebhookHandler webhook.Webhook
	GitlabClient   gitlab.Gitlab

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	chimeraURL string
}

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	config := p.getConfiguration()
	if err := config.IsValid(); err != nil {
		return err
	}

	p.registerChimeraURL()

	if config.UsePreregisteredApplication && p.chimeraURL == "" {
		return errors.New("cannot use pre-registered application if Chimera URL is not set or empty. " +
			"For now using pre-registered application is intended for Cloud instances only. " +
			"If you are running on-prem disable the setting and use a custom application, otherwise set PluginSettings.ChimeraOAuthProxyURL " +
			"or MM_PLUGINSETTINGS_CHIMERAOAUTHPROXYURL environment variable")
	}

	command, err := p.getCommand()
	if err != nil {
		return errors.Wrap(err, "failed to get command")
	}

	err = p.API.RegisterCommand(command)
	if err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		Username:    "gitlab",
		DisplayName: "GitLab Plugin",
		Description: "A bot account created by the plugin GitLab.",
	})
	if err != nil {
		return errors.Wrap(err, "can't ensure bot")
	}
	p.BotUserID = botID

	p.WebhookHandler = webhook.NewWebhook(&gitlabRetreiver{p: p})

	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "can't retrieve bundle path")
	}
	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "failed to read profile image")
	}
	if appErr := p.API.SetProfileImage(botID, profileImage); appErr != nil {
		return errors.Wrap(err, "failed to set profile image")
	}

	siteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if siteURL == nil || *siteURL == "" {
		return errEmptySiteURL
	}

	return nil
}

func (p *Plugin) getOAuthConfig() *oauth2.Config {
	config := p.getConfiguration()

	scopes := []string{}
	redirectURL := fmt.Sprintf("%s/plugins/%s/oauth/complete", *p.API.GetConfig().ServiceSettings.SiteURL, manifest.ID)

	if config.UsePreregisteredApplication {
		p.API.LogDebug("Using Chimera Proxy OAuth configuration")
		return p.getOAuthConfigForChimeraApp(scopes, redirectURL)
	}

	authURL, _ := url.Parse(config.GitlabURL)
	tokenURL, _ := url.Parse(config.GitlabURL)

	authURL.Path = path.Join(authURL.Path, "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "oauth", "token")

	return &oauth2.Config{
		ClientID:     config.GitlabOAuthClientID,
		ClientSecret: config.GitlabOAuthClientSecret,
		Scopes:       scopes,
		RedirectURL:  redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL.String(),
			TokenURL: tokenURL.String(),
		},
	}
}

func (p *Plugin) getOAuthConfigForChimeraApp(scopes []string, redirectURL string) *oauth2.Config {
	baseURL := fmt.Sprintf("%s/v1/gitlab/%s", p.chimeraURL, chimeraGitLabAppIdentifier)
	authURL, _ := url.Parse(baseURL)
	tokenURL, _ := url.Parse(baseURL)

	authURL.Path = path.Join(authURL.Path, "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "oauth", "token")

	return &oauth2.Config{
		ClientID:     "placeholder",
		ClientSecret: "placeholder",
		Scopes:       scopes,
		RedirectURL:  redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL.String(),
			TokenURL:  tokenURL.String(),
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}
}

func (p *Plugin) storeGitlabUserInfo(info *gitlab.UserInfo) error {
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

	if err := p.API.KVSet(info.UserID+GitlabTokenKey, jsonInfo); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) deleteGitlabUserInfo(userID string) error {
	if err := p.API.KVDelete(userID + GitlabTokenKey); err != nil {
		return fmt.Errorf("encountered error deleting GitLab user info: %w", err)
	}
	return nil
}

func (p *Plugin) getGitlabUserInfoByMattermostID(userID string) (*gitlab.UserInfo, *APIErrorResponse) {
	config := p.getConfiguration()

	var userInfo gitlab.UserInfo

	if infoBytes, err := p.API.KVGet(userID + GitlabTokenKey); err != nil || infoBytes == nil {
		return nil, &APIErrorResponse{ID: APIErrorIDNotConnected, Message: "Must connect user account to GitLab first.", StatusCode: http.StatusBadRequest}
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
	if err := p.API.KVSet(gitlabUsername+GitlabUsernameKey, []byte(userID)); err != nil {
		return fmt.Errorf("encountered error saving GitLab username mapping: %w", err)
	}
	return nil
}

func (p *Plugin) storeGitlabIDToUserIDMapping(gitlabUsername string, gitlabID int) error {
	if err := p.API.KVSet(fmt.Sprintf("%d%s", gitlabID, GitlabIDUsernameKey), []byte(gitlabUsername)); err != nil {
		return fmt.Errorf("encountered error saving GitLab id mapping: %w", err)
	}
	return nil
}

func (p *Plugin) deleteGitlabToUserIDMapping(gitlabUsername string) error {
	if err := p.API.KVDelete(gitlabUsername + GitlabUsernameKey); err != nil {
		return fmt.Errorf("encountered error deleting GitLab username mapping: %w", err)
	}
	return nil
}

func (p *Plugin) deleteGitlabIDToUserIDMapping(gitlabID int) error {
	if err := p.API.KVDelete(fmt.Sprintf("%d%s", gitlabID, GitlabIDUsernameKey)); err != nil {
		return fmt.Errorf("encountered error deleting GitLab id mapping: %w", err)
	}
	return nil
}

func (p *Plugin) getGitlabToUserIDMapping(gitlabUsername string) string {
	userID, err := p.API.KVGet(gitlabUsername + GitlabUsernameKey)
	if err != nil {
		p.API.LogError("can't get userId from store with username", "err", err.DetailedError, "username", gitlabUsername)
	}
	return string(userID)
}

func (p *Plugin) getGitlabIDToUsernameMapping(gitlabUserID string) string {
	gitlabUsername, err := p.API.KVGet(gitlabUserID + GitlabIDUsernameKey)
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

	if err := p.deleteGitlabUserInfo(userID); err != nil {
		p.API.LogError("can't delete token in store", "err", err.Error, "userId", userID)
	}
	if err := p.deleteGitlabToUserIDMapping(userInfo.GitlabUsername); err != nil {
		p.API.LogError("can't delete username in store", "err", err.Error, "username", userInfo.GitlabUsername)
	}
	if err := p.deleteGitlabIDToUserIDMapping(userInfo.GitlabUserID); err != nil {
		p.API.LogError("can't delete user id in store", "err", err.Error, "id", userInfo.GitlabUserID)
	}

	if user, err := p.API.GetUser(userID); err == nil && user.Props != nil && len(user.Props["git_user"]) > 0 {
		delete(user.Props, "git_user")
		if _, err := p.API.UpdateUser(user); err != nil {
			p.API.LogError("can't update user after delete git account", "err", err.DetailedError)
		}
	}

	p.API.PublishWebSocketEvent(
		WsEventDisconnect,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

// registerChimeraURL fetches the Chimera URL from server settings or env var and sets it in the plugin object.
func (p *Plugin) registerChimeraURL() {
	chimeraURLSetting := p.API.GetConfig().PluginSettings.ChimeraOAuthProxyURL
	if chimeraURLSetting != nil && *chimeraURLSetting != "" {
		p.chimeraURL = *chimeraURLSetting
		return
	}
	// Due to setting name change in v6 (ChimeraOAuthProxyUrl -> ChimeraOAuthProxyURL)
	// fall back to env var to work with older servers.
	p.chimeraURL = os.Getenv("MM_PLUGINSETTINGS_CHIMERAOAUTHPROXYURL")
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

func (p *Plugin) PostToDo(info *gitlab.UserInfo) {
	hasTodo, text, err := p.GetToDo(info)
	if err != nil {
		p.API.LogError("can't post todo", "err", err.Error())
		return
	}
	if !hasTodo {
		return
	}

	if err := p.CreateBotDMPost(info.UserID, text, "custom_git_todo"); err != nil {
		p.API.LogError("can't create dm post in post todo", "err", err.DetailedError)
	}
}

func (p *Plugin) GetToDo(user *gitlab.UserInfo) (bool, string, error) {
	var hasTodo bool

	unreads, err := p.GitlabClient.GetUnreads(user)
	if err != nil {
		return false, "", err
	}

	yourAssignments, err := p.GitlabClient.GetYourAssignments(user)
	if err != nil {
		return false, "", err
	}

	yourMergeRequests, err := p.GitlabClient.GetYourPrs(user)
	if err != nil {
		return false, "", err
	}

	reviews, err := p.GitlabClient.GetReviews(user)
	if err != nil {
		return false, "", err
	}

	text := "##### Unread Messages\n"

	notificationCount := 0
	notificationContent := ""
	for _, n := range unreads {
		if p.isNamespaceAllowed(n.Project.NameWithNamespace) != nil {
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

		hasTodo = true
	}

	text += "##### Review Requests\n"

	if len(reviews) == 0 {
		text += "You don't have any merge requests awaiting your review.\n"
	} else {
		text += fmt.Sprintf("You have %v merge requests awaiting your review:\n", len(reviews))

		for _, pr := range reviews {
			text += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
		}

		hasTodo = true
	}

	text += "##### Assignments\n"

	if len(yourAssignments) == 0 {
		text += "You don't have any issues awaiting your dev.\n"
	} else {
		text += fmt.Sprintf("You have %v issues awaiting dev:\n", len(yourAssignments))

		for _, pr := range yourAssignments {
			text += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
		}

		hasTodo = true
	}

	text += "##### Your Open Merge Requests\n"

	if len(yourMergeRequests) == 0 {
		text += "You don't have any open merge requests.\n"
	} else {
		text += fmt.Sprintf("You have %v open merge requests:\n", len(yourMergeRequests))

		for _, pr := range yourMergeRequests {
			text += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
		}

		hasTodo = true
	}

	return hasTodo, text, nil
}

func (p *Plugin) isNamespaceAllowed(namespace string) error {
	allowedNamespace := strings.TrimSpace(p.getConfiguration().GitlabGroup)
	if allowedNamespace != "" && allowedNamespace != namespace && !strings.HasPrefix(namespace, allowedNamespace) {
		return fmt.Errorf("only repositories in the %s namespace are allowed", allowedNamespace)
	}

	return nil
}

func (p *Plugin) sendRefreshEvent(userID string) {
	p.API.PublishWebSocketEvent(
		WsEventRefresh,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

// HasProjectHook checks if the subscribed GitLab Project or its parrent Group has a webhook
// with a URL that matches the Mattermost Site URL.
func (p *Plugin) HasProjectHook(user *gitlab.UserInfo, namespace string, project string) (bool, error) {
	hooks, err := p.GitlabClient.GetProjectHooks(user, namespace, project)
	if err != nil {
		return false, errors.New("unable to connect to GitLab")
	}

	// ignore error because many project won't be part of groups
	hasGroupHook, _ := p.HasGroupHook(user, namespace)

	if hasGroupHook {
		return true, err
	}

	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	found := false
	for _, hook := range hooks {
		if strings.Contains(hook.URL, siteURL) {
			found = true
		}
	}
	return found, nil
}

// HasGroupHook checks if the subscribed GitLab Group has a webhook
// with a URL that matches the Mattermost Site URL.
func (p *Plugin) HasGroupHook(user *gitlab.UserInfo, namespace string) (bool, error) {
	hooks, err := p.GitlabClient.GetGroupHooks(user, namespace)
	if err != nil {
		return false, errors.New("unable to connect to GitLab")
	}

	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	found := false
	for _, hook := range hooks {
		if strings.Contains(hook.URL, siteURL) {
			found = true
		}
	}

	return found, err
}
