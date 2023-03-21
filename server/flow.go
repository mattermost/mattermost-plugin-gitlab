package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

type Tracker interface {
	TrackEvent(event string, properties map[string]interface{})
	TrackUserEvent(event, userID string, properties map[string]interface{})
}

type FlowManager struct {
	client                          *pluginapi.Client
	pluginURL                       string
	botUserID                       string
	router                          *mux.Router
	getConfiguration                func() *configuration
	getGitlabUserInfoByMattermostID func(userID string) (*gitlab.UserInfo, *APIErrorResponse)
	getGitlabClient                 func() gitlab.Gitlab

	tracker Tracker

	setupFlow        *flow.Flow
	oauthFlow        *flow.Flow
	webhokFlow       *flow.Flow
	announcementFlow *flow.Flow
}

func (p *Plugin) NewFlowManager() *FlowManager {
	fm := &FlowManager{
		client:                          p.client,
		pluginURL:                       *p.client.Configuration.GetConfig().ServiceSettings.SiteURL + "/" + "plugins" + "/" + manifest.Id,
		botUserID:                       p.BotUserID,
		router:                          p.router,
		getConfiguration:                p.getConfiguration,
		getGitlabUserInfoByMattermostID: p.getGitlabUserInfoByMattermostID,
		getGitlabClient:                 p.getGitlabClient,

		tracker: p,
	}

	fm.setupFlow = fm.newFlow("setup").WithSteps(
		fm.stepWelcome(),

		fm.stepDelegateQuestion(),
		fm.stepDelegateConfirmation(),
		fm.stepDelegateComplete(),

		fm.stepInstanceURL(),
		fm.stepOAuthInfo(),
		fm.stepOAuthInput(),
		fm.stepOAuthConnect(),

		fm.stepWebhookQuestion(),
		fm.stepWebhookWarning(),
		fm.stepWebhookConfirmation(),

		fm.stepAnnouncementQuestion(),
		fm.stepAnnouncementConfirmation(),

		fm.doneStep(),

		fm.stepCancel("setup"),
	)

	fm.oauthFlow = fm.newFlow("oauth").WithSteps(
		fm.stepInstanceURL(),
		fm.stepOAuthInfo(),
		fm.stepOAuthInfo(),
		fm.stepOAuthInput(),
		fm.stepOAuthConnect().Terminal(),

		fm.stepCancel("setup oauth"),
	)
	fm.webhokFlow = fm.newFlow("webhook").WithSteps(
		fm.stepWebhookQuestion(),
		flow.NewStep(stepWebhookConfirmation).
			WithText("Use `/gitlab subscriptions add` to subscribe any Mattermost channel to your GitLab repository. [Learn more](https://mattermost.gitbook.io/plugin-gitlab/feature-summary#subscribe-to-unsubscribe-from-a-repository)").
			Terminal(),

		fm.stepCancel("setup webhook"),
	)
	fm.announcementFlow = fm.newFlow("announcement").WithSteps(
		fm.stepAnnouncementQuestion(),
		fm.stepAnnouncementConfirmation().Terminal(),

		fm.stepCancel("setup announcement"),
	)

	return fm
}

func (fm *FlowManager) doneStep() flow.Step {
	return flow.NewStep(stepDone).
		WithText(":tada: You successfully installed GitLab.").
		OnRender(fm.onDone).Terminal()
}

func (fm *FlowManager) onDone(f *flow.Flow) {
	fm.trackCompleteSetupWizard(f.UserID)

	delegatedFrom := f.GetState().GetString(keyDelegatedFrom)
	if delegatedFrom != "" {
		err := fm.setupFlow.ForUser(delegatedFrom).Go(stepDelegateComplete)
		fm.client.Log.Warn("failed start configuration wizard for delegate", "error", err)
	}
}

func (fm *FlowManager) newFlow(name flow.Name) *flow.Flow {
	flow := flow.NewFlow(
		name,
		fm.client,
		fm.pluginURL,
		fm.botUserID,
	)

	flow.InitHTTP(fm.router)

	return flow
}

