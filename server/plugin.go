// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/poster"
	"github.com/pkg/errors"
	gitlabLib "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/webhook"
)

const (
	GitlabUserInfoKey             = "_userinfo"
	GitlabUserTokenKey            = "_usertoken"
	GitlabMigrationTokenKey       = "_gitlabtoken"
	TokenMutexKey                 = "-oauth-token" //#nosec G101 -- False positive
	GitlabUsernameKey             = "_gitlabusername"
	GitlabIDUsernameKey           = "_gitlabidusername"
	WsEventConnect                = "gitlab_connect"
	WsEventDisconnect             = "gitlab_disconnect"
	WsEventRefresh                = "gitlab_refresh"
	wsEventCreateIssue            = "create_issue"
	SettingNotifications          = "notifications"
	SettingReminders              = "reminders"
	SettingOn                     = "on"
	SettingOff                    = "off"
	WsChannelSubscriptionsUpdated = "gitlab_channel_subscriptions_updated"

	chimeraGitLabAppIdentifier = "plugin-gitlab"

	NotificationActionNameMemberAccessRequest = "member_access_requested"

	invalidTokenError = "401 {error: invalid_token}" //#nosec G101 -- False positive

	// The duration before token expiry when we should refresh the token.
	tokenExpiryBuffer = 5 * time.Minute
)

type Plugin struct {
	plugin.MattermostPlugin
	client *pluginapi.Client

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	chimeraURL string

	router *mux.Router

	BotUserID   string
	poster      poster.Poster
	flowManager *FlowManager

	oauthBroker *OAuthBroker

	WebhookHandler webhook.Webhook
	GitlabClient   gitlab.Gitlab
}

// gitlabPermalinkRegex is used to parse gitlab permalinks in post messages.
var gitlabPermalinkRegex = regexp.MustCompile(`https?://(?P<haswww>www\.)?gitlab\.com/(?P<user>[\w/.?-]+)/(?P<repo>[\w-]+)/-/blob/(?P<commit>\w+)/(?P<path>[\w-/.]+)#(?P<line>[\w-]+)?`)

// NewPlugin returns an instance of a Plugin.
func NewPlugin() *Plugin {
	return &Plugin{}
}

func (p *Plugin) OnActivate() error {
	if p.client == nil {
		p.client = pluginapi.NewClient(p.API, p.Driver)
	}
	siteURL := getSiteURL(p.client)
	if siteURL == "" {
		return errors.New("siteURL is not set. Please set it and restart the plugin")
	}

	err := p.setDefaultConfiguration()
	if err != nil {
		return errors.Wrap(err, "failed to set default configuration")
	}

	p.registerChimeraURL()

	if p.getConfiguration().UsePreregisteredApplication && p.chimeraURL == "" {
		return errors.New("cannot use pre-registered application if Chimera URL is not set or empty. " +
			"For now using pre-registered application is intended for Cloud instances only. " +
			"If you are running on-prem disable the setting and use a custom application, otherwise set PluginSettings.ChimeraOAuthProxyURL " +
			"or MM_PLUGINSETTINGS_CHIMERAOAUTHPROXYURL environment variable")
	}

	p.initializeAPI()

	p.oauthBroker = NewOAuthBroker(p.sendOAuthCompleteEvent)

	botID, err := p.client.Bot.EnsureBot(&model.Bot{
		OwnerId:     manifest.Id, // Workaround to support older server version affected by https://github.com/mattermost/mattermost-server/pull/21560
		Username:    "gitlab",
		DisplayName: "GitLab Plugin",
		Description: "A bot account created by the plugin GitLab.",
	}, pluginapi.ProfileImagePath(filepath.Join("assets", "profile.png")))
	if err != nil {
		return errors.Wrap(err, "can't ensure bot")
	}
	p.BotUserID = botID

	p.WebhookHandler = webhook.NewWebhook(&gitlabRetreiver{p: p})

	p.poster = poster.NewPoster(&p.client.Post, p.BotUserID)
	flowManager, err := p.NewFlowManager()
	if err != nil {
		return errors.Wrap(err, "failed to create flow manager")
	}
	p.flowManager = flowManager

	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.oauthBroker.Close()

	return nil
}

