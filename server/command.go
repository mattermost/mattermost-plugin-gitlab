package main

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
	gitlabLib "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

const commandHelp = `* |/gitlab connect| - Connect your Mattermost account to your GitLab account
* |/gitlab disconnect| - Disconnect your Mattermost account from your GitLab account
* |/gitlab todo| - Get a list of todos, assigned issues, assigned merge requests and merge requests awaiting your review
* |/gitlab subscriptions list| - Will list the current channel subscriptions
* |/gitlab subscriptions add owner[/repo] [features]| - Subscribe the current channel to receive notifications about opened merge requests and issues for a group or repository
  * |features| is a comma-delimited list of one or more the following:
    * issues - includes new and closed issues
	* jobs - includes jobs status updates
	* merges - includes new and closed merge requests
    * pushes - includes pushes
	* issue_comments - includes new issue comments
	* merge_request_comments - include new merge-request comments
	* pipeline - includes pipeline runs
	* tag - include tag creation
    * pull_reviews - includes merge request reviews
	* label:"<labelname>" - must include "merges" or "issues" in feature list when using a label
    * Defaults to "merges,issues,tag"
* |/gitlab subscriptions delete owner/repo| - Unsubscribe the current channel from a repository
* |/gitlab pipelines run [owner]/repo [ref]| - Run a pipeline for specific repository and ref (branch/tag)
* |/gitlab me| - Display the connected GitLab account
* |/gitlab settings [setting] [value]| - Update your user settings
  * |setting| can be "notifications" or "reminders"
  * |value| can be "on" or "off"
* |/gitlab webhook list [owner]/repo| - Will list associated group or project hooks.
* |/gitlab webhook add owner[/repo] [options] [url] [token]|
  * |options| is a comma-delimited list of one or more the following:
	 * |*| - or missing defaults to all with SSL verification enabled
	 * *noSSL - all triggers with SSL verification not enabled.
	 * PushEvents
	 * TagPushEvents 
	 * Comments 
	 * ConfidentialComments 
	 * IssuesEvents
	 * ConfidentialIssuesEvents 
	 * MergeRequestsEvents 
	 * JobEvents 
	 * PipelineEvents 
	 * WikiPageEvents
	 * SSLverification
  * |url| is the URL that will be called when triggered. Defaults to this plugins URL
  * |token| Secret token. Defaults to secret token used in plugin's settings.
* |/gitlab about| - Display build information about the plugin
`
const (
	webhookHowToURL                   = "https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook"
	inboundWebhookURL                 = "plugins/com.github.manland.mattermost-plugin-gitlab/webhook"
	specifyRepositoryMessage          = "Please specify a repository."
	specifyRepositoryAndBranchMessage = "Please specify a repository and a branch."
	unknownActionMessage              = "Unknown action, please use `/gitlab help` to see all actions available."
	newWebhookEmptySiteURLmessage     = "Unable to create webhook. The Mattermot Site URL is not set. " +
		"Set it in the Admin Console or rerun /gitlab webhook add group/project URL including the desired URL."
)

const (
	groupNotFoundError   = "404 {message: 404 Group Not Found}"
	groupNotFoundMessage = "Unable to find GitLab group: "

	projectNotFoundError   = "404 {message: 404 Project Not Found}"
	projectNotFoundMessage = "Unable to find project with namespace: "

	invalidSubscribeSubCommand = "Invalid subscribe command. Available commands are add, delete, and list"

	invalidPipelinesSubCommand = "Invalid pipelines command. Available commands are run, list"
)

const (
	commandAdd    = "add"
	commandDelete = "delete"
	commandList   = "list"

	commandRun = "run"
)

const (
	commandTimeout = 30 * time.Second
)

func (p *Plugin) getCommand(config *configuration) (*model.Command, error) {
	iconData, err := command.GetIconData(&p.client.System, "assets/icon.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	return &model.Command{
		Trigger:              "gitlab",
		AutoComplete:         true,
		AutoCompleteDesc:     "Available commands: connect, disconnect, todo, subscriptions, me, pipelines, settings, webhook, setup, help, about",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(config),
		AutocompleteIconData: iconData,
	}, nil
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: args.ChannelId,
		RootId:    args.RootId,
		Message:   text,
	}
	p.client.Post.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) getCommandResponse(args *model.CommandArgs, text string) *model.CommandResponse {
	p.postCommandResponse(args, text)
	return &model.CommandResponse{}
}

