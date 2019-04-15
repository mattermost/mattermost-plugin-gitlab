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

	"github.com/manland/go-gitlab"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"

	"golang.org/x/oauth2"
)

const (
	API_ERROR_ID_NOT_CONNECTED = "not_connected"
	GITLAB_USERNAME            = "Gitlab Plugin"
)

type APIErrorResponse struct {
	ID         string `json:"id"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func writeAPIError(w http.ResponseWriter, err *APIErrorResponse) {
	b, _ := json.Marshal(err)
	w.WriteHeader(err.StatusCode)
	w.Write(b)
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
	case "/api/v1/mentions":
		p.getMentions(w, r)
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
		p.API.LogError("can't sotre state oauth2", "err", err.DetailedError)
		http.Error(w, "can't sotre state oauth2", http.StatusInternalServerError)
		return
	}

	url := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)

	http.Redirect(w, r, url, http.StatusFound)
}

func (p *Plugin) completeConnectUserToGitlab(w http.ResponseWriter, r *http.Request) {
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

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		p.API.LogError("can't exchange state", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := p.gitlabConnect(*tok)
	gitUser, _, err := client.Users.CurrentUser()
	if err != nil {
		p.API.LogError("can't retreive user info from gitlab", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userInfo := &GitlabUserInfo{
		UserID:         userID,
		GitlabUserId:   gitUser.ID,
		Token:          tok,
		GitlabUsername: gitUser.Username,
		LastToDoPostAt: model.GetMillis(),
		Settings: &UserSettings{
			SidebarButtons: SETTING_BUTTONS_TEAM,
			DailyReminder:  true,
			Notifications:  true,
		},
		AllowedPrivateRepos: config.EnablePrivateRepo,
	}

	if err := p.storeGitlabUserInfo(userInfo); err != nil {
		p.API.LogError("can't store user info", "err", err.Error())
		http.Error(w, "Unable to connect user to Gitlab", http.StatusInternalServerError)
		return
	}

	if err := p.storeGitlabToUserIDMapping(userInfo.GitlabUsername, userID); err != nil {
		p.API.LogError("can't store user id mapping", "err", err.Error())
	}

	// Post intro post
	message := fmt.Sprintf("#### Welcome to the Mattermost Gitlab Plugin!\n"+
		"You've connected your Mattermost account to %s on Gitlab. Read about the features of this plugin below:\n\n"+
		"##### Daily Reminders\n"+
		"The first time you log in each day, you will get a post right here letting you know what messages you need to read and what pull requests are awaiting your review.\n"+
		"Turn off reminders with `/gitlab settings reminders off`.\n\n"+
		"##### Notifications\n"+
		"When someone mentions you, requests your review, comments on or modifies one of your pull requests/issues, or assigns you, you'll get a post here about it.\n"+
		"Turn off notifications with `/gitlab settings notifications off`.\n\n"+
		"##### Sidebar Buttons\n"+
		"Check out the buttons in the left-hand sidebar of Mattermost.\n"+
		"* The first button tells you how many pull requests you have submitted.\n"+
		"* The second shows the number of PR that are awaiting your review.\n"+
		"* The third shows the number of PR and issues your are assiged to.\n"+
		"* The fourth tracks the number of unread messages you have.\n"+
		"* The fifth will refresh the numbers.\n\n"+
		"Click on them!\n\n"+
		"##### Slash Commands\n"+
		strings.Replace(COMMAND_HELP, "|", "`", -1), userInfo.GitlabUsername)

	if err := p.CreateBotDMPost(userID, message, "custom_git_welcome"); err != nil {
		p.API.LogError("can't send help message with bot dm", "err", err.Error())
	}

	p.API.PublishWebSocketEvent(
		WS_EVENT_CONNECT,
		map[string]interface{}{
			"connected":        true,
			"gitlab_username":  userInfo.GitlabUsername,
			"gitlab_client_id": config.GitlabOAuthClientID,
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
		<p>Completed connecting to Gitlab. Please close this window.</p>
	</body>
</html>
`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (p *Plugin) handleProfileImage(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	img, err := os.Open(filepath.Join(config.PluginsDirectory, manifest.Id, "assets", "profile.png"))
	if err != nil {
		http.NotFound(w, r)
		p.API.LogError("Unable to read gitlab profile image", "err", err.Error())
		return
	}
	defer img.Close()

	w.Header().Set("Content-Type", "image/png")
	_, err = io.Copy(w, img)
	if err != nil {
		p.API.LogError("can't copy image profile to http response writer", "err", err.Error())
	}
}

type ConnectedResponse struct {
	Connected         bool          `json:"connected"`
	GitlabUsername    string        `json:"gitlab_username"`
	GitlabClientID    string        `json:"gitlab_client_id"`
	EnterpriseBaseURL string        `json:"enterprise_base_url,omitempty"`
	Organization      string        `json:"organization"`
	Settings          *UserSettings `json:"settings"`
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
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	req := &GitlabUserRequest{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil || req.UserID == "" {
		if err != nil {
			mlog.Error("Error decoding JSON body: " + err.Error())
		}
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Please provide a JSON object with a non-blank user_id field.", StatusCode: http.StatusBadRequest})
		return
	}

	userInfo, apiErr := p.getGitlabUserInfoByMattermostID(req.UserID)
	if apiErr != nil {
		if apiErr.ID == API_ERROR_ID_NOT_CONNECTED {
			writeAPIError(w, &APIErrorResponse{ID: "", Message: "User is not connected to a Gitlab account.", StatusCode: http.StatusNotFound})
		} else {
			writeAPIError(w, apiErr)
		}
		return
	}

	if userInfo == nil {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "User is not connected to a Gitlab account.", StatusCode: http.StatusNotFound})
		return
	}

	resp := &GitlabUserResponse{Username: userInfo.GitlabUsername}
	b, jsonErr := json.Marshal(resp)
	if jsonErr != nil {
		mlog.Error("Error encoding JSON response: " + jsonErr.Error())
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an unexpected error. Please try again.", StatusCode: http.StatusInternalServerError})
	}
	w.Write(b)
}

func (p *Plugin) getConnected(w http.ResponseWriter, r *http.Request) {
	config := p.getConfiguration()

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	resp := &ConnectedResponse{
		Connected:         false,
		EnterpriseBaseURL: config.EnterpriseBaseURL,
		Organization:      config.GitlabGroup,
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
					p.API.LogError("can't sotre user info", "err", err.Error())
				}
			}
		}

		privateRepoStoreKey := info.UserID + GITLAB_PRIVATE_REPO_KEY
		if config.EnablePrivateRepo && !info.AllowedPrivateRepos {
			hasBeenNotified := false
			if val, err := p.API.KVGet(privateRepoStoreKey); err == nil {
				hasBeenNotified = val != nil
			} else {
				p.API.LogError("Unable to get private repo key value", "err", err.Error())
			}

			if !hasBeenNotified {
				if err := p.CreateBotDMPost(info.UserID, "Private repositories have been enabled for this plugin. To be able to use them you must disconnect and reconnect your Gitlab account. To reconnect your account, use the following slash commands: `/gitlab disconnect` followed by `/gitlab connect`.", ""); err != nil {
					p.API.LogError("Unable to send DM post about private config change", "err", err.Error())
				}
				if err := p.API.KVSet(privateRepoStoreKey, []byte("1")); err != nil {
					p.API.LogError("Unable to set private repo key value", "err", err.Error())
				}
			}
		}
	}

	b, _ := json.Marshal(resp)
	w.Write(b)
}

func (p *Plugin) getMentions(w http.ResponseWriter, r *http.Request) {

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var client *gitlab.Client

	if info, err := p.getGitlabUserInfoByMattermostID(userID); err != nil {
		writeAPIError(w, err)
		return
	} else {
		client = p.gitlabConnect(*info.Token)
	}

	result, _, err := client.Search.Issues("", &gitlab.SearchOptions{}) //TODO what mention means in gitlab ?
	if err != nil {
		mlog.Error(err.Error())
	}

	resp, _ := json.Marshal(result)
	w.Write(resp)
}

func (p *Plugin) getUnreads(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var client *gitlab.Client

	if info, err := p.getGitlabUserInfoByMattermostID(userID); err != nil {
		writeAPIError(w, err)
		return
	} else {
		client = p.gitlabConnect(*info.Token)
	}

	notifications, _, err := client.Todos.ListTodos(&gitlab.ListTodosOptions{})
	if err != nil {
		mlog.Error(err.Error())
	}

	resp, _ := json.Marshal(notifications)
	w.Write(resp)
}

func (p *Plugin) getReviews(w http.ResponseWriter, r *http.Request) {
	// config := p.getConfiguration()
	// TODO only for a group ?

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var client *gitlab.Client
	user, err := p.getGitlabUserInfoByMattermostID(userID)

	if err != nil {
		writeAPIError(w, err)
		return
	}
	client = p.gitlabConnect(*user.Token)
	opened := "opened"
	scope := "all"

	result, _, errRequest := client.MergeRequests.ListMergeRequests(&gitlab.ListMergeRequestsOptions{
		AssigneeID: &user.GitlabUserId,
		State:      &opened,
		Scope:      &scope,
	})

	if errRequest != nil {
		mlog.Error(errRequest.Error())
		return
	}

	resp, _ := json.Marshal(result)
	w.Write(resp)
}

func (p *Plugin) getYourPrs(w http.ResponseWriter, r *http.Request) {
	// config := p.getConfiguration()
	// TODO only for a group ?

	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var client *gitlab.Client
	user, err := p.getGitlabUserInfoByMattermostID(userID)

	if err != nil {
		writeAPIError(w, err)
		return
	}
	client = p.gitlabConnect(*user.Token)
	opened := "opened"

	result, _, errRequest := client.MergeRequests.ListMergeRequests(&gitlab.ListMergeRequestsOptions{
		AuthorID: &user.GitlabUserId,
		State:    &opened,
	})
	if errRequest != nil {
		mlog.Error(errRequest.Error())
		return
	}

	resp, _ := json.Marshal(result)
	w.Write(resp)
}

func (p *Plugin) getYourAssignments(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var client *gitlab.Client
	user, err := p.getGitlabUserInfoByMattermostID(userID)

	if err != nil {
		writeAPIError(w, err)
		return
	}
	client = p.gitlabConnect(*user.Token)
	opened := "opened"

	result, _, errRequest := client.Issues.ListIssues(&gitlab.ListIssuesOptions{
		AssigneeID: &user.GitlabUserId,
		State:      &opened,
	})
	if errRequest != nil {
		mlog.Error(errRequest.Error())
		return
	}

	resp, _ := json.Marshal(result)
	w.Write(resp)
}

func (p *Plugin) postToDo(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Not authorized.", StatusCode: http.StatusUnauthorized})
		return
	}

	var client *gitlab.Client
	user, err := p.getGitlabUserInfoByMattermostID(userID)

	if err != nil {
		writeAPIError(w, err)
		return
	}

	client = p.gitlabConnect(*user.Token)

	text, errRequest := p.GetToDo(user, client)
	if errRequest != nil {
		mlog.Error(errRequest.Error())
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error getting the to do items.", StatusCode: http.StatusUnauthorized})
		return
	}

	if err := p.CreateBotDMPost(userID, text, "custom_git_todo"); err != nil {
		writeAPIError(w, &APIErrorResponse{ID: "", Message: "Encountered an error posting the to do items.", StatusCode: http.StatusUnauthorized})
	}

	w.Write([]byte("{\"status\": \"OK\"}"))
}

func (p *Plugin) updateSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	var settings *UserSettings
	json.NewDecoder(r.Body).Decode(&settings)
	if settings == nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	info, err := p.getGitlabUserInfoByMattermostID(userID)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	info.Settings = settings

	if err := p.storeGitlabUserInfo(info); err != nil {
		mlog.Error(err.Error())
		http.Error(w, "Encountered error updating settings", http.StatusInternalServerError)
	}

	resp, _ := json.Marshal(info.Settings)
	w.Write(resp)
}