func (p *Plugin) OnInstall(c *plugin.Context, event model.OnInstallEvent) error {
	conf := p.getConfiguration()

	// Don't start wizard if OAuth is configured
	if conf.IsOAuthConfigured() {
		p.client.Log.Debug("OAuth is configured, skipping setup wizard",
			"GitlabOAuthClientID", lastN(conf.GitlabOAuthClientID, 4),
			"GitlabOAuthClientSecret", lastN(conf.GitlabOAuthClientSecret, 4),
			"UsePreregisteredApplication", conf.UsePreregisteredApplication)
		return nil
	}

	return p.flowManager.StartSetupWizard(event.UserId, "")
}

func (p *Plugin) OnPluginClusterEvent(c *plugin.Context, ev model.PluginClusterEvent) {
	p.HandleClusterEvent(ev)
}

func (p *Plugin) UserHasBeenDeactivated(c *plugin.Context, user *model.User) {
	if info, _ := p.getGitlabUserInfoByMattermostID(user.Id); info != nil {
		p.disconnectGitlabAccount(user.Id)
	}
}

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	// If not enabled in config, ignore.
	if p.getConfiguration().EnableCodePreview == "disable" {
		return nil, ""
	}

	if post.UserId == "" {
		return nil, ""
	}

	msg := post.Message
	replacements := p.getPermalinkReplacements(msg)
	if len(replacements) == 0 {
		return nil, ""
	}
	info, err := p.getGitlabUserInfoByMattermostID(post.UserId)
	if err != nil {
		if err.ID == APIErrorIDNotConnected {
			p.API.LogDebug("Error while processing permalinks in the post", "Error", err.Error())
		} else {
			p.API.LogDebug("Error in getting user info", "Error", err.Error())
		}
		return nil, ""
	}

	var glClient *gitlabLib.Client
	if cErr := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GitlabConnect(*token)
		if err != nil {
			return err
		}

		glClient = resp
		return nil
	}); cErr != nil {
		p.API.LogDebug("Error in getting GitLab client", "Error", cErr.Error())
		return nil, ""
	}

	post.Message = p.makeReplacements(msg, replacements, glClient)
	return post, ""
}