// ExecuteCommand is the entrypoint for /gitlab commands. It returns a message to display to the user or an error.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (res *model.CommandResponse, appErr *model.AppError) {
	var (
		split      = strings.Fields(args.Command)
		cmd        = split[0]
		action     string
		parameters []string
	)
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}
	if cmd != "/gitlab" {
		return &model.CommandResponse{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			p.client.Log.Warn("Recovered from a panic",
				"Command", args.Command,
				"UserId", args.UserId,
				"error", r,
				"stack", string(debug.Stack()))
			res = &model.CommandResponse{}
			p.postCommandResponse(args, "An unexpected error occurred. Please try again later.")
			if *p.client.Configuration.GetConfig().ServiceSettings.EnableDeveloper {
				p.postCommandResponse(args, fmt.Sprintf("error: %v, \nstack:\n```\n%s\n```", r, string(debug.Stack())))
			}
		}
	}()

	if action == "about" {
		text, err := command.BuildInfo(manifest)
		if err != nil {
			text = errors.Wrap(err, "failed to get build info").Error()
		}
		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	if action == "setup" {
		message := p.handleSetup(c, args, parameters)
		if message != "" {
			p.postCommandResponse(args, message)
		}
		return &model.CommandResponse{}, nil
	}

	config := p.getConfiguration()

	if err := config.IsValid(); err != nil {
		isSysAdmin, err := p.isAuthorizedSysAdmin(args.UserId)
		var text string
		switch {
		case err != nil:
			text = "Error checking user's permissions"
			p.client.Log.Warn(text, "error", err.Error())
		case isSysAdmin:
			text = "Before using this plugin, you'll need to configure it by running `/gitlab setup`"
		default:
			text = "Please contact your system administrator to configure the GitLab plugin."
		}

		p.postCommandResponse(args, text)
		return &model.CommandResponse{}, nil
	}

	if action == "connect" {
		config := p.client.Configuration.GetConfig()
		if config.ServiceSettings.SiteURL == nil {
			return p.getCommandResponse(args, "Encountered an error connecting to GitLab."), nil
		}

		resp := p.getCommandResponse(args, fmt.Sprintf("[Click here to link your GitLab account.](%s/plugins/%s/oauth/connect)", *config.ServiceSettings.SiteURL, manifest.Id))
		return resp, nil
	}

	if action == "help" || action == "" {
		text := "###### Mattermost GitLab Plugin - Slash Command Help\n" + strings.ReplaceAll(commandHelp, "|", "`")
		return p.getCommandResponse(args, text), nil
	}

	info, apiErr := p.getGitlabUserInfoByMattermostID(args.UserId)
	if apiErr != nil {
		text := "Unknown error."
		if apiErr.ID == APIErrorIDNotConnected {
			text = "You must connect your account to GitLab first. Either click on the GitLab logo in the bottom left of the screen or enter `/gitlab connect`."
		}
		return p.getCommandResponse(args, text), nil
	}

	switch action {
	case "subscriptions", "subscription", "subscribe":
		message := p.subscribeCommand(ctx, parameters, args.ChannelId, config, info)
		response := p.getCommandResponse(args, message)
		return response, nil
	case "unsubscribe":
		// subcommand subscriptions delete is preferred but unsubscribe remains to prevent breaking existing workflows
		var message string
		var err error

		if len(parameters) == 0 {
			message = specifyRepositoryMessage
		} else {
			message, err = p.subscriptionDelete(info, config, parameters[0], args.ChannelId)
			if err != nil {
				message = err.Error()
			}
		}
		response := p.getCommandResponse(args, message)
		return response, nil
	case "disconnect":
		p.disconnectGitlabAccount(args.UserId)
		return p.getCommandResponse(args, "Disconnected your GitLab account."), nil
	case "todo":
		_, text, err := p.GetToDo(ctx, info)
		if err != nil {
			p.client.Log.Warn("can't get todo in command", "err", err.Error())
			return p.getCommandResponse(args, "Encountered an error getting your todo items."), nil
		}
		return p.getCommandResponse(args, text), nil
	case "me":
		var gitUser *gitlabLib.User
		err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			resp, err := p.GitlabClient.GetUserDetails(ctx, info, token)
			if err != nil {
				return err
			}
			gitUser = resp
			return nil
		})

		if err != nil {
			return p.getCommandResponse(args, "Encountered an error getting your GitLab profile."), nil
		}

		text := fmt.Sprintf("You are connected to GitLab as:\n# [![image](%s =40x40)](%s) [%s](%s)", gitUser.AvatarURL, gitUser.WebURL, gitUser.Username, gitUser.WebsiteURL)
		return p.getCommandResponse(args, text), nil
	case "settings":
		if len(parameters) < 2 {
			return p.getCommandResponse(args, "Please specify both a setting and value. Use `/gitlab help` for more usage information."), nil
		}

		setting := parameters[0]
		strValue := parameters[1]
		value := false
		if strValue == SettingOn {
			value = true
		} else if strValue != SettingOff {
			return p.getCommandResponse(args, "Invalid value. Accepted values are: \"on\" or \"off\"."), nil
		}

		switch setting {
		case SettingNotifications:
			if value {
				if err := p.storeGitlabToUserIDMapping(info.GitlabUsername, info.UserID); err != nil {
					p.client.Log.Warn("can't store GitLab to user id mapping", "err", err.Error())
					return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
				}
				if err := p.storeGitlabIDToUserIDMapping(info.GitlabUsername, info.GitlabUserID); err != nil {
					p.client.Log.Warn("can't store GitLab to GitLab id mapping", "err", err.Error())
					return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
				}
			} else if err := p.deleteGitlabToUserIDMapping(info.GitlabUsername); err != nil {
				p.client.Log.Warn("can't delete GitLab username in kvstore", "err", err.Error())
				return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
			}
			info.Settings.Notifications = value
		case SettingReminders:
			info.Settings.DailyReminder = value
		default:
			return p.getCommandResponse(args, "Unknown setting."), nil
		}

		if err := p.storeGitlabUserInfo(info); err != nil {
			p.client.Log.Warn("can't store user info after update by command", "err", err.Error())
			return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
		}

		return p.getCommandResponse(args, "Settings updated."), nil

	case "webhook":
		message := p.webhookCommand(ctx, parameters, info, config.EnablePrivateRepo)
		response := p.getCommandResponse(args, message)
		return response, nil
	case "pipelines":
		message := p.pipelinesCommand(ctx, parameters, args.ChannelId, info)
		response := p.getCommandResponse(args, message)
		return response, nil
	default:
		return p.getCommandResponse(args, unknownActionMessage), nil
	}
}