const (
	// Delegate Steps

	stepDelegateQuestion     flow.Name = "delegate-question"
	stepDelegateConfirmation flow.Name = "delegate-confirmation"
	stepDelegateComplete     flow.Name = "delegate-complete"

	// OAuth steps

	stepGitlabURL    flow.Name = "gitlab-url"
	stepOAuthInfo    flow.Name = "oauth-info"
	stepOAuthInput   flow.Name = "oauth-input"
	stepOAuthConnect flow.Name = "oauth-connect"

	// Webhook steps

	stepWebhookQuestion     flow.Name = "webhook-question"
	stepWebhookWarning      flow.Name = "webhook-warning"
	stepWebhookConfirmation flow.Name = "webhook-confirmation"

	// Announcement steps

	stepAnnouncementQuestion     flow.Name = "announcement-question"
	stepAnnouncementConfirmation flow.Name = "announcement-confirmation"

	// Miscellaneous Steps

	stepWelcome flow.Name = "welcome"
	stepDone    flow.Name = "done"
	stepCancel  flow.Name = "cancel"

	keyDelegatedFrom               = "DelegatedFrom"
	keyDelegatedTo                 = "DelegatedTo"
	keyGitlabURL                   = "GitlabURL"
	keyUsePreregisteredApplication = "UsePreregisteredApplication"
	keyIsOAuthConfigured           = "IsOAuthConfigured"
)

func cancelButton() flow.Button {
	return flow.Button{
		Name:    "Cancel setup",
		Color:   flow.ColorDanger,
		OnClick: flow.Goto(stepCancel),
	}
}

func (fm *FlowManager) stepCancel(command string) flow.Step {
	return flow.NewStep(stepCancel).
		Terminal().
		WithText(fmt.Sprintf("Gitlab integration setup has stopped. Restart setup later by running `/gitlab %s`. Learn more about the plugin [here](https://mattermost.gitbook.io/plugin-gitlab/).", command)).
		WithColor(flow.ColorDanger)
}

func continueButtonF(f func(f *flow.Flow) (flow.Name, flow.State, error)) flow.Button {
	return flow.Button{
		Name:    "Continue",
		Color:   flow.ColorPrimary,
		OnClick: f,
	}
}

func continueButton(next flow.Name) flow.Button {
	return continueButtonF(flow.Goto(next))
}

func (fm *FlowManager) getBaseState() flow.State {
	config := fm.getConfiguration()
	isOAuthConfigured := config.GitlabOAuthClientID != "" || config.GitlabOAuthClientSecret != ""
	return flow.State{
		keyGitlabURL:                   config.GitlabURL,
		keyUsePreregisteredApplication: config.UsePreregisteredApplication,
		keyIsOAuthConfigured:           isOAuthConfigured,
	}
}

func (fm *FlowManager) StartSetupWizard(userID string, delegatedFrom string) error {
	state := fm.getBaseState()
	state[keyDelegatedFrom] = delegatedFrom

	err := fm.setupFlow.ForUser(userID).Start(state)
	if err != nil {
		return err
	}

	fm.trackStartSetupWizard(userID, delegatedFrom != "")

	return nil
}

