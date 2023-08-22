package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	gitlabLib "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-plugin-gitlab/server/subscription"
)

const (
	APIErrorIDNotConnected = "not_connected"

	requestTimeout = 30 * time.Second
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	p.router.ServeHTTP(w, r)
}

func (p *Plugin) initializeAPI() {
	p.router = mux.NewRouter()
	p.router.Use(p.withRecovery)

	oauthRouter := p.router.PathPrefix("/oauth").Subrouter()
	apiRouter := p.router.PathPrefix("/api/v1").Subrouter()
	apiRouter.Use(p.checkConfigured)

	p.router.HandleFunc("/webhook", p.handleWebhook).Methods(http.MethodPost)

	oauthRouter.HandleFunc("/connect", p.checkAuth(p.attachContext(p.connectUserToGitlab), ResponseTypePlain)).Methods(http.MethodGet)
	oauthRouter.HandleFunc("/complete", p.checkAuth(p.attachContext(p.completeConnectUserToGitlab), ResponseTypePlain)).Methods(http.MethodGet)

	apiRouter.HandleFunc("/connected", p.attachContext(p.getConnected)).Methods(http.MethodGet)

	apiRouter.HandleFunc("/user", p.checkAuth(p.attachContext(p.getGitlabUser), ResponseTypeJSON)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/todo", p.checkAuth(p.attachUserContext(p.postToDo), ResponseTypeJSON)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/reviews", p.checkAuth(p.attachUserContext(p.getReviews), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/assignedprs", p.checkAuth(p.attachUserContext(p.getYourAssignedPrs), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/prdetails", p.checkAuth(p.attachUserContext(p.getPrDetails), ResponseTypePlain)).Methods(http.MethodPost)
	apiRouter.HandleFunc("/assignedissues", p.checkAuth(p.attachUserContext(p.getYourAssignedIssues), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/todolist", p.checkAuth(p.attachUserContext(p.getToDoList), ResponseTypePlain)).Methods(http.MethodGet)
	apiRouter.HandleFunc("/settings", p.checkAuth(p.attachUserContext(p.updateSettings), ResponseTypePlain)).Methods(http.MethodPost)

	apiRouter.HandleFunc("/channel/{channel_id:[A-Za-z0-9]+}/subscriptions", p.checkAuth(p.attachUserContext(p.getChannelSubscriptions), ResponseTypeJSON)).Methods(http.MethodGet)
}

type Context struct {
	Ctx    context.Context
	UserID string
	Log    logger.Logger
}

func (p *Plugin) createContext(_ http.ResponseWriter, r *http.Request) (*Context, context.CancelFunc) {
	userID := r.Header.Get("Mattermost-User-ID")

	logger := logger.New(p.API).With(logger.LogContext{
		"userid": userID,
	})

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)

	context := &Context{
		Ctx:    ctx,
		UserID: userID,
		Log:    logger,
	}

	return context, cancel
}

// HTTPHandlerFuncWithContext is http.HandleFunc but with a Context attached
type HTTPHandlerFuncWithContext func(c *Context, w http.ResponseWriter, r *http.Request)

func (p *Plugin) attachContext(handler HTTPHandlerFuncWithContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		context, cancel := p.createContext(w, r)
		defer cancel()

		handler(context, w, r)
	}
}

type UserContext struct {
	Context
	GitlabInfo *gitlab.UserInfo
}

// HTTPHandlerFuncWithUserContext is http.HandleFunc but with a UserContext attached
type HTTPHandlerFuncWithUserContext func(c *UserContext, w http.ResponseWriter, r *http.Request)

func (p *Plugin) attachUserContext(handler HTTPHandlerFuncWithUserContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		context, cancel := p.createContext(w, r)
		defer cancel()

		info, apiErr := p.getGitlabUserInfoByMattermostID(context.UserID)
		if apiErr != nil {
			p.writeAPIError(w, apiErr)
			return
		}

		context.Log = context.Log.With(logger.LogContext{
			"gitlab username": info.GitlabUsername,
			"gitlab userid":   info.GitlabUserID,
		})

		userContext := &UserContext{
			Context:    *context,
			GitlabInfo: info,
		}

		handler(userContext, w, r)
	}
}