func (p *Plugin) handleSetup(c *plugin.Context, args *model.CommandArgs, parameters []string) string {
	userID := args.UserId
	isSysAdmin, err := p.isAuthorizedSysAdmin(userID)
	if err != nil {
		p.client.Log.Warn("Failed to check if user is System Admin", "error", err.Error())

		return "Error checking user's permissions"
	}

	if !isSysAdmin {
		return "Only System Admins are allowed to set up the plugin."
	}

	if len(parameters) == 0 {
		err = p.flowManager.StartSetupWizard(userID, "")
	} else {
		command := parameters[0]

		switch {
		case command == "oauth":
			err = p.flowManager.StartOauthWizard(userID)
		case command == "webhook":
			err = p.flowManager.StartWebhookWizard(userID)
		case command == "announcement":
			err = p.flowManager.StartAnnouncementWizard(userID)
		default:
			return fmt.Sprintf("Unknown subcommand %v", command)
		}
	}

	if err != nil {
		return err.Error()
	}

	return ""
}

// webhookCommand processes the /gitlab webhook commands
func (p *Plugin) webhookCommand(ctx context.Context, parameters []string, info *gitlab.UserInfo, enablePrivateRepo bool) string {
	if len(parameters) < 1 {
		return unknownActionMessage
	}
	subCommand := parameters[0]

	switch subCommand {
	case commandList:
		if len(parameters) != 2 {
			return unknownActionMessage
		}

		namespace := parameters[1]
		var group, project string
		err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			respGroup, respProject, err := p.GitlabClient.ResolveNamespaceAndProject(ctx, info, token, namespace, enablePrivateRepo)
			if err != nil {
				return err
			}
			group = respGroup
			project = respProject
			return nil
		})

		if err != nil {
			return err.Error()
		}

		var webhookInfo []*gitlab.WebhookInfo
		if project != "" {
			err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
				resp, err := p.GitlabClient.GetProjectHooks(ctx, info, token, group, project)
				if err != nil {
					return err
				}
				webhookInfo = resp
				return nil
			})
			if err != nil {
				if strings.Contains(err.Error(), projectNotFoundError) {
					return projectNotFoundMessage + namespace
				}
				return err.Error()
			}
		} else {
			err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
				resp, err := p.GitlabClient.GetGroupHooks(ctx, info, token, group)
				if err != nil {
					return err
				}
				webhookInfo = resp
				return nil
			})
			if err != nil {
				if strings.Contains(err.Error(), groupNotFoundError) {
					return groupNotFoundMessage + group
				}
				return err.Error()
			}
		}
		if len(webhookInfo) == 0 {
			return fmt.Sprintf("No webhooks found in %s", namespace)
		}
		var formatedWebhooks string
		for _, hook := range webhookInfo {
			formatedWebhooks += hook.String()
		}
		return formatedWebhooks

	case commandAdd:
		if len(parameters) < 2 {
			return unknownActionMessage
		}

		siteURL := *p.client.Configuration.GetConfig().ServiceSettings.SiteURL
		if siteURL == "" {
			return newWebhookEmptySiteURLmessage
		}

		urlPath := fmt.Sprintf("%v/%s", siteURL, inboundWebhookURL)
		if len(parameters) > 3 {
			urlPath = parameters[3]
		}

		// default to all triggers unless specified
		hookOptions := parseTriggers("*")
		if len(parameters) > 2 {
			triggersCsv := parameters[2]
			hookOptions = parseTriggers(triggersCsv)
		}
		hookOptions.URL = urlPath

		if len(parameters) > 4 {
			hookOptions.Token = parameters[4]
		} else {
			hookOptions.Token = p.getConfiguration().WebhookSecret
		}

		namespace := parameters[1]
		var group, project string
		namespaceErr := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			respGroup, respProject, err := p.GitlabClient.ResolveNamespaceAndProject(ctx, info, token, namespace, enablePrivateRepo)
			if err != nil {
				return err
			}
			group = respGroup
			project = respProject
			return nil
		})
		if namespaceErr != nil {
			return namespaceErr.Error()
		}

		newWebhook, err := p.createHook(ctx, p.GitlabClient, info, group, project, hookOptions)
		if err != nil {
			return err.Error()
		}
		return fmt.Sprintf("Webhook Created:\n%s", newWebhook.String())

	default:
		return fmt.Sprintf("Unknown webhook command: %s", subCommand)
	}
}