func (p *Plugin) setDefaultConfiguration() error {
	config := p.getConfiguration()

	changed, err := config.setDefaults(pluginapi.IsCloud(p.client.System.GetLicense()))
	if err != nil {
		return err
	}

	if changed {
		configMap, err := config.ToMap()
		if err != nil {
			return err
		}

		err = p.client.Configuration.SavePluginConfig(configMap)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) getGitlabClient() gitlab.Gitlab {
	return p.GitlabClient
}

func (p *Plugin) getOAuthConfig() *oauth2.Config {
	config := p.getConfiguration()
	instanceConfiguration, err := p.getInstance(config.DefaultInstanceName)
	if err != nil {
		p.client.Log.Warn("Failed to get instance configuration", "error", err.Error())
		return nil
	}

	scopes := []string{"api", "read_user"}

	redirectURL := fmt.Sprintf("%s/oauth/complete", getPluginURL(p.client))

	if config.UsePreregisteredApplication {
		p.client.Log.Debug("Using Chimera Proxy OAuth configuration")
		return p.getOAuthConfigForChimeraApp(scopes, redirectURL)
	}

	authURL, _ := url.Parse(config.GitlabURL)
	tokenURL, _ := url.Parse(config.GitlabURL)

	authURL.Path = path.Join(authURL.Path, "oauth", "authorize")
	tokenURL.Path = path.Join(tokenURL.Path, "oauth", "token")

	return &oauth2.Config{
		ClientID:     instanceConfiguration.GitlabOAuthClientID,
		ClientSecret: instanceConfiguration.GitlabOAuthClientSecret,
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
	jsonInfo, err := json.Marshal(info)
	if err != nil {
		return err
	}

	if _, err := p.client.KV.Set(info.UserID+GitlabUserInfoKey, jsonInfo); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) storeGitlabUserToken(userID string, token *oauth2.Token) error {
	config := p.getConfiguration()

	jsonToken, err := json.Marshal(token)
	if err != nil {
		return err
	}

	encryptedToken, err := encrypt([]byte(config.EncryptionKey), string(jsonToken))
	if err != nil {
		return err
	}

	if _, err := p.client.KV.Set(userID+GitlabUserTokenKey, []byte(encryptedToken)); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) deleteGitlabUserInfo(userID string) error {
	if err := p.client.KV.Delete(userID + GitlabUserInfoKey); err != nil {
		return errors.Wrap(err, "encountered error deleting GitLab user info")
	}
	return nil
}

func (p *Plugin) deleteGitlabUserToken(userID string) error {
	if err := p.client.KV.Delete(userID + GitlabUserTokenKey); err != nil {
		return errors.Wrap(err, "encountered error deleting GitLab user token")
	}
	return nil
}

func (p *Plugin) getGitlabUserInfoByMattermostID(userID string) (*gitlab.UserInfo, *APIErrorResponse) {
	var userInfo gitlab.UserInfo

	var infoBytes []byte
	err := p.client.KV.Get(userID+GitlabUserInfoKey, &infoBytes)

	if err != nil || infoBytes == nil {
		var gitlabTokenBytes []byte
		appErr := p.client.KV.Get(userID+GitlabMigrationTokenKey, &gitlabTokenBytes)
		if appErr != nil || gitlabTokenBytes == nil {
			return nil, &APIErrorResponse{ID: APIErrorIDNotConnected, Message: "Must connect user account to GitLab first.", StatusCode: http.StatusBadRequest}
		}
		if err := json.Unmarshal(gitlabTokenBytes, &userInfo); err != nil {
			return nil, &APIErrorResponse{ID: "", Message: "Unable to parse user info from migration key.", StatusCode: http.StatusInternalServerError}
		}
	} else if err := json.Unmarshal(infoBytes, &userInfo); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse user info.", StatusCode: http.StatusInternalServerError}
	}

	return &userInfo, nil
}

func (p *Plugin) migrateGitlabToken(userID string) (*oauth2.Token, *APIErrorResponse) {
	config := p.getConfiguration()

	var userInfo struct {
		gitlab.UserInfo
		Token *oauth2.Token
	}

	mutex, err := cluster.NewMutex(p.API, userID+TokenMutexKey)
	if err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to obtain mutex for KV migration.", StatusCode: http.StatusInternalServerError}
	}
	mutex.Lock()
	defer mutex.Unlock()

	var gitlabTokenBytes []byte
	err = p.client.KV.Get(userID+GitlabMigrationTokenKey, &gitlabTokenBytes)
	if err != nil || gitlabTokenBytes == nil {
		return p.getGitlabUserTokenByMattermostID(userID)
	}

	if err = json.Unmarshal(gitlabTokenBytes, &userInfo); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse user info for KV migration.", StatusCode: http.StatusInternalServerError}
	}

	unencryptedToken, err := decrypt([]byte(config.EncryptionKey), userInfo.Token.AccessToken)
	if err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt token for KV migration.", StatusCode: http.StatusInternalServerError}
	}

	userInfo.Token.AccessToken = unencryptedToken

	userInfoWithoutToken := &gitlab.UserInfo{
		UserID:         userInfo.UserID,
		GitlabUserID:   userInfo.GitlabUserID,
		GitlabUsername: userInfo.GitlabUsername,
		LastToDoPostAt: userInfo.LastToDoPostAt,
		Settings:       userInfo.Settings,
	}

	if err = p.storeGitlabUserInfo(userInfoWithoutToken); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to store user info for KV migration.", StatusCode: http.StatusInternalServerError}
	}

	if err = p.storeGitlabUserToken(userInfo.UserID, userInfo.Token); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to store token for KV migration.", StatusCode: http.StatusInternalServerError}
	}

	if err = p.client.KV.Delete(userInfo.UserID + GitlabMigrationTokenKey); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to delete KV entry for migration.", StatusCode: http.StatusInternalServerError}
	}

	return userInfo.Token, nil
}