func (fm *FlowManager) trackStartSetupWizard(userID string, fromInvite bool) {
	fm.tracker.TrackUserEvent("setup_wizard_start", userID, map[string]interface{}{
		"from_invite": fromInvite,
		"time":        model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteSetupWizard(userID string) {
	fm.tracker.TrackUserEvent("setup_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) StartOauthWizard(userID string) error {
	state := fm.getBaseState()

	err := fm.oauthFlow.ForUser(userID).Start(state)
	if err != nil {
		return err
	}

	fm.trackStartOauthWizard(userID)

	return nil
}

func (fm *FlowManager) trackStartOauthWizard(userID string) {
	fm.tracker.TrackUserEvent("oauth_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteOauthWizard(userID string) {
	fm.tracker.TrackUserEvent("oauth_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) stepWelcome() flow.Step {
	welcomePretext := ":wave: Welcome to your GitLab integration! [Learn more](https://mattermost.gitbook.io/plugin-gitlab/)"

	welcomeText := `
{{- if .UsePreregisteredApplication -}}
Just a few configuration steps to go!
- **Step 1:** Connect your GiLab account
- **Step 2:** Create a webhook in GitLab
{{- else -}}
Just a few configuration steps to go!
- **Step 1:** Register an OAuth application in GitLab and enter OAuth values.
- **Step 2:** Connect your GitLab account
- **Step 3:** Create a webhook in GitLab
{{- end -}}`

	return flow.NewStep(stepWelcome).
		WithText(welcomeText).
		WithPretext(welcomePretext).
		WithButton(continueButton(""))
}

func (fm *FlowManager) stepDelegateQuestion() flow.Step {
	delegateQuestionText := "Are you setting this GitLab integration up, or is someone else?"
	return flow.NewStep(stepDelegateQuestion).
		WithText(delegateQuestionText).
		WithButton(flow.Button{
			Name:  "I'll do it myself",
			Color: flow.ColorPrimary,
			OnClick: func(f *flow.Flow) (flow.Name, flow.State, error) {
				if f.GetState().GetBool(keyUsePreregisteredApplication) {
					return stepOAuthConnect, nil, nil
				}

				return stepGitlabURL, nil, nil
			},
		}).
		WithButton(flow.Button{
			Name:  "I need someone else",
			Color: flow.ColorDefault,
			Dialog: &model.Dialog{
				Title:       "Send instructions",
				SubmitLabel: "Send",
				Elements: []model.DialogElement{
					{
						DisplayName: "To",
						Name:        "delegate",
						Type:        "select",
						DataSource:  "users",
						Placeholder: "Search for people",
					},
				},
			},
			OnDialogSubmit: fm.submitDelegateSelection,
		})
}

func (fm *FlowManager) submitDelegateSelection(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	delegateIDRaw, ok := submitted["delegate"]
	if !ok {
		return "", nil, nil, errors.New("delegate missing")
	}
	delegateID, ok := delegateIDRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("delegate is not a string")
	}

	delegate, err := fm.client.User.Get(delegateID)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed get user")
	}

	err = fm.StartSetupWizard(delegate.Id, f.UserID)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed start configuration wizard")
	}

	return stepDelegateConfirmation, flow.State{
		keyDelegatedTo: delegate.Username,
	}, nil, nil
}

func (fm *FlowManager) stepDelegateConfirmation() flow.Step {
	return flow.NewStep(stepDelegateConfirmation).
		WithText("GitLab integration setup details have been sent to @{{ .DelegatedTo }}").
		WithButton(flow.Button{
			Name:     "Waiting for @{{ .DelegatedTo }}...",
			Color:    flow.ColorDefault,
			Disabled: true,
		}).
		WithButton(cancelButton())
}

func (fm *FlowManager) stepDelegateComplete() flow.Step {
	return flow.NewStep(stepDelegateComplete).
		WithText("@{{ .DelegatedTo }} completed configuring the integration.").
		Next(stepDone)
}

func (fm *FlowManager) stepInstanceURL() flow.Step {
	enterpriseText := "Are you using `gitlab.com`?"
	return flow.NewStep(stepGitlabURL).
		WithText(enterpriseText).
		WithButton(flow.Button{
			Name:  "Yes",
			Color: flow.ColorDefault,
			OnClick: func(f *flow.Flow) (flow.Name, flow.State, error) {
				err := fm.setGitlabURL(gitlab.Gitlabdotcom)
				if err != nil {
					return "", nil, err
				}

				return stepOAuthInfo, nil, nil
			},
		}).
		WithButton(flow.Button{
			Name:  "No",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:            "GitLab URL",
				IntroductionText: "Enter the **GitLab URL** of your GitLab instance (Example: https://gitlab.example.com).",
				SubmitLabel:      "Save & continue",
				Elements: []model.DialogElement{
					{

						DisplayName: "GitLab URL",
						Name:        "gitlab_url",
						Type:        "text",
						SubType:     "url",
						Placeholder: "Enter GitLab URL",
					},
				},
			},
			OnDialogSubmit: fm.submitGitlabURL,
		}).
		WithButton(cancelButton())
}

func (fm *FlowManager) submitGitlabURL(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	errorList := map[string]string{}

	gitlabURLRaw, ok := submitted["gitlab_url"]
	if !ok {
		return "", nil, nil, errors.New("gitlab_url missing")
	}
	gitlabURL, ok := gitlabURLRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("gitlab_url is not a string")
	}

	err := isValidURL(gitlabURL)
	if err != nil {
		errorList["gitlab_url"] = err.Error()
	}

	if len(errorList) != 0 {
		return "", nil, errorList, nil
	}

	config := fm.getConfiguration()
	config.GitlabURL = gitlabURL
	config.sanitize()

	configMap, err := config.ToMap()
	if err != nil {
		return "", nil, nil, err
	}

	err = fm.client.Configuration.SavePluginConfig(configMap)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to save plugin config")
	}

	return "", flow.State{
		keyGitlabURL: config.GitlabURL,
	}, nil, nil
}