func parseTriggers(triggersCsv string) *gitlab.AddWebhookOptions {
	var sslVerification, pushEvents, tagPushEvents, issuesEvents, confidentialIssuesEvents, noteEvents bool
	var confidentialNoteEvents, mergeRequestsEvents, jobEvents, pipelineEvents, wikiPageEvents bool
	var all bool
	if triggersCsv == "*" {
		all = true
		sslVerification = true
	}
	if strings.EqualFold(triggersCsv, "*noSSL") {
		all = true
		sslVerification = false
	}
	triggers := strings.Split(triggersCsv, ",")
	for _, trigger := range triggers {
		if strings.EqualFold(trigger, "SSLverification") {
			sslVerification = true
		}
		if all || strings.EqualFold(trigger, "PushEvents") {
			pushEvents = true
		}
		if all || strings.EqualFold(trigger, "TagPushEvents") {
			tagPushEvents = true
		}
		if all || strings.EqualFold(trigger, "IssuesEvents") {
			issuesEvents = true
		}
		if all || strings.EqualFold(trigger, "ConfidentialIssuesEvents") {
			confidentialIssuesEvents = true
		}
		if all || strings.EqualFold(trigger, "Comments") {
			noteEvents = true
		}
		if all || strings.EqualFold(trigger, "ConfidentialComments") {
			confidentialNoteEvents = true
		}
		if all || strings.EqualFold(trigger, "MergeRequestsEvents") {
			mergeRequestsEvents = true
		}
		if all || strings.EqualFold(trigger, "JobEvents") {
			jobEvents = true
		}
		if all || strings.EqualFold(trigger, "PipelineEvents") {
			pipelineEvents = true
		}
		if all || strings.EqualFold(trigger, "WikiPageEvents") {
			wikiPageEvents = true
		}
	}

	return &gitlab.AddWebhookOptions{
		EnableSSLVerification:    sslVerification,
		ConfidentialNoteEvents:   confidentialNoteEvents,
		PushEvents:               pushEvents,
		IssuesEvents:             issuesEvents,
		ConfidentialIssuesEvents: confidentialIssuesEvents,
		MergeRequestsEvents:      mergeRequestsEvents,
		TagPushEvents:            tagPushEvents,
		NoteEvents:               noteEvents,
		JobEvents:                jobEvents,
		PipelineEvents:           pipelineEvents,
		WikiPageEvents:           wikiPageEvents,
	}
}

