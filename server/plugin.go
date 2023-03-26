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
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/cluster"
	"github.com/mattermost/mattermost-plugin-api/experimental/bot/poster"
	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
	gitlabLib "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	root "github.com/mattermost/mattermost-plugin-gitlab"
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
	WsChannelSubscriptionsUpdated = "gitlab_channel_subscriptions_updated"
	SettingNotifications          = "notifications"
	SettingReminders              = "reminders"
	SettingOn                     = "on"
	SettingOff                    = "off"

	chimeraGitLabAppIdentifier = "plugin-gitlab"

	invalidTokenError = "401 {error: invalid_token}" //#nosec G101 -- False positive
)

var (
	manifest model.Manifest = root.Manifest
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

	telemetryClient telemetry.Client
	tracker         telemetry.Tracker

	BotUserID   string
	poster      poster.Poster
	flowManager *FlowManager

	oauthBroker *OAuthBroker

	WebhookHandler webhook.Webhook
	GitlabClient   gitlab.Gitlab
}

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	siteURL := p.API.GetConfig().ServiceSettings.SiteURL
	if siteURL == nil || *siteURL == "" {
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

	p.telemetryClient, err = telemetry.NewRudderClient()
	if err != nil {
		p.API.LogWarn("Telemetry client not started", "error", err.Error())
	}

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
	p.flowManager = p.NewFlowManager()

	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.oauthBroker.Close()

	return nil
}

func (p *Plugin) OnInstall(c *plugin.Context, event model.OnInstallEvent) error {
	// Don't start wizard if OAuth is configured
	if p.getConfiguration().IsOAuthConfigured() {
		return nil
	}

	return p.flowManager.StartSetupWizard(event.UserId, "")
}

func (p *Plugin) OnSendDailyTelemetry() {
	p.SendDailyTelemetry()
}

func (p *Plugin) OnPluginClusterEvent(c *plugin.Context, ev model.PluginClusterEvent) {
	p.HandleClusterEvent(ev)
}

func (p *Plugin) setDefaultConfiguration() error {
	config := p.getConfiguration()

	changed, err := config.setDefaults(pluginapi.IsCloud(p.API.GetLicense()))
	if err != nil {
		return err
	}

	if changed {
		configMap, err := config.ToMap()
		if err != nil {
			return err
		}

		appErr := p.API.SavePluginConfig(configMap)
		if appErr != nil {
			return appErr
		}
	}

	return nil
}

func (p *Plugin) getGitlabClient() gitlab.Gitlab {
	return p.GitlabClient
}