func (fm *FlowManager) stepOAuthInfo() flow.Step {
	oauthPretext := `
##### :white_check_mark: Step 1: Register an OAuth Application in GitLab
You must first register the Mattermost GitLab Plugin as an authorized OAuth app.`
	oauthMessage := fmt.Sprintf(""+
		"1. In a browser, go to {{ .GitlabURL }}/-/profile/applications.\n"+
		"2. Set the following values:\n"+
		"	- Name: `Mattermost GitLab Plugin - <your company name>`\n"+
		"	- Redirect URI: `%s/oauth/complete`\n"+
		"3. Unselect **Expire access tokens**.\n"+
		"4. Select `api` and `read_user` in Scopes.\n"+
		"5. Select **Save application**\n",
		fm.pluginURL,
	)

	return flow.NewStep(stepOAuthInfo).
		WithPretext(oauthPretext).
		WithText(oauthMessage).
		WithImage("public/new-oauth-application.png").
		WithButton(continueButton("")).
		WithButton(cancelButton())
}

func (fm *FlowManager) stepOAuthInput() flow.Step {
	return flow.NewStep(stepOAuthInput).
		WithText("Click the Continue button below to open a dialog to enter the **Application ID** and **Secret**.").
		WithButton(flow.Button{
			Name:  "Continue",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:            "GitLab OAuth values",
				IntroductionText: "Please enter the **Application ID** and **Secret** you copied in a previous step.{{ if .IsOAuthConfigured }}\n\n**Any existing OAuth configuration will be overwritten.**{{end}}",
				SubmitLabel:      "Save & continue",
				Elements: []model.DialogElement{
					{
						DisplayName: "GitLab OAuth Application ID",
						Name:        "client_id",
						Type:        "text",
						SubType:     "text",
						Placeholder: "Enter GitLab OAuth Application ID",
					},
					{
						DisplayName: "GitLab OAuth Secret",
						Name:        "client_secret",
						Type:        "text",
						SubType:     "text",
						Placeholder: "Enter GitLab OAuth Secret",
					},
				},
			},
			OnDialogSubmit: fm.submitOAuthConfig,
		}).
		WithButton(cancelButton())
}

func (fm *FlowManager) submitOAuthConfig(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	errorList := map[string]string{}

	clientIDRaw, ok := submitted["client_id"]
	if !ok {
		return "", nil, nil, errors.New("client_id missing")
	}
	clientID, ok := clientIDRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("client_id is not a string")
	}

	clientID = strings.TrimSpace(clientID)

	if len(clientID) != 64 {
		errorList["client_id"] = "Client ID should be 64 characters long"
	}

	clientSecretRaw, ok := submitted["client_secret"]
	if !ok {
		return "", nil, nil, errors.New("client_secret missing")
	}
	clientSecret, ok := clientSecretRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("client_secret is not a string")
	}

	clientSecret = strings.TrimSpace(clientSecret)

	if len(clientSecret) != 64 {
		errorList["client_secret"] = "Client Secret should be 64 characters long"
	}

	if len(errorList) != 0 {
		return "", nil, errorList, nil
	}

	config := fm.getConfiguration()
	config.GitlabOAuthClientID = clientID
	config.GitlabOAuthClientSecret = clientSecret

	configMap, err := config.ToMap()
	if err != nil {
		return "", nil, nil, err
	}

	err = fm.client.Configuration.SavePluginConfig(configMap)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to save plugin config")
	}

	return "", nil, nil, nil
}

func (fm *FlowManager) stepOAuthConnect() flow.Step {
	connectPretext := "##### :white_check_mark: Step {{ if .UsePreregisteredApplication }}1{{ else }}2{{ end }}: Connect your GitLab account"
	connectURL := fmt.Sprintf("%s/oauth/connect", fm.pluginURL)
	connectText := fmt.Sprintf("Go [here](%s) to connect your account.", connectURL)
	return flow.NewStep(stepOAuthConnect).
		WithText(connectText).
		WithPretext(connectPretext).
		OnRender(func(f *flow.Flow) { fm.trackCompleteOauthWizard(f.UserID) })
	// The API handler will advance to the next step and complete the flow
}