func (p *Plugin) subscriptionDelete(info *gitlab.UserInfo, config *configuration, fullPath, channelID string) (string, error) {
	normalizedPath := normalizePath(fullPath, config.GitlabURL)
	deleted, updatedSubscriptions, err := p.Unsubscribe(channelID, normalizedPath)
	if err != nil {
		p.client.Log.Warn("can't unsubscribe channel in command", "err", err.Error())
		return "Encountered an error trying to unsubscribe. Please try again.", nil
	}

	if !deleted {
		return "Subscription not found, please check repository name.", nil
	}

	p.sendChannelSubscriptionsUpdated(updatedSubscriptions, channelID)

	return fmt.Sprintf("Successfully deleted subscription for %s.", normalizedPath), nil
}

// subscriptionsListCommand list GitLab subscriptions in a channel
func (p *Plugin) subscriptionsListCommand(channelID string) string {
	var txt string
	subs, err := p.GetSubscriptionsByChannel(channelID)
	if err != nil {
		return err.Error()
	}
	if len(subs) == 0 {
		txt = "Currently there are no subscriptions in this channel"
	} else {
		txt = "### Subscriptions in this channel\n"
	}
	for _, sub := range subs {
		txt += fmt.Sprintf("* `%s` - %s\n", strings.Trim(sub.Repository, "/"), sub.Features)
	}
	return txt
}

// subscriptionsAddCommand subscripes to A GitLab Project
func (p *Plugin) subscriptionsAddCommand(ctx context.Context, info *gitlab.UserInfo, config *configuration, fullPath, channelID, features string) string {
	var namespace, project string
	err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		respGroup, respProject, err := p.GitlabClient.ResolveNamespaceAndProject(ctx, info, token, fullPath, config.EnablePrivateRepo)
		if err != nil {
			return err
		}
		namespace = respGroup
		project = respProject
		return nil
	})
	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) {
			return "Resource with such path is not found."
		} else if errors.Is(err, gitlab.ErrPrivateResource) {
			return "Requested resource is private."
		}
		p.client.Log.Warn(
			"unable to resolve subscription namespace and project name",
			"err", err.Error(),
		)
		return err.Error()
	}

	updatedSubscriptions, subscribeErr := p.Subscribe(info, namespace, project, channelID, features)
	if subscribeErr != nil {
		p.client.Log.Warn(
			"failed to subscribe",
			"namespace", namespace,
			"project", project,
			"err", subscribeErr.Error(),
		)
		return subscribeErr.Error()
	}
	var hasHook bool
	if project != "" {
		hasHook, err = p.HasProjectHook(ctx, info, namespace, project)
		if err != nil {
			return fmt.Sprintf(
				"Unable to determine status of Webhook. See [setup instructions](%s) to validate.",
				webhookHowToURL,
			)
		}
	} else {
		hasHook, err = p.HasGroupHook(ctx, info, namespace)
		if err != nil {
			return fmt.Sprintf(
				"Unable to determine status of Webhook. See [setup instructions](%s) to validate.",
				webhookHowToURL,
			)
		}
	}
	var hookStatusMessage string
	if !hasHook {
		// no web hook found
		hookStatusMessage = fmt.Sprintf(
			"\nA Webhook is needed, run ```/gitlab webhook add %s``` to create one now.",
			fullPath,
		)
	}

	p.sendChannelSubscriptionsUpdated(updatedSubscriptions, channelID)

	return fmt.Sprintf("Successfully subscribed to %s.%s", fullPath, hookStatusMessage)
}