func (p *Plugin) getOAuthConfig() *oauth2.Config {
	config := p.getConfiguration()

	scopes := []string{"api", "read_user"}
	redirectURL := fmt.Sprintf("%s/plugins/%s/oauth/complete", *p.API.GetConfig().ServiceSettings.SiteURL, manifest.Id)

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
	jsonInfo, err := json.Marshal(info)
	if err != nil {
		return err
	}

	if err := p.API.KVSet(info.UserID+GitlabUserInfoKey, jsonInfo); err != nil {
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

	if err := p.API.KVSet(userID+GitlabUserTokenKey, []byte(encryptedToken)); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) deleteGitlabUserInfo(userID string) error {
	if err := p.API.KVDelete(userID + GitlabUserInfoKey); err != nil {
		return errors.Wrap(err, "encountered error deleting GitLab user info")
	}
	return nil
}

func (p *Plugin) deleteGitlabUserToken(userID string) error {
	if err := p.API.KVDelete(userID + GitlabUserTokenKey); err != nil {
		return errors.Wrap(err, "encountered error deleting GitLab user token")
	}
	return nil
}

func (p *Plugin) getGitlabUserInfoByMattermostID(userID string) (*gitlab.UserInfo, *APIErrorResponse) {
	var userInfo gitlab.UserInfo

	infoBytes, err := p.API.KVGet(userID + GitlabUserInfoKey)

	if err != nil || infoBytes == nil {
		gitlabTokenBytes, appErr := p.API.KVGet(userID + GitlabMigrationTokenKey)
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

	gitlabTokenBytes, appErr := p.API.KVGet(userID + GitlabMigrationTokenKey)
	if appErr != nil || gitlabTokenBytes == nil {
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

	var userInfoWithoutToken = &gitlab.UserInfo{
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

	if appErr = p.API.KVDelete(userInfo.UserID + GitlabMigrationTokenKey); appErr != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to delete KV entry for migration.", StatusCode: http.StatusInternalServerError}
	}

	return userInfo.Token, nil
}

func (p *Plugin) getGitlabUserTokenByMattermostID(userID string) (*oauth2.Token, *APIErrorResponse) {
	config := p.getConfiguration()
	var token oauth2.Token

	tokenBytes, appErr := p.API.KVGet(userID + GitlabUserTokenKey)
	if appErr != nil || tokenBytes == nil {
		return nil, &APIErrorResponse{ID: APIErrorIDNotConnected, Message: "Must connect user account to GitLab first.", StatusCode: http.StatusBadRequest}
	}

	unencryptedToken, err := decrypt([]byte(config.EncryptionKey), string(tokenBytes))
	if err != nil {
		p.API.LogWarn("can't decrypt token", "err", err.Error())
		return nil, &APIErrorResponse{ID: "", Message: "Unable to decrypt access token.", StatusCode: http.StatusInternalServerError}
	}

	if err := json.Unmarshal([]byte(unencryptedToken), &token); err != nil {
		return nil, &APIErrorResponse{ID: "", Message: "Unable to parse token.", StatusCode: http.StatusInternalServerError}
	}

	return &token, nil
}

func (p *Plugin) storeGitlabToUserIDMapping(gitlabUsername, userID string) error {
	if err := p.API.KVSet(gitlabUsername+GitlabUsernameKey, []byte(userID)); err != nil {
		return errors.Wrap(err, "encountered error saving GitLab username mapping")
	}
	return nil
}

func (p *Plugin) storeGitlabIDToUserIDMapping(gitlabUsername string, gitlabID int) error {
	if err := p.API.KVSet(fmt.Sprintf("%d%s", gitlabID, GitlabIDUsernameKey), []byte(gitlabUsername)); err != nil {
		return errors.Wrap(err, "encountered error saving GitLab id mapping")
	}
	return nil
}

func (p *Plugin) deleteGitlabToUserIDMapping(gitlabUsername string) error {
	if err := p.API.KVDelete(gitlabUsername + GitlabUsernameKey); err != nil {
		return errors.Wrap(err, "encountered error deleting GitLab username mapping")
	}
	return nil
}

func (p *Plugin) deleteGitlabIDToUserIDMapping(gitlabID int) error {
	if err := p.API.KVDelete(fmt.Sprintf("%d%s", gitlabID, GitlabIDUsernameKey)); err != nil {
		return errors.Wrap(err, "encountered error deleting GitLab id mapping")
	}
	return nil
}

func (p *Plugin) getGitlabToUserIDMapping(gitlabUsername string) string {
	userID, err := p.API.KVGet(gitlabUsername + GitlabUsernameKey)
	if err != nil {
		p.API.LogWarn("can't get userId from store with username", "err", err.DetailedError, "username", gitlabUsername)
	}
	return string(userID)
}

func (p *Plugin) getGitlabIDToUsernameMapping(gitlabUserID string) string {
	gitlabUsername, err := p.API.KVGet(gitlabUserID + GitlabIDUsernameKey)
	if err != nil {
		p.API.LogWarn("can't get user id by login", "err", err.DetailedError)
	}
	return string(gitlabUsername)
}

func (p *Plugin) disconnectGitlabAccount(userID string) {
	userInfo, err := p.getGitlabUserInfoByMattermostID(userID)
	if err != nil {
		p.API.LogWarn("can't get GitLab user info from mattermost id", "err", err.Message)
		return
	}
	if userInfo == nil {
		return
	}

	if err := p.deleteGitlabUserInfo(userID); err != nil {
		p.API.LogWarn("can't delete user info in store", "err", err.Error, "userId", userID)
	}
	if err := p.deleteGitlabUserToken(userID); err != nil {
		p.API.LogWarn("can't delete token in store", "err", err.Error, "userId", userID)
	}
	if err := p.deleteGitlabToUserIDMapping(userInfo.GitlabUsername); err != nil {
		p.API.LogWarn("can't delete username in store", "err", err.Error, "username", userInfo.GitlabUsername)
	}
	if err := p.deleteGitlabIDToUserIDMapping(userInfo.GitlabUserID); err != nil {
		p.API.LogWarn("can't delete user id in store", "err", err.Error, "id", userInfo.GitlabUserID)
	}

	if user, err := p.API.GetUser(userID); err == nil && user.Props != nil && len(user.Props["git_user"]) > 0 {
		delete(user.Props, "git_user")
		if _, err := p.API.UpdateUser(user); err != nil {
			p.API.LogWarn("can't update user after delete git account", "err", err.DetailedError)
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
		p.API.LogWarn("Couldn't get bot's DM channel", "user_id", userID)
		return err
	}

	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: channel.Id,
		Message:   message,
		Type:      postType,
	}

	if _, err := p.API.CreatePost(post); err != nil {
		p.API.LogWarn("can't post DM", "err", err.DetailedError)
		return err
	}

	return nil
}

func (p *Plugin) PostToDo(ctx context.Context, info *gitlab.UserInfo) {
	hasTodo, text, err := p.GetToDo(ctx, info)
	if err != nil {
		p.API.LogWarn("can't post todo", "err", err.Error())
		return
	}
	if !hasTodo {
		return
	}

	if err := p.CreateBotDMPost(info.UserID, text, "custom_git_todo"); err != nil {
		p.API.LogWarn("can't create dm post in post todo", "err", err.DetailedError)
	}
}

func (p *Plugin) GetToDo(ctx context.Context, user *gitlab.UserInfo) (bool, string, error) {
	hasTodo := false
	g, ctx := errgroup.WithContext(ctx)

	notificationText := ""
	g.Go(func() error {
		var unreads []*gitlabLib.Todo
		err := p.useGitlabClient(user, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			resp, err := p.GitlabClient.GetUnreads(ctx, info, token)
			if err != nil {
				return err
			}
			unreads = resp
			return nil
		})
		if err != nil {
			return err
		}

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
			notificationText += "You don't have any unread messages.\n"
		} else {
			notificationText += fmt.Sprintf("You have %v unread messages:\n", notificationCount)
			notificationText += notificationContent

			hasTodo = true
		}

		return nil
	})

	reviewText := ""
	g.Go(func() error {
		var reviews []*gitlab.MergeRequest
		err := p.useGitlabClient(user, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			resp, err := p.GitlabClient.GetReviews(ctx, info, token)
			if err != nil {
				return err
			}
			reviews = resp
			return nil
		})
		if err != nil {
			return err
		}

		if len(reviews) == 0 {
			reviewText += "You don't have any merge requests awaiting your review.\n"
		} else {
			reviewText += fmt.Sprintf("You have %v merge requests awaiting your review:\n", len(reviews))

			for _, pr := range reviews {
				reviewText += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
			}

			hasTodo = true
		}

		return nil
	})

	assignmentText := ""
	g.Go(func() error {
		var yourAssignments []*gitlab.Issue
		err := p.useGitlabClient(user, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			resp, err := p.GitlabClient.GetYourAssignments(ctx, info, token)
			if err != nil {
				return err
			}
			yourAssignments = resp
			return nil
		})
		if err != nil {
			return err
		}

		if len(yourAssignments) == 0 {
			assignmentText += "You don't have any issues awaiting your dev.\n"
		} else {
			assignmentText += fmt.Sprintf("You have %v issues awaiting dev:\n", len(yourAssignments))

			for _, pr := range yourAssignments {
				assignmentText += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
			}

			hasTodo = true
		}

		return nil
	})

	mergeRequestText := ""
	g.Go(func() error {
		var mergeRequests []*gitlab.MergeRequest
		err := p.useGitlabClient(user, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			resp, err := p.GitlabClient.GetYourPrs(ctx, info, token)
			if err != nil {
				return err
			}
			mergeRequests = resp
			return nil
		})
		if err != nil {
			return err
		}

		if len(mergeRequests) == 0 {
			mergeRequestText += "You don't have any open merge requests.\n"
		} else {
			mergeRequestText += fmt.Sprintf("You have %v open merge requests:\n", len(mergeRequests))

			for _, pr := range mergeRequests {
				mergeRequestText += fmt.Sprintf("* [%v](%v)\n", pr.Title, pr.WebURL)
			}

			hasTodo = true
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		return false, "", err
	}

	text := "##### Unread Messages\n"
	text += notificationText

	text += "##### Review Requests\n"
	text += reviewText

	text += "##### Assignments\n"
	text += assignmentText

	text += "##### Your Open Merge Requests\n"
	text += mergeRequestText

	return hasTodo, text, nil
}