func (fm *FlowManager) StartWebhookWizard(userID string) error {
	state := fm.getBaseState()

	err := fm.webhokFlow.ForUser(userID).Start(state)
	if err != nil {
		return err
	}

	fm.trackStartWebhookWizard(userID)

	return nil
}

func (fm *FlowManager) trackStartWebhookWizard(userID string) {
	fm.tracker.TrackUserEvent("webhook_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteWebhookWizard(userID string) {
	fm.tracker.TrackUserEvent("webhook_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) stepWebhookQuestion() flow.Step {
	questionPretext := `##### :white_check_mark: Step {{ if .UsePreregisteredApplication }}2{{ else }}3{{ end }}: Create a Webhook in GitLab
The final setup step requires a Mattermost System Admin to create a webhook for each GitLab group or project to receive notifications for, or want to subscribe to.`
	return flow.NewStep(stepWebhookQuestion).
		WithText("Do you want to create a webhook?").
		WithPretext(questionPretext).
		WithButton(flow.Button{
			Name:  "Yes",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:       "Create webhook",
				SubmitLabel: "Create",
				Elements: []model.DialogElement{
					{

						DisplayName: "Gitlab project or group name",
						Name:        "namespace",
						Type:        "text",
						SubType:     "text",
						Placeholder: "Enter GitLab project or group name",
						HelpText:    "Specify the GitLab project or group to connect to Mattermost. For example, mattermost/mattermost-server.",
					},
				},
			},
			OnDialogSubmit: fm.submitWebhook,
		}).
		WithButton(flow.Button{
			Name:    "No",
			Color:   flow.ColorDefault,
			OnClick: flow.Goto(stepWebhookWarning),
		})
}

func (fm *FlowManager) submitWebhook(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	namespaceRaw, ok := submitted["namespace"]
	if !ok {
		return "", nil, nil, errors.New("namespace missing")
	}
	namespace, ok := namespaceRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("namespace is not a string")
	}

	config := fm.getConfiguration()

	info, apiErr := fm.getGitlabUserInfoByMattermostID(f.UserID)
	if apiErr != nil {
		return "", nil, nil, apiErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), 28*time.Second) // HTTP request times out after 30 seconds
	defer cancel()

	gitlabClient := fm.getGitlabClient()

	group, project, err := gitlabClient.ResolveNamespaceAndProject(ctx, info, namespace, config.EnablePrivateRepo)
	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) {
			return "", nil, nil, errors.New("project or group was not found")
		}

		return "", nil, nil, gitlab.PrettyError(err)
	}

	hookOptions := &gitlab.AddWebhookOptions{
		URL:                      fmt.Sprintf("%s/webhook", fm.pluginURL),
		ConfidentialNoteEvents:   true,
		PushEvents:               true,
		IssuesEvents:             true,
		ConfidentialIssuesEvents: true,
		MergeRequestsEvents:      true,
		TagPushEvents:            true,
		NoteEvents:               true,
		JobEvents:                true,
		PipelineEvents:           true,
		WikiPageEvents:           true,
		EnableSSLVerification:    true,
		Token:                    config.WebhookSecret,
	}

	var fullName string
	var repoOrGroup string

	if group == "" {
		fullName = group
		repoOrGroup = "group"
	} else {
		fullName = group + "/" + project
		repoOrGroup = "repository"
	}

	_, err = CreateHook(ctx, gitlabClient, info, group, project, hookOptions)
	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) {
			return "", nil, nil, errors.New("project or group was not found")
		}
		if errors.Is(err, gitlab.ErrForbidden) {
			err = errors.Errorf("It seems like you don't have privileges to create webhooks in %s. Ask an admin of that %s to run /gitlab setup webhook for you.", fullName, repoOrGroup)
			return "", nil, nil, err
		}

		return "", nil, nil, errors.Wrap(gitlab.PrettyError(err), "failed to create hook")
	}

	return stepWebhookConfirmation, nil, nil, nil
}

func (fm *FlowManager) stepWebhookWarning() flow.Step {
	warnText := "The GitLab plugin uses a webhook to connect a GitLab account to Mattermost to listen for incoming GitLab events. " +
		"You can't subscribe a channel to a repository for notifications until webhooks are configured.\n" +
		"Restart setup later by running `/gitab setup webhook`"

	return flow.NewStep(stepWebhookWarning).
		WithText(warnText).
		WithColor(flow.ColorDanger).
		Next("")
}