// subscribeCommand process the /gitlab subscribe command.
// It returns a message and handles all errors my including helpful information in the message
func (p *Plugin) subscribeCommand(ctx context.Context, parameters []string, channelID string, config *configuration, info *gitlab.UserInfo) string {
	if len(parameters) == 0 {
		return invalidSubscribeSubCommand
	}

	subcommand := parameters[0]

	switch subcommand {
	case commandList:
		return p.subscriptionsListCommand(channelID)
	case commandAdd:
		features := "merges,issues,tag"
		if len(parameters) > 2 {
			features = strings.Join(parameters[2:], " ")
		}
		// Resolve namespace and project name
		fullPath := normalizePath(parameters[1], config.GitlabURL)

		return p.subscriptionsAddCommand(ctx, info, config, fullPath, channelID, features)
	case commandDelete:
		if len(parameters) < 2 {
			return specifyRepositoryMessage
		}

		message, err := p.subscriptionDelete(info, config, parameters[1], channelID)
		if err != nil {
			return err.Error()
		}
		return message
	default:
		return invalidSubscribeSubCommand
	}
}
func (p *Plugin) pipelinesCommand(ctx context.Context, parameters []string, channelID string, info *gitlab.UserInfo) string {
	if len(parameters) == 0 {
		return invalidPipelinesSubCommand
	}
	subcommand := parameters[0]
	switch subcommand {
	case commandRun:
		if len(parameters) < 3 {
			return specifyRepositoryAndBranchMessage
		}
		namespace := parameters[1]
		ref := parameters[2]
		return p.pipelineRunCommand(ctx, namespace, ref, channelID, info)
	default:
		return unknownActionMessage
	}
}

// pipelineRunCommand run a pipeline in a project
func (p *Plugin) pipelineRunCommand(ctx context.Context, namespace, ref, channelID string, info *gitlab.UserInfo) string {
	var pipelineInfo *gitlab.PipelineInfo
	err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		groupName, projectName, err := p.GitlabClient.ResolveNamespaceAndProject(ctx, info, token, namespace, true)
		if err != nil {
			return err
		}
		project, err := p.GitlabClient.GetProject(ctx, info, token, groupName, projectName)
		if err != nil {
			return err
		}
		projectID := fmt.Sprintf("%d", project.ID)
		pipelineInfo, err = p.GitlabClient.TriggerProjectPipeline(info, token, projectID, ref)
		if err != nil {
			return errors.Wrapf(err, "failed to run pipeline for Project: :%s", projectName)
		}
		return nil
	})
	if err != nil {
		return err.Error()
	}
	var txt string
	if pipelineInfo == nil {
		txt = "Currently there is no pipeline info"
		return txt
	}
	txt = "### Pipeline info\n"
	txt += fmt.Sprintf("**Status**: %s\n", pipelineInfo.Status)
	txt += fmt.Sprintf("**SHA**: %s\n", pipelineInfo.SHA)
	txt += fmt.Sprintf("**Ref**: %s\n", pipelineInfo.Ref)
	txt += fmt.Sprintf("**Triggered By**: %s\n", pipelineInfo.User)
	txt += fmt.Sprintf("**Visit pipeline [here](%s)** \n\n", pipelineInfo.WebURL)
	return txt
}

func (p *Plugin) isAuthorizedSysAdmin(userID string) (bool, error) {
	user, err := p.client.User.Get(userID)
	if err != nil {
		return false, err
	}
	if !strings.Contains(user.Roles, "system_admin") {
		return false, nil
	}
	return true, nil
}

