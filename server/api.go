package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

const (
	APIErrorIDNotConnected = "not_connected"
)

type APIErrorResponse struct {
	ID         string `json:"id"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func (p *Plugin) writeAPIError(w http.ResponseWriter, err *APIErrorResponse) {
	b, _ := json.Marshal(err)
	w.WriteHeader(err.StatusCode)
	if _, err := w.Write(b); err != nil {
		p.API.LogError("can't write api error http response", "err", err.Error())
	}
}

func (p *Plugin) writeAPIResponse(w http.ResponseWriter, resp interface{}) {
	b, jsonErr := json.Marshal(resp)
	if jsonErr != nil {
		p.API.LogError("Error encoding JSON response", "err", jsonErr.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an unexpected error. Please try again.", StatusCode: http.StatusInternalServerError})
	}
	if _, err := w.Write(b); err != nil {
		p.API.LogError("can't write response user to http", "err", err.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an unexpected error. Please try again.", StatusCode: http.StatusInternalServerError})
	}
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	if err := config.IsValid(); err != nil {
		http.Error(w, "This plugin is not configured.", http.StatusNotImplemented)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch path := r.URL.Path; path {
	case "/webhook":
		p.handleWebhook(w, r)
	case "/assets/profile.png":
		p.handleProfileImage(w, r)
	case "/oauth/connect":
		p.connectUserToGitlab(w, r)
	case "/oauth/complete":
		p.completeConnectUserToGitlab(w, r)
	case "/api/v1/connected":
		p.getConnected(w, r)
	case "/api/v1/todo":
		p.postToDo(w, r)
	case "/api/v1/reviews":
		p.getReviews(w, r)
	case "/api/v1/yourprs":
		p.getYourPrs(w, r)
	case "/api/v1/yourassignments":
		p.getYourAssignments(w, r)
	case "/api/v1/unreads":
		p.getUnreads(w, r)
	case "/api/v1/settings":
		p.updateSettings(w, r)
	case "/api/v1/user":
		p.getGitlabUser(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) connectUserToGitlab(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	conf := p.getOAuthConfig()

	state := fmt.Sprintf("%v_%v", model.NewId()[0:15], userID)

	if err := p.API.KVSet(state, []byte(state)); err != nil {
		p.API.LogError("can't store state oauth2", "err", err.DetailedError)
		http.Error(w, "can't store state oauth2", http.StatusInternalServerError)
		return
	}

	url := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)

	http.Redirect(w, r, url, http.StatusFound)
}

func (p *Plugin) completeConnectUserToGitlab(w http.ResponseWriter, r *http.Request) {
	authedUserID := r.Header.Get("Mattermost-User-ID")
	if authedUserID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	config := p.getConfiguration()

	ctx := context.Background()
	conf := p.getOAuthConfig()

	code := r.URL.Query().Get("code")
	if len(code) == 0 {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")

	if storedState, err := p.API.KVGet(state); err != nil {
		p.API.LogError("can't get state from store", "err", err.Error())
		http.Error(w, "missing stored state", http.StatusBadRequest)
		return
	} else if string(storedState) != state {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	userID := strings.Split(state, "_")[1]

	if err := p.API.KVDelete(state); err != nil {
		p.API.LogError("can't delete state in store", "err", err.DetailedError)
	}

	if userID != authedUserID {
		http.Error(w, "Not authorized, incorrect user", http.StatusUnauthorized)
		return
	}

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		p.API.LogError("can't exchange state", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userInfo, err := p.GitlabClient.GetCurrentUser(userID, *tok)
	if err != nil {
		p.API.LogError("can't retrieve user info from gitLab API", "err", err.Error())
		http.Error(w, "Unable to connect user to GitLab", http.StatusInternalServerError)
		return
	}

	if err := p.storeGitlabUserInfo(userInfo); err != nil {
		p.API.LogError("can't store user info", "err", err.Error())
		http.Error(w, "Unable to connect user to GitLab", http.StatusInternalServerError)
		return
	}

	if err := p.storeGitlabToUserIDMapping(userInfo.GitlabUsername, userID); err != nil {
		p.API.LogError("can't store GitLab to user id mapping", "err", err.Error())
	}

	if err := p.storeGitlabIDToUserIDMapping(userInfo.GitlabUsername, userInfo.GitlabUserID); err != nil {
		p.API.LogError("can't store GitLab to GitLab id mapping", "err", err.Error())
	}

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
		"* The first button tells you how many merge requests you have submitted.\n"+
		"* The second shows the number of merge requests that are awaiting your review.\n"+
		"* The third shows the number of merge requests and issues you are assigned to.\n"+
		"* The fourth tracks the number of unread messages you have.\n"+
		"* The fifth will refresh the numbers.\n\n"+
		"Click on them!\n\n"+
		"##### Slash Commands\n"+
		strings.ReplaceAll(commandHelp, "|", "`"), userInfo.GitlabUsername)

	if err := p.CreateBotDMPost(userID, message, "custom_git_welcome"); err != nil {
		p.API.LogError("can't send help message with bot dm", "err", err.Error())
	}

	p.API.PublishWebSocketEvent(
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

func (p *Plugin) handleProfileImage(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	img, err := os.Open(filepath.Join(config.PluginsDirectory, manifest.ID, "assets", "profile.png"))
	if err != nil {
		http.NotFound(w, r)
		p.API.LogError("Unable to read GitLab profile image", "err", err.Error())
		return
	}
	defer func() {
		if err = img.Close(); err != nil {
			p.API.LogError("can't close img", "err", err.Error())
		}
	}()

	w.Header().Set("Content-Type", "image/png")
	_, err = io.Copy(w, img)
	if err != nil {
		p.API.LogError("can't copy image profile to http response writer", "err", err.Error())
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

func (p *Plugin) getGitlabUser(w http.ResponseWriter, r *http.Request) {
	requestorID := r.Header.Get("Mattermost-User-ID")
	if requestorID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	req := &GitlabUserRequest{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || req.UserID == "" {
		if err != nil {
			p.API.LogError("Error decoding JSON body", "err", err.Error())
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

func (p *Plugin) getConnected(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	resp := &ConnectedResponse{
		Connected:    false,
		GitlabURL:    config.GitlabURL,
		Organization: config.GitlabGroup,
	}

	info, _ := p.getGitlabUserInfoByMattermostID(userID)
	if info != nil && info.Token != nil {
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
				p.PostToDo(info)
				info.LastToDoPostAt = now
				if err := p.storeGitlabUserInfo(info); err != nil {
					p.API.LogError("can't store user info", "err", err.Error())
				}
			}
		}
	}

	p.writeAPIResponse(w, resp)
}

func (p *Plugin) getUnreads(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	user, err := p.getGitlabUserInfoByMattermostID(userID)
	if err != nil {
		p.writeAPIError(w, err)
		return
	}

	result, errRequest := p.GitlabClient.GetUnreads(user)
	if errRequest != nil {
		p.API.LogError("unable to list unreads in GitLab API", "err", errRequest.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to list unreads in GitLab API.", StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, result)
}

func (p *Plugin) getReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	user, err := p.getGitlabUserInfoByMattermostID(userID)

	if err != nil {
		p.writeAPIError(w, err)
		return
	}

	result, errRequest := p.GitlabClient.GetReviews(user)

	if errRequest != nil {
		p.API.LogError("unable to list merge-request where assignee in GitLab API", "err", errRequest.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to list merge-request in GitLab API.", StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, result)
}

func (p *Plugin) getYourPrs(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	user, err := p.getGitlabUserInfoByMattermostID(userID)

	if err != nil {
		p.writeAPIError(w, err)
		return
	}

	result, errRequest := p.GitlabClient.GetYourPrs(user)

	if errRequest != nil {
		p.API.LogError("can't list merge-request where author in GitLab API", "err", errRequest.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to list merge-request in GitLab API.", StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, result)
}

func (p *Plugin) getYourAssignments(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	user, err := p.getGitlabUserInfoByMattermostID(userID)

	if err != nil {
		p.writeAPIError(w, err)
		return
	}

	result, errRequest := p.GitlabClient.GetYourAssignments(user)

	if errRequest != nil {
		p.API.LogError("unable to list issue where assignee in GitLab API", "err", errRequest.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Unable to list issue in GitLab API.", StatusCode: http.StatusInternalServerError})
		return
	}

	p.writeAPIResponse(w, result)
}

func (p *Plugin) postToDo(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	user, err := p.getGitlabUserInfoByMattermostID(userID)

	if err != nil {
		p.writeAPIError(w, err)
		return
	}

	text, errRequest := p.GetToDo(user)
	if errRequest != nil {
		p.API.LogError("can't get todo", "err", errRequest.Error())
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error getting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	if err := p.CreateBotDMPost(userID, text, "custom_git_todo"); err != nil {
		p.writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error posting the to do items.", StatusCode: http.StatusUnauthorized})
	}

	p.writeAPIResponse(w, struct{ status string }{status: "OK"})
}

func (p *Plugin) updateSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var settings *gitlab.UserSettings
	err := json.NewDecoder(r.Body).Decode(&settings)
	if settings == nil || err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	info, errGitlab := p.getGitlabUserInfoByMattermostID(userID)
	if errGitlab != nil {
		p.writeAPIError(w, errGitlab)
		return
	}

	info.Settings = settings

	if err := p.storeGitlabUserInfo(info); err != nil {
		p.API.LogError("can't store GitLab user info when update settings", "err", err.Error())
		http.Error(w, "Encountered error updating settings", http.StatusInternalServerError)
	}

	p.writeAPIResponse(w, info.Settings)
}