func (fm *FlowManager) stepWebhookConfirmation() flow.Step {
	return flow.NewStep(stepWebhookConfirmation).
		WithTitle("Success! :tada: You've successfully set up your Mattermost GitLab integration! ").
		WithText("Use `/gitlab subscriptions add` to subscribe any Mattermost channel to your GitLab repository. [Learn more](https://mattermost.gitbook.io/plugin-gitlab/feature-summary#subscribe-to-unsubscribe-from-a-repository)").
		OnRender(func(f *flow.Flow) { fm.trackCompleteWebhookWizard(f.UserID) }).
		Next("")
}

func (fm *FlowManager) StartAnnouncementWizard(userID string) error {
	state := fm.getBaseState()

	err := fm.announcementFlow.ForUser(userID).Start(state)
	if err != nil {
		return err
	}

	fm.trackStartAnnouncementWizard(userID)

	return nil
}

func (fm *FlowManager) trackStartAnnouncementWizard(userID string) {
	fm.tracker.TrackUserEvent("announcement_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompletAnnouncementWizard(userID string) {
	fm.tracker.TrackUserEvent("announcement_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) stepAnnouncementQuestion() flow.Step {
	defaultMessage := "Hi team,\n" +
		"\n" +
		"We've set up the Mattermost Gitlab plugin to enable notifications from Gitlab in Mattermost. To get started, run the `/gitlab connect` slash command from any channel within Mattermost to connect that channel with GitLab. See the [documentation](https://mattermost.gitbook.io/plugin-gitlab/) for details on using the GitLab plugin."

	return flow.NewStep(stepAnnouncementQuestion).
		WithText("Want to let your team know?").
		WithButton(flow.Button{
			Name:  "Send Message",
			Color: flow.ColorPrimary,
			Dialog: &model.Dialog{
				Title:       "Notify your team",
				SubmitLabel: "Send message",
				Elements: []model.DialogElement{
					{
						DisplayName: "To",
						Name:        "channel_id",
						Type:        "select",
						Placeholder: "Select channel",
						DataSource:  "channels",
					},
					{
						DisplayName: "Message",
						Name:        "message",
						Type:        "textarea",
						Default:     defaultMessage,
						HelpText:    "You can edit this message before sending it.",
					},
				},
			},
			OnDialogSubmit: fm.submitChannelAnnouncement,
		}).
		WithButton(flow.Button{
			Name:    "Not now",
			Color:   flow.ColorDefault,
			OnClick: flow.Goto(stepDone),
		})
}

func (fm *FlowManager) stepAnnouncementConfirmation() flow.Step {
	return flow.NewStep(stepAnnouncementConfirmation).
		WithText("Message to ~{{ .ChannelName }} was sent.").
		Next("").
		OnRender(func(f *flow.Flow) { fm.trackCompletAnnouncementWizard(f.UserID) })
}

func (fm *FlowManager) submitChannelAnnouncement(f *flow.Flow, submitted map[string]interface{}) (flow.Name, flow.State, map[string]string, error) {
	channelIDRaw, ok := submitted["channel_id"]
	if !ok {
		return "", nil, nil, errors.New("channel_id missing")
	}
	channelID, ok := channelIDRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("channel_id is not a string")
	}

	channel, err := fm.client.Channel.Get(channelID)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to get channel")
	}

	messageRaw, ok := submitted["message"]
	if !ok {
		return "", nil, nil, errors.New("message is not a string")
	}
	message, ok := messageRaw.(string)
	if !ok {
		return "", nil, nil, errors.New("message is not a string")
	}

	post := &model.Post{
		UserId:    f.UserID,
		ChannelId: channel.Id,
		Message:   message,
	}
	err = fm.client.Post.CreatePost(post)
	if err != nil {
		return "", nil, nil, errors.Wrap(err, "failed to create announcement post")
	}

	return stepAnnouncementConfirmation, flow.State{
		"ChannelName": channel.Name,
	}, nil, nil
}

func (fm *FlowManager) setGitlabURL(gitlabURL string) error {
	config := fm.getConfiguration()
	config.GitlabURL = gitlabURL

	configMap, err := config.ToMap()
	if err != nil {
		return err
	}

	err = fm.client.Configuration.SavePluginConfig(configMap)
	if err != nil {
		return errors.Wrap(err, "failed to save plugin config")
	}

	return nil
}