func (p *Plugin) getGitlabUserTokenByMattermostID(userID string) (*oauth2.Token, *APIErrorResponse) {
	config := p.getConfiguration()
	var token oauth2.Token

	var tokenBytes []byte
	err := p.client.KV.Get(userID+GitlabUserTokenKey, &tokenBytes)
	if err != nil || tokenBytes == nil {
		return nil, &APIErrorResponse{ID: APIErrorIDNotConnected, Message: "Must connect user account to GitLab first.", StatusCode: http.StatusBadRequest}
	}

	unencryptedToken, err := decrypt([]byte(config.EncryptionKey), string(tokenBytes))
	if err != nil {
		p.client.Log.Warn("can't decrypt token", "err", err.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	if err := json.Unmarshal([]byte(unencryptedToken), &token); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse token.", StatusCode: http.StatusInternalServerError}
	}

	return &token, nil
}

func (p *Plugin) storeGitlabToUserIDMapping(gitlabUsername, userID string) error {
	if _, err := p.client.KV.Set(gitlabUsername+GitlabUsernameKey, []byte(userID)); err != nil {
		return errors.Wrap(err, "encountered error saving GitLab username mapping")
	}
	return nil
}

func (p *Plugin) storeGitlabIDToUserIDMapping(gitlabUsername string, gitlabID int) error {
	if _, err := p.client.KV.Set(fmt.Sprintf("%d%s", gitlabID, GitlabIDUsernameKey), []byte(gitlabUsername)); err != nil {
		return errors.Wrap(err, "encountered error saving GitLab id mapping")
	}
	return nil
}

func (p *Plugin) deleteGitlabToUserIDMapping(gitlabUsername string) error {
	if err := p.client.KV.Delete(gitlabUsername + GitlabUsernameKey); err != nil {
		return errors.Wrap(err, "encountered error deleting GitLab username mapping")
	}
	return nil
}

func (p *Plugin) deleteGitlabIDToUserIDMapping(gitlabID int) error {
	if err := p.client.KV.Delete(fmt.Sprintf("%d%s", gitlabID, GitlabIDUsernameKey)); err != nil {
		return errors.Wrap(err, "encountered error deleting GitLab id mapping")
	}
	return nil
}

func (p *Plugin) getGitlabToUserIDMapping(gitlabUsername string) string {
	var userID []byte
	err := p.client.KV.Get(gitlabUsername+GitlabUsernameKey, &userID)
	if err != nil {
		p.client.Log.Warn("can't get userId from store with username", "err", err.Error(), "username", gitlabUsername)
	}
	return string(userID)
}

func (p *Plugin) getGitlabIDToUsernameMapping(gitlabUserID string) string {
	var gitlabUsername []byte
	err := p.client.KV.Get(gitlabUserID+GitlabIDUsernameKey, &gitlabUsername)
	if err != nil {
		p.client.Log.Warn("can't get user id by login", "err", err.Error())
	}
	return string(gitlabUsername)
}

func (p *Plugin) disconnectGitlabAccount(userID string) {
	userInfo, err := p.getGitlabUserInfoByMattermostID(userID)
	if err != nil {
		p.client.Log.Warn("can't get GitLab user info from mattermost id", "err", err.Message)
		return
	}
	if userInfo == nil {
		return
	}

	if err := p.deleteGitlabUserInfo(userID); err != nil {
		p.client.Log.Warn("can't delete user info in store", "err", err.Error, "userId", userID)
	}
	if err := p.deleteGitlabUserToken(userID); err != nil {
		p.client.Log.Warn("can't delete token in store", "err", err.Error, "userId", userID)
	}
	if err := p.deleteGitlabToUserIDMapping(userInfo.GitlabUsername); err != nil {
		p.client.Log.Warn("can't delete username in store", "err", err.Error, "username", userInfo.GitlabUsername)
	}
	if err := p.deleteGitlabIDToUserIDMapping(userInfo.GitlabUserID); err != nil {
		p.client.Log.Warn("can't delete user id in store", "err", err.Error, "id", userInfo.GitlabUserID)
	}

	if user, err := p.client.User.Get(userID); err == nil && user.Props != nil && len(user.Props["git_user"]) > 0 {
		delete(user.Props, "git_user")
		if err := p.client.User.Update(user); err != nil {
			p.client.Log.Warn("can't update user after delete git account", "err", err.Error())
		}
	}

	p.client.Frontend.PublishWebSocketEvent(
		WsEventDisconnect,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

// registerChimeraURL fetches the Chimera URL from server settings or env var and sets it in the plugin object.
func (p *Plugin) registerChimeraURL() {
	chimeraURLSetting := p.client.Configuration.GetConfig().PluginSettings.ChimeraOAuthProxyURL
	if chimeraURLSetting != nil && *chimeraURLSetting != "" {
		p.chimeraURL = *chimeraURLSetting
		return
	}
	// Due to setting name change in v6 (ChimeraOAuthProxyUrl -> ChimeraOAuthProxyURL)
	// fall back to env var to work with older servers.
	p.chimeraURL = os.Getenv("MM_PLUGINSETTINGS_CHIMERAOAUTHPROXYURL")
}

func (p *Plugin) CreateBotDMPost(userID, message, postType string) error {
	channel, err := p.client.Channel.GetDirect(userID, p.BotUserID)
	if err != nil {
		p.client.Log.Warn("Couldn't get bot's DM channel", "user_id", userID)
		return err
	}

	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
	}

	if err := p.client.Post.CreatePost(post); err != nil {
		p.client.Log.Warn("can't post DM", "err", err.Error())
		return err
	}

	return nil
}

func (p *Plugin) PostToDo(ctx context.Context, info *gitlab.UserInfo) {
	hasTodo, text, err := p.GetToDo(ctx, info)
	if err != nil {
		p.client.Log.Warn("can't post todo", "err", err.Error())
		return
	}
	if !hasTodo {
		return
	}

	if err := p.CreateBotDMPost(info.UserID, text, "custom_git_todo"); err != nil {
		p.client.Log.Warn("can't create dm post in post todo", "err", err.Error())
	}
}

func (p *Plugin) GetToDo(ctx context.Context, user *gitlab.UserInfo) (bool, string, error) {
	hasTodo := false

	var notificationText, reviewText, assignmentText, mergeRequestText string
	err := p.useGitlabClient(user, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetLHSData(ctx, info, token)
		if err != nil {
			return err
		}

		todos := resp.Todos
		notificationCount := 0
		var notificationContent strings.Builder

		for _, n := range todos {
			if n == nil {
				continue
			}

			if n.Project != nil && p.isNamespaceAllowed(n.Project.NameWithNamespace) != nil {
				continue
			}
			notificationCount++

			switch n.ActionName {
			// Handle special cases where the provided "Title" value is blank
			case NotificationActionNameMemberAccessRequest:
				fmt.Fprintf(&notificationContent, "* %v : [%v](%v) has requested access to [%v](%v)\n", n.ActionName, n.Author.Name, n.Author.WebURL, n.Body, n.TargetURL)
			default:
				fmt.Fprintf(&notificationContent, "* %v : [%v](%v)\n", n.ActionName, n.Target.Title, n.TargetURL)
			}
		}

		if notificationCount == 0 {
			notificationText += "You don't have any todos.\n"
		} else {
			notificationText += fmt.Sprintf("You have %v todos:\n", notificationCount)
			notificationText += notificationContent.String()

			hasTodo = true
		}

		reviews := resp.Reviews
		if len(reviews) == 0 {
			reviewText += "You don't have any merge requests awaiting your review.\n"
		} else {
			reviewText += fmt.Sprintf("You have %v merge requests awaiting your review:\n", len(reviews))

			for _, pr := range reviews {
				reviewText += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
			}

			hasTodo = true
		}

		yourAssignedIssues := resp.AssignedIssues
		if len(yourAssignedIssues) == 0 {
			assignmentText += "You don't have any issues awaiting your dev.\n"
		} else {
			assignmentText += fmt.Sprintf("You have %v issues awaiting dev:\n", len(yourAssignedIssues))

			for _, pr := range yourAssignedIssues {
				assignmentText += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
			}

			hasTodo = true
		}

		mergeRequests := resp.AssignedPRs
		if len(mergeRequests) == 0 {
			mergeRequestText += "You don't have any merge requests assigned.\n"
		} else {
			mergeRequestText += fmt.Sprintf("You have %v merge requests assigned:\n", len(mergeRequests))

			for _, pr := range mergeRequests {
				mergeRequestText += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
			}

			hasTodo = true
		}

		return nil
	})
	if err != nil {
		return false, "", err
	}

	text := "##### To-Do list\n"
	text += notificationText

	text += "##### Review Requests\n"
	text += reviewText

	text += "##### Issues\n"
	text += assignmentText

	text += "##### Merge Requests Assigned\n"
	text += mergeRequestText

	return hasTodo, text, nil
}

func (p *Plugin) isNamespaceAllowed(namespace string) error {
	allowedNamespace := strings.TrimSpace(p.getConfiguration().GitlabGroup)
	if allowedNamespace == "" {
		return nil
	}

	if namespace != allowedNamespace && !strings.HasPrefix(namespace, allowedNamespace+"/") {
		return errors.Errorf("only repositories in the %s namespace are allowed", allowedNamespace)
	}
	return nil
}

func (p *Plugin) sendRefreshEvent(userID string) {
	p.client.Frontend.PublishWebSocketEvent(
		WsEventRefresh,
		nil,
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) openIssueCreateModal(userID, channelID, title string) {
	p.API.PublishWebSocketEvent(
		wsEventCreateIssue,
		map[string]any{
			"title":      title,
			"channel_id": channelID,
		},
		&model.WebsocketBroadcast{UserId: userID},
	)
}

func (p *Plugin) sendChannelSubscriptionsUpdated(subs *Subscriptions, channelID string) {
	config := p.getConfiguration()

	subscriptions := filterSubscriptionsByChannel(subs, channelID)

	var payload struct {
		ChannelID     string                 `json:"channel_id"`
		Subscriptions []SubscriptionResponse `json:"subscriptions"`
	}
	payload.ChannelID = channelID
	payload.Subscriptions = subscriptionsToResponse(config, subscriptions)

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		p.client.Log.Warn(
			"unable to marshal payload for updated channel subscriptions",
			"err", err.Error(),
		)
		return
	}

	p.client.Frontend.PublishWebSocketEvent(
		WsChannelSubscriptionsUpdated,
		map[string]any{"payload": string(payloadJSON)},
		&model.WebsocketBroadcast{ChannelId: channelID},
	)
}

// HasProjectHook checks if the subscribed GitLab Project or its parrent Group has a webhook
// with a URL that matches the Mattermost Site URL.
func (p *Plugin) HasProjectHook(ctx context.Context, user *gitlab.UserInfo, namespace string, project string) (bool, error) {
	var hooks []*gitlab.WebhookInfo
	err := p.useGitlabClient(user, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetProjectHooks(ctx, info, token, namespace, project)
		if err != nil {
			return err
		}
		hooks = resp
		return nil
	})
	if err != nil {
		return false, errors.New("unable to connect to GitLab")
	}

	// ignore error because many project won't be part of groups
	hasGroupHook, _ := p.HasGroupHook(ctx, user, namespace)

	if hasGroupHook {
		return true, err
	}

	siteURL := getSiteURL(p.client)
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
func (p *Plugin) HasGroupHook(ctx context.Context, user *gitlab.UserInfo, namespace string) (bool, error) {
	var hooks []*gitlab.WebhookInfo
	err := p.useGitlabClient(user, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetGroupHooks(ctx, info, token, namespace)
		if err != nil {
			return err
		}
		hooks = resp
		return nil
	})
	if err != nil {
		return false, errors.New("unable to connect to GitLab")
	}

	siteURL := getSiteURL(p.client)
	found := false
	for _, hook := range hooks {
		if strings.Contains(hook.URL, siteURL) {
			found = true
		}
	}

	return found, err
}

