package main

import (
	"fmt"

	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/experimental/flow"
	"github.com/mattermost/mattermost-plugin-api/experimental/telemetry"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

type FlowManager struct {
	client                          *pluginapi.Client
	pluginURL                       string
	botUserID                       string
	router                          *mux.Router
	getConfiguration                func() *configuration
	getGitlabUserInfoByMattermostID func(userID string) (*gitlab.UserInfo, *APIErrorResponse)

	tracker telemetry.Tracker

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

		tracker: p.tracker,
	}

	fm.setupFlow = fm.newFlow("setup").WithSteps(
		/*
			fm.stepWelcome(),
			fm.stepDelegateQuestion(),
			fm.stepDelegateConfirmation(),
			fm.stepDelegateComplete(),

			fm.stepEnterprise(),
			fm.stepOAuthInfo(),
			fm.stepOAuthInput(),
			fm.stepOAuthConnect(),

			fm.stepWebhookQuestion(),
			fm.stepWebhookWarning(),
			fm.stepConfirmationStep(),
		*/

		fm.stepAnnouncementQuestion(),
		fm.stepAnnouncementConfirmation(),

		fm.doneStep(),

		fm.stepCancel("setup"),
	)

	fm.oauthFlow = fm.newFlow("oauth").WithSteps(
	/*
		fm.stepEnterprise(),
		fm.stepOAuthInfo(),
		fm.stepOAuthInfo(),
		fm.stepOAuthInput(),
		fm.stepOAuthConnect().Terminal(),

		fm.stepCancel("setup oauth"),
	*/
	)
	fm.webhokFlow = fm.newFlow("webhook").WithSteps(
	/*
		fm.stepWebhookQuestion(),
		flow.NewStep(stepWebhookConfirmation).
			WithText("Use `/github subscriptions add` to subscribe any Mattermost channel to your GitHub repository. [Learn more](https://example.org)").
			Terminal(),

		fm.stepCancel("setup webhook"),
	*/
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
		WithText(":tada: You successfully installed GitHub.").
		OnRender(fm.onDone).Terminal()
}

func (fm *FlowManager) onDone(f *flow.Flow) {
	fm.trackCompleteSetupWizard(f.UserID)

	delegatedFrom := f.State.GetString(keyDelegatedFrom)
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

	stepEnterprise   flow.Name = "enterprise"
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
	keyGitlabURL                   = "GitlabURL"
	keyUsePreregisteredApplication = "UsePreregisteredApplication"
	keyIsOAuthConfigured           = "IsOAuthConfigured"
)

func cancelButton() flow.Button {
	return flow.Button{
		Name:    "Cancel",
		Color:   flow.ColorDanger,
		OnClick: flow.Goto(stepCancel),
	}
}

func (fm *FlowManager) stepCancel(command string) flow.Step {
	return flow.NewStep(stepCancel).
		Terminal().
		WithText(fmt.Sprintf("Gitlab integration setup has stopped. Restart setup later by running `/gitlab %s`. Learn more about the plugin [here](%s).", command, manifest.HomepageURL)).
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
	_ = fm.tracker.TrackUserEvent("setup_wizard_start", userID, map[string]interface{}{
		"from_invite": fromInvite,
		"time":        model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteSetupWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("setup_wizard_complete", userID, map[string]interface{}{
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
	_ = fm.tracker.TrackUserEvent("oauth_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteOauthWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("oauth_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
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
	_ = fm.tracker.TrackUserEvent("webhook_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompleteWebhookWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("webhook_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
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
	_ = fm.tracker.TrackUserEvent("announcement_wizard_start", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) trackCompletAnnouncementWizard(userID string) {
	_ = fm.tracker.TrackUserEvent("announcement_wizard_complete", userID, map[string]interface{}{
		"time": model.GetMillis(),
	})
}

func (fm *FlowManager) stepAnnouncementQuestion() flow.Step {
	defaultMessage := "Hi team,\n" +
		"\n" +
		"We've set up the Mattermost Gitlab plugin to enable notifications from Gitlab in Mattermost. To get started, run the `/gitlab connect` slash command from any channel within Mattermost to connect that channel with GitHub. See the [documentation](https://mattermost.gitbook.io/plugin-gitlab/) for details on using the GitHub plugin."

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