func (p *Plugin) isNamespaceAllowed(namespace string) error {
	allowedNamespace := strings.TrimSpace(p.getConfiguration().GitlabGroup)
	if allowedNamespace != "" && allowedNamespace != namespace && !strings.HasPrefix(namespace, allowedNamespace) {
		return errors.Errorf("only repositories in the %s namespace are allowed", allowedNamespace)
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
		p.API.LogWarn(
			"unable to marshal payload for updated channel subscriptions",
			"err", err.Error(),
		)
		return
	}

	p.API.PublishWebSocketEvent(
		WsChannelSubscriptionsUpdated,
		map[string]interface{}{"payload": string(payloadJSON)},
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

	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	found := false
	for _, hook := range hooks {
		if strings.Contains(hook.URL, siteURL) {
			found = true
		}
	}

	return found, err
}

func (p *Plugin) refreshToken(userInfo *gitlab.UserInfo, token *oauth2.Token) (*oauth2.Token, error) {
	conf := p.getOAuthConfig()
	src := conf.TokenSource(context.Background(), token)

	newToken, err := src.Token() // this actually goes and renews the tokens

	if err != nil {
		if strings.Contains(err.Error(), "\"error\":\"invalid_grant\"") {
			p.handleRevokedToken(userInfo)
		}
		return nil, errors.Wrap(err, "unable to get the new refreshed token")
	}

	if newToken.AccessToken != token.AccessToken {
		p.API.LogDebug("Gitlab token refreshed.", "UserID", userInfo.UserID)

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
		p.API.LogWarn("Error sending revoked token DM post", "err", err.Error())
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

	if time.Until(token.Expiry) > 1*time.Minute {
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

	if time.Until(lockedToken.Expiry) > 1*time.Minute {
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
		p.handleRevokedToken(info)
	}

	return err
}