// getUsername returns the GitLab username for a given Mattermost user,
// if the user is connected to GitLab via this plugin.
// Otherwise it returns the Mattermost username.
// One use of this function is to include an at-mention username in the GitLab comment itself.
func (p *Plugin) getUsername(userID string) (string, *APIErrorResponse) {
	info, err := p.getGitlabUserInfoByMattermostID(userID)
	if err != nil && err.ID != APIErrorIDNotConnected {
		return "", err
	} else if err != nil {
		user, appErr := p.API.GetUser(userID)
		if appErr != nil {
			return "", &APIErrorResponse{
				StatusCode: appErr.StatusCode,
				Message:    appErr.Message,
			}
		}

		return user.Username, nil
	}

	return info.GitlabUsername, nil
}

func (p *Plugin) refreshToken(userInfo *gitlab.UserInfo, token *oauth2.Token) (*oauth2.Token, error) {
	conf := p.getOAuthConfig()

	// Use ReuseTokenSourceWithExpiry to ensure the oauth2 library uses the same expiry buffer
	// as our plugin's check. This prevents a race condition where our check decides to refresh
	// but oauth2's default 10-second buffer says the token is still valid.
	src := oauth2.ReuseTokenSourceWithExpiry(token, conf.TokenSource(context.Background(), token), tokenExpiryBuffer)

	newToken, err := src.Token() // this actually goes and renews the tokens
	if err != nil {
		if strings.Contains(err.Error(), "\"error\":\"invalid_grant\"") {
			p.client.Log.Warn("Failed to refresh OAuth token as the existing one has an invalid grant. Revoking the token.", "userInfo", userInfo, "error", err.Error())
			p.handleRevokedToken(userInfo)
		}
		return nil, errors.Wrap(err, "unable to get the new refreshed token")
	}

	if newToken.AccessToken != token.AccessToken {
		p.client.Log.Debug("Gitlab token refreshed.", "UserID", userInfo.UserID)

		if err := p.storeGitlabUserToken(userInfo.UserID, newToken); err != nil {
			return nil, errors.Wrap(err, "unable to store user info with refreshed token")
		}

		return newToken, nil
	}

	return token, nil
}