func (p *Plugin) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				p.client.Log.Warn("Recovered from a panic",
					"url", r.URL.String(),
					"error", x,
					"stack", string(debug.Stack()))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) checkConfigured(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := p.getConfiguration()

		if err := config.IsValid(); err != nil {
			http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) checkAuth(handler http.HandlerFunc, responseType ResponseType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			switch responseType {
			case ResponseTypeJSON:
				p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
			case ResponseTypePlain:
				http.Error(w, "Not authorized", http.StatusUnauthorized)
			default:
				p.client.Log.Debug("Unknown ResponseType detected")
			}
			return
		}

		handler(w, r)
	}
}

// ResponseType indicates type of response returned by api
type ResponseType string

const (
	// ResponseTypeJSON indicates that response type is json
	ResponseTypeJSON ResponseType = "JSON_RESPONSE"
	// ResponseTypePlain indicates that response type is text plain
	ResponseTypePlain ResponseType = "TEXT_RESPONSE"
)

type APIErrorResponse struct {
	ID         string `json:"id"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func (e *APIErrorResponse) Error() string {
	return e.Message
}

func (p *Plugin) writeAPIError(w http.ResponseWriter, err *APIErrorResponse) {
	b, _ := json.Marshal(err)
	w.WriteHeader(err.StatusCode)
	if _, err := w.Write(b); err != nil {
		p.client.Log.Warn("can't write api error http response", "err", err.Error())
	}
}

func (p *Plugin) writeAPIResponse(w http.ResponseWriter, resp interface{}) {
	b, jsonErr := json.Marshal(resp)
	if jsonErr != nil {
		p.client.Log.Warn("Error encoding JSON response", "err", jsonErr.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an unexpected error. Please try again.", StatusCode: http.StatusInternalServerError})
	}
	if _, err := w.Write(b); err != nil {
		p.client.Log.Warn("can't write response user to http", "err", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an unexpected error. Please try again.", StatusCode: http.StatusInternalServerError})
	}
}

func (p *Plugin) connectUserToGitlab(c *Context, w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	conf := p.getOAuthConfig()

	state := fmt.Sprintf("%v_%v", model.NewId()[0:15], userID)

	if _, err := p.client.KV.Set(state, []byte(state)); err != nil {
		c.Log.WithError(err).Warnf("Can't store state oauth2")
		http.Error(w, "can't store state oauth2", http.StatusInternalServerError)
		return
	}

	url := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)

	ch := p.oauthBroker.SubscribeOAuthComplete(userID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		var errorMsg string
		select {
		case err := <-ch:
			if err != nil {
				errorMsg = err.Error()
			}
		case <-ctx.Done():
			errorMsg = "Timed out waiting for OAuth connection. Please check if the SiteURL is correct."
		}

		if errorMsg != "" {
			_, err := p.poster.DMWithAttachments(userID, &model.SlackAttachment{
				Text:  fmt.Sprintf("There was an error connecting to your GitLab: `%s` Please double check your configuration.", errorMsg),
				Color: string(flow.ColorDanger),
			})
			if err != nil {
				c.Log.WithError(err).Warnf("Failed to DM with cancel information")
			}
		}

		p.oauthBroker.UnsubscribeOAuthComplete(userID, ch)
	}()

	http.Redirect(w, r, url, http.StatusFound)
}

func (p *Plugin) completeConnectUserToGitlab(c *Context, w http.ResponseWriter, r *http.Request) {
	authedUserID := r.Header.Get("Mattermost-User-ID")
	if authedUserID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var rErr error
	defer func() {
		p.oauthBroker.publishOAuthComplete(authedUserID, rErr, false)
	}()

	config := p.getConfiguration()

	conf := p.getOAuthConfig()

	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		rErr = errors.New("missing authorization code")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")

	var storedState []byte
	err := p.client.KV.Get(state, &storedState)
	if err != nil {
		c.Log.WithError(err).Warnf("Can't get state from store")

		rErr = errors.Wrap(err, "missing stored state")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
		return
	}

	err = p.client.KV.Delete(state)
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to delete state token")

		rErr = errors.Wrap(err, "error deleting stored state")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
	}

	if string(storedState) != state {
		rErr = errors.New("invalid state token")
		http.Error(w, rErr.Error(), http.StatusBadRequest)
		return
	}

	userID := strings.Split(state, "_")[1]

	if userID != authedUserID {
		rErr = errors.New("not authorized, incorrect user")
		http.Error(w, rErr.Error(), http.StatusUnauthorized)
		return
	}

	tok, err := conf.Exchange(c.Ctx, code)
	if err != nil {
		c.Log.WithError(err).Warnf("Can't exchange state")

		rErr = errors.Wrap(err, "Failed to exchange oauth code into token")
		http.Error(w, rErr.Error(), http.StatusInternalServerError)
		return
	}

	userInfo, err := p.GitlabClient.GetCurrentUser(c.Ctx, userID, *tok)
	if err != nil {
		c.Log.WithError(err).Warnf("Can't retrieve user info from gitLab API")

		rErr = errors.Wrap(err, "unable to connect user to GitLab")
		http.Error(w, rErr.Error(), http.StatusInternalServerError)
		return
	}

	if err = p.storeGitlabUserInfo(userInfo); err != nil {
		c.Log.WithError(err).Warnf("Can't store user info")

		rErr = errors.Wrap(err, "Unable to connect user to GitLab")
		http.Error(w, rErr.Error(), http.StatusInternalServerError)
		return
	}

	if err = p.storeGitlabUserToken(userInfo.UserID, tok); err != nil {
		c.Log.WithError(err).Warnf("Can't store user token")

		rErr = errors.Wrap(err, "Unable to connect user to GitLab")
		http.Error(w, rErr.Error(), http.StatusInternalServerError)
		return
	}

	if err = p.storeGitlabToUserIDMapping(userInfo.GitlabUsername, userID); err != nil {
		c.Log.WithError(err).Warnf("Can't store GitLab to user id mapping")
	}

	if err = p.storeGitlabIDToUserIDMapping(userInfo.GitlabUsername, userInfo.GitlabUserID); err != nil {
		c.Log.WithError(err).Warnf("Can't store GitLab to GitLab id mapping")
	}

	flow := p.flowManager.setupFlow.ForUser(authedUserID)

	stepName, err := flow.GetCurrentStep()
	if err != nil {
		c.Log.WithError(err).Warnf("Failed to get current step")
	}

	if stepName == stepOAuthConnect {
		err = flow.Go(stepWebhookQuestion)
		if err != nil {
			c.Log.WithError(err).Warnf("Failed go to next step")
		}
	} else {
		// Only post introduction message if no setup wizard is running
		// Post intro post
		message := fmt.Sprintf("#### Welcome to the Mattermost GitLab Plugin!\n"+
			"You've connected your Mattermost account to %s on GitLab. Read about the features of this plugin below:\n\n"+
			"##### Daily Reminders\n"+
			"The first time you log in each day, you will get a post right here letting you know what messages you need to read and what merge requests are awaiting your review.\n"+
			"Turn off reminders with `/gitlab settings reminders off`.\n\n"+
			"##### Notifications\n"+
			"When someone mentions you, requests your review, comments on or modifies one of your merge requests/issues, or assigns you, you'll get a post here about it.\n"+
			"Turn off notifications with `/gitlab settings notifications off`.\n\n"+
			"##### Sidebar Buttons\n"+
			"Check out the buttons in the left-hand sidebar of Mattermost.\n"+
			"* The first button tells you how many merge requests you are assigned to.\n"+
			"* The second shows the number of merge requests that are awaiting your review.\n"+
			"* The third shows the number of issues you are assigned to.\n"+
			"* The fourth tracks the number of todos you have.\n"+
			"* The fifth will refresh the numbers.\n\n"+
			"Click on them!\n\n"+
			"##### Slash Commands\n"+
			strings.ReplaceAll(commandHelp, "|", "`"), userInfo.GitlabUsername)

		if err := p.CreateBotDMPost(userID, message, "custom_git_welcome"); err != nil {
			c.Log.WithError(err).Warnf("Can't send help message with bot dm")
		}
	}

	p.TrackUserEvent("account_connected", userID, nil)

	p.client.Frontend.PublishWebSocketEvent(
		WsEventConnect,
		map[string]interface{}{
			"connected":        true,
			"gitlab_username":  userInfo.GitlabUsername,
			"gitlab_client_id": config.GitlabOAuthClientID,
			"gitlab_url":       config.GitlabURL,
			"organization":     config.GitlabGroup,
		},
		&model.WebsocketBroadcast{UserId: userID},
	)

	html := `
<!DOCTYPE html>
<html>
	<head>
		<script>
			window.close();
		</script>
	</head>
	<body>
		<p>Completed connecting to GitLab. Please close this window.</p>
	</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: ">Completed connecting to GitLab. Please close this window.", StatusCode: http.StatusInternalServerError})
	}
}

type ConnectedResponse struct {
	Connected      bool                 `json:"connected"`
	GitlabUsername string               `json:"gitlab_username"`
	GitlabClientID string               `json:"gitlab_client_id"`
	GitlabURL      string               `json:"gitlab_url,omitempty"`
	Organization   string               `json:"organization"`
	Settings       *gitlab.UserSettings `json:"settings"`
}

type GitlabUserRequest struct {
	UserID string `json:"user_id"`
}

type GitlabUserResponse struct {
	Username string `json:"username"`
}

func (p *Plugin) getGitlabUser(c *Context, w http.ResponseWriter, r *http.Request) {
	req := &GitlabUserRequest{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || req.UserID == "" {
		if err != nil {
			c.Log.WithError(err).Warnf("Error decoding JSON body")
		}
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a JSON object with a non-blank user_id field.", StatusCode: http.StatusBadRequest})
		return
	}

	userInfo, apiErr := p.getGitlabUserInfoByMattermostID(req.UserID)
	if apiErr != nil {
		if apiErr.ID == APIErrorIDNotConnected {
			p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "User is not connected to a GitLab account.", StatusCode: http.StatusNotFound})
		} else {
			p.writeAPIError(w, apiErr)
		}
		return
	}

	if userInfo == nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "User is not connected to a GitLab account.", StatusCode: http.StatusNotFound})
		return
	}

	p.writeAPIResponse(w, &GitlabUserResponse{Username: userInfo.GitlabUsername})
}

func (p *Plugin) getConnected(c *Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	resp := &ConnectedResponse{
		Connected:    false,
		GitlabURL:    config.GitlabURL,
		Organization: config.GitlabGroup,
	}

	info, _ := p.getGitlabUserInfoByMattermostID(c.UserID)
	if info != nil {
		resp.Connected = true
		resp.GitlabUsername = info.GitlabUsername
		resp.GitlabClientID = config.GitlabOAuthClientID
		resp.Settings = info.Settings

		if info.Settings.DailyReminder && r.URL.Query().Get("reminder") == "true" {
			lastPostAt := info.LastToDoPostAt

			var timezone *time.Location
			offset, _ := strconv.Atoi(r.Header.Get("X-Timezone-Offset"))
			timezone = time.FixedZone("local", -60*offset)

			// Post to do message if it's the next day and been more than an hour since the last post
			now := model.GetMillis()
			nt := time.Unix(now/1000, 0).In(timezone)
			lt := time.Unix(lastPostAt/1000, 0).In(timezone)
			if nt.Sub(lt).Hours() >= 1 && (nt.Day() != lt.Day() || nt.Month() != lt.Month() || nt.Year() != lt.Year()) {
				p.PostToDo(c.Ctx, info)
				info.LastToDoPostAt = now
				if err := p.storeGitlabUserInfo(info); err != nil {
					c.Log.WithError(err).Warnf("Can't store user info")
				}
			}
		}
	}

	p.writeAPIResponse(w, resp)
}

func (p *Plugin) getToDoList(c *UserContext, w http.ResponseWriter, r *http.Request) {
	var result []*gitlabLib.Todo
	err := p.useGitlabClient(c.GitlabInfo, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetToDoList(c.Ctx, info, token)
		if err != nil {
			return err
		}
		result = resp
		return nil
	})

	if err != nil {
		c.Log.WithError(err).Warnf("Unable to list todos in GitLab API")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to list todos in GitLab API.", StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, result)
}

func (p *Plugin) getReviews(c *UserContext, w http.ResponseWriter, r *http.Request) {
	var result []*gitlab.MergeRequest
	err := p.useGitlabClient(c.GitlabInfo, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetReviews(c.Ctx, info, token)
		if err != nil {
			return err
		}
		result = resp
		return nil
	})

	if err != nil {
		c.Log.WithError(err).Warnf("Unable to list merge-request where assignee in GitLab API")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to list merge-request in GitLab API.", StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, result)
}

func (p *Plugin) getYourAssignedPrs(c *UserContext, w http.ResponseWriter, r *http.Request) {
	var assignedPRs []*gitlab.MergeRequest
	err := p.useGitlabClient(c.GitlabInfo, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetYourAssignedPrs(c.Ctx, info, token)
		if err != nil {
			return err
		}
		assignedPRs = resp
		return nil
	})
	if err != nil {
		c.Log.WithError(err).Warnf("Can't list merge-request where author in GitLab API")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to list merge-request in GitLab API.", StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, assignedPRs)
}

func (p *Plugin) getPrDetails(c *UserContext, w http.ResponseWriter, r *http.Request) {
	var prList []*gitlab.PRDetails
	if err := json.NewDecoder(r.Body).Decode(&prList); err != nil {
		c.Log.WithError(err).Warnf("Error decoding PRDetails JSON body")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("Error decoding PRDetails JSON body. Error: %s", err.Error()), StatusCode: http.StatusBadRequest})
		return
	}
	var result []*gitlab.PRDetails
	err := p.useGitlabClient(c.GitlabInfo, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetYourPrDetails(c.Ctx, c.Log, info, token, prList)
		if err != nil {
			return err
		}
		result = resp
		return nil
	})
	if err != nil {
		c.Log.WithError(err).Warnf("Can't list merge-request details in GitLab API")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: fmt.Sprintf("Can't list merge-request details in GitLab API. Error: %s", err.Error()), StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, result)
}

func (p *Plugin) getYourAssignedIssues(c *UserContext, w http.ResponseWriter, r *http.Request) {
	var result []*gitlab.Issue
	err := p.useGitlabClient(c.GitlabInfo, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		resp, err := p.GitlabClient.GetYourAssignedIssues(c.Ctx, info, token)
		if err != nil {
			return err
		}
		result = resp
		return nil
	})

	if err != nil {
		c.Log.WithError(err).Warnf("Unable to list issue where assignee in GitLab API")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to list issue in GitLab API.", StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, result)
}

func (p *Plugin) postToDo(c *UserContext, w http.ResponseWriter, r *http.Request) {
	_, text, err := p.GetToDo(c.Ctx, c.GitlabInfo)
	if err != nil {
		c.Log.WithError(err).Warnf("Can't get todo")
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error getting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	if err := p.CreateBotDMPost(c.UserID, text, "custom_git_todo"); err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error posting the to do items.", StatusCode: http.StatusUnauthorized})
	}

	p.writeAPIResponse(w, struct{ status string }{status: "OK"})
}

func (p *Plugin) updateSettings(c *UserContext, w http.ResponseWriter, r *http.Request) {
	var settings *gitlab.UserSettings
	err := json.NewDecoder(r.Body).Decode(&settings)
	if settings == nil || err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	info, apiErr := p.getGitlabUserInfoByMattermostID(c.UserID)
	if apiErr != nil {
		p.writeAPIError(w, apiErr)
		return
	}

	info.Settings = settings

	if err := p.storeGitlabUserInfo(info); err != nil {
		c.Log.WithError(err).Errorf("can't store GitLab user info when update settings")
		http.Error(w, "Encountered error updating settings", http.StatusInternalServerError)
	}

	p.writeAPIResponse(w, info.Settings)
}

type SubscriptionResponse struct {
	RepositoryName string   `json:"repository_name"`
	RepositoryURL  string   `json:"repository_url"`
	Features       []string `json:"features"`
	CreatorID      string   `json:"creator_id"`
}

func subscriptionsToResponse(config *configuration, subscriptions []*subscription.Subscription) []SubscriptionResponse {
	gitlabURL, _ := url.Parse(config.GitlabURL)

	subscriptionResponses := make([]SubscriptionResponse, 0, len(subscriptions))

	for _, subscription := range subscriptions {
		features := []string{}
		if len(subscription.Features) > 0 {
			features = strings.Split(subscription.Features, ",")
		}

		repositoryURL := *gitlabURL
		repositoryURL.Path = path.Join(gitlabURL.EscapedPath(), subscription.Repository)

		subscriptionResponses = append(subscriptionResponses, SubscriptionResponse{
			RepositoryName: subscription.Repository,
			RepositoryURL:  repositoryURL.String(),
			Features:       features,
			CreatorID:      subscription.CreatorID,
		})
	}

	return subscriptionResponses
}

func (p *Plugin) getChannelSubscriptions(c *UserContext, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	channelID := vars["channel_id"]

	if !p.client.User.HasPermissionToChannel(c.UserID, channelID, model.PermissionReadChannel) {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	config := p.getConfiguration()
	subscriptions, err := p.GetSubscriptionsByChannel(channelID)
	if err != nil {
		p.client.Log.Warn("unable to get subscriptions by channel", "err", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to get subscriptions by channel.", StatusCode: http.StatusInternalServerError})
		return
	}

	resp := subscriptionsToResponse(config, subscriptions)

	b, err := json.Marshal(resp)
	if err != nil {
		p.client.Log.Warn("failed to marshal channel subscriptions response", "err", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an unexpected error. Please try again.", StatusCode: http.StatusInternalServerError})
	} else if _, err := w.Write(b); err != nil {
		p.client.Log.Warn("can't write api error http response", "err", err.Error())
	}
}