func getAutocompleteData(config *configuration) *model.AutocompleteData {
	if !config.IsOAuthConfigured() {
		gitlab := model.NewAutocompleteData("gitlab", "[command]", "Available commands: setup, about")

		setup := model.NewAutocompleteData("setup", "", "Set up the GitLab plugin")
		gitlab.AddCommand(setup)

		about := command.BuildInfoAutocomplete("about")
		gitlab.AddCommand(about)

		return gitlab
	}

	gitlab := model.NewAutocompleteData("gitlab", "[command]", "Available commands: connect, disconnect, todo, subscriptions, me, pipelines, settings, webhook, setup, help, about")

	connect := model.NewAutocompleteData("connect", "", "Connect your GitLab account")
	gitlab.AddCommand(connect)

	disconnect := model.NewAutocompleteData("disconnect", "", "disconnect your GitLab account")
	gitlab.AddCommand(disconnect)

	todo := model.NewAutocompleteData("todo", "", "Get a list of todos, assigned issues, assigned merge requests and merge requests awaiting your review")
	gitlab.AddCommand(todo)

	subscriptions := model.NewAutocompleteData("subscriptions", "[command]", "Available commands: Add, List, Delete")

	subscriptionsList := model.NewAutocompleteData(commandList, "", "List current channel subscriptions")
	subscriptions.AddCommand(subscriptionsList)

	subscriptionsAdd := model.NewAutocompleteData(commandAdd, "owner[/repo] [features]", "Subscribe the current channel to receive notifications from a project")
	subscriptionsAdd.AddTextArgument("Project path: includes user or group name with optional slash project name", "owner[/repo]", "")
	subscriptionsAdd.AddTextArgument("comma-delimited list of features to subscribe to: issues, merges, pushes, issue_comments, merge_request_comments, pipeline, tag, pull_reviews, label:<labelName>", "[features] (optional)", `/[^,-\s]+(,[^,-\s]+)*/`)
	subscriptions.AddCommand(subscriptionsAdd)

	subscriptionsDelete := model.NewAutocompleteData(commandDelete, "owner[/repo]", "Unsubscribe the current channel from a repository")
	subscriptionsDelete.AddTextArgument("Project path: includes user or group name with optional slash project name", "owner[/repo]", "")
	subscriptions.AddCommand(subscriptionsDelete)

	gitlab.AddCommand(subscriptions)

	me := model.NewAutocompleteData("me", "", "Displays the connected GitLab account")
	gitlab.AddCommand(me)

	pipelines := model.NewAutocompleteData("pipelines", "[command]", "Available commands: Run, Trigger")
	pipelineRun := model.NewAutocompleteData(commandRun, "owner[/repo] [ref]", "Run a pipeline for the provided project")
	pipelineRun.AddTextArgument("Project path: includes user or group name with optional slash project name", "", "owner[/repo] [ref]")
	pipelines.AddCommand(pipelineRun)

	gitlab.AddCommand(pipelines)

	settings := model.NewAutocompleteData("settings", "[setting]", "Update your user settings")
	settingOptions := []model.AutocompleteListItem{{
		HelpText: "Turn notifications on/off",
		Item:     "notifications",
	}, {
		HelpText: "Turn reminders on/off",
		Item:     "reminders",
	}}
	settings.AddStaticListArgument("Setting to update", true, settingOptions)

	value := []model.AutocompleteListItem{{
		HelpText: "Turn setting on",
		Item:     "on",
	}, {
		HelpText: "Turn setting off",
		Item:     "off",
	}}
	settings.AddStaticListArgument("New value", true, value)
	gitlab.AddCommand(settings)

	webhook := model.NewAutocompleteData("webhook", "[command]", "Available Commands: list, add")
	webhookList := model.NewAutocompleteData(commandList, "owner/[repo]", "List existing project or group webhooks")
	webhookList.AddTextArgument("Project path: includes user or group name with optional slash project name", "owner[/repo]", "")
	webhook.AddCommand(webhookList)

	webhookAdd := model.NewAutocompleteData(commandAdd, "owner/[repo] [options] [url] [token]", "Add a project or group webhook")
	webhookAdd.AddTextArgument("Group or Project path: includes user or group name with optional slash project name", "owner[/repo]", "")
	webhookAdd.AddTextArgument("[Optional] options: comma-delimited list of actions to trigger a webhook, defaults to all with SSL verification", "[* or *noSSL] or [PushEvents,][TagPushEvents,][Comments,][ConfidentialComments,][IssuesEvents,][ConfidentialIssuesEvents,][MergeRequestsEvents,][JobEvents,][PipelineEvents,][WikiPageEvents,][SSLverification]", "")
	webhookAdd.AddTextArgument("[Optional] url: URL to be triggered triggered. Defaults to this plugins URL", "[url]", "")
	webhookAdd.AddTextArgument("[Optional] token: Secret for webhook. Defaults to token used in plugin's settings.", "[token]", "")
	webhook.AddCommand(webhookAdd)

	gitlab.AddCommand(webhook)

	setup := model.NewAutocompleteData("setup", "[command]", "Available commands: oauth, webhook, announcement")
	setup.RoleID = model.SystemAdminRoleId
	setup.AddCommand(model.NewAutocompleteData("oauth", "", "Set up the OAuth2 Application in GitLab"))
	setup.AddCommand(model.NewAutocompleteData("webhook", "", "Create a webhook from GitLab to Mattermost"))
	setup.AddCommand(model.NewAutocompleteData("announcement", "", "Announce to your team that they can use GitLab integration"))
	gitlab.AddCommand(setup)

	help := model.NewAutocompleteData("help", "", "Display GiLab Plug Help.")
	gitlab.AddCommand(help)

	about := command.BuildInfoAutocomplete("about")
	gitlab.AddCommand(about)

	return gitlab
}