func (p *Plugin) handleRevokedToken(info *gitlab.UserInfo) {
	p.disconnectGitlabAccount(info.UserID)
	err := p.CreateBotDMPost(info.UserID, "Your GitLab account was disconnected due to an invalid or revoked authorization token. Reconnect your account using the `/gitlab connect` command.", "custom_git_revoked_token")
	if err != nil {
		p.client.Log.Warn("Error sending revoked token DM post", "err", err.Error())
	}
}

func (p *Plugin) getOrRefreshTokenWithMutex(info *gitlab.UserInfo) (*oauth2.Token, error) {
	token, apiErr := p.getGitlabUserTokenByMattermostID(info.UserID)

	if apiErr != nil {
		token, apiErr = p.migrateGitlabToken(info.UserID)
		if apiErr != nil {
			return nil, apiErr
		}
	}

	if time.Until(token.Expiry) > tokenExpiryBuffer {
		return token, nil
	}

	mutex, err := cluster.NewMutex(p.API, info.UserID+TokenMutexKey)
	if err != nil {
		return nil, err
	}

	mutex.Lock()
	defer mutex.Unlock()

	lockedToken, apiErr := p.getGitlabUserTokenByMattermostID(info.UserID)
	if apiErr != nil {
		return nil, apiErr
	}

	if time.Until(lockedToken.Expiry) > tokenExpiryBuffer {
		return lockedToken, nil
	}

	newToken, err := p.refreshToken(info, lockedToken)
	if err != nil {
		return nil, err
	}

	return newToken, nil
}

func (p *Plugin) useGitlabClient(info *gitlab.UserInfo, toRun func(info *gitlab.UserInfo, token *oauth2.Token) error) error {
	token, err := p.getOrRefreshTokenWithMutex(info)
	if err != nil {
		return err
	}

	err = toRun(info, token)

	if err != nil && strings.Contains(err.Error(), invalidTokenError) {
		p.client.Log.Warn("Revoking OAuth token while using Gitlab client as it is invalid", "userInfo", info, "error", err.Error())
		p.handleRevokedToken(info)
	}

	return err
}

func (p *Plugin) logWarnings(warnings []string) {
	for _, warning := range warnings {
		p.client.Log.Warn(warning)
	}
}
