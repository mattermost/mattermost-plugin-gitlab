package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

const commandHelp = `* |/gitlab connect| - Connect your Mattermost account to your GitLab account
* |/gitlab disconnect| - Disconnect your Mattermost account from your GitLab account
* |/gitlab todo| - Get a list of unread messages and merge requests awaiting your review
* |/gitlab subscriptions list| - Will list the current channel subscriptions
* |/gitlab subscriptions add owner[/repo] [features]| - Subscribe the current channel to receive notifications about opened merge requests and issues for a group or repository
  * |features| is a comma-delimited list of one or more the following:
    * issues - includes new and closed issues
	* merges - includes new and closed merge requests
    * pushes - includes pushes
	* issue_comments - includes new issue comments
	* merge_request_comments - include new merge-request comments
	* pipeline - include pipeline
	* tag - include tag creation
    * pull_reviews - includes merge request reviews
	* label:"<labelname>" - must include "merges" or "issues" in feature list when using a label
    * Defaults to "merges,issues,tag"
* |/gitlab subscriptions delete owner/repo| - Unsubscribe the current channel from a repository
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
`
const (
	webhookHowToURL               = "https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook"
	inboundWebhookURL             = "plugins/com.github.manland.mattermost-plugin-gitlab/webhook"
	specifyRepositoryMessage      = "Please specify a repository."
	unknownActionMessage          = "Unknown action, please use `/gitlab help` to see all actions available."
	newWebhookEmptySiteURLmessage = "Unable to create webhook. The Mattermot Site URL is not set. " +
		"Set it in the Admin Console or rerun /gitlab webhook add group/project URL including the desired URL."
)

const (
	groupNotFoundError   = "404 {message: 404 Group Not Found}"
	groupNotFoundMessage = "Unable to find GitLab group: "

	projectNotFoundError   = "404 {message: 404 Project Not Found}"
	projectNotFoundMessage = "Unable to find project with namespace: "

	invalidSubscribeSubCommand = "Invalid subscribe command. Available commands are add, delete, and list"
)

const (
	commandAdd    = "add"
	commandDelete = "delete"
	commandList   = "list"
)

func (p *Plugin) getCommand() (*model.Command, error) {
	iconData, err := command.GetIconData(p.API, "assets/icon.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	return &model.Command{
		Trigger:              "gitlab",
		AutoComplete:         true,
		AutoCompleteDesc:     "Available commands: connect, disconnect, todo, me, settings, subscriptions, webhook, and help",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(),
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
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) getCommandResponse(args *model.CommandArgs, text string) *model.CommandResponse {
	p.postCommandResponse(args, text)
	return &model.CommandResponse{}
}

// ExecuteCommand is the entrypoint for /gitlab commands. It returns a message to display to the user or an error.
func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	var (
		split      = strings.Fields(args.Command)
		command    = split[0]
		action     string
		parameters []string
	)
	if len(split) > 1 {
		action = split[1]
	}
	if len(split) > 2 {
		parameters = split[2:]
	}
	if command != "/gitlab" {
		return &model.CommandResponse{}, nil
	}

	if action == "connect" {
		config := p.API.GetConfig()
		if config.ServiceSettings.SiteURL == nil {
			return p.getCommandResponse(args, "Encountered an error connecting to GitLab."), nil
		}

		resp := p.getCommandResponse(args, fmt.Sprintf("[Click here to link your GitLab account.](%s/plugins/%s/oauth/connect)", *config.ServiceSettings.SiteURL, manifest.ID))
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

	config := p.getConfiguration()

	switch action {
	case "subscriptions", "subscription", "subscribe":
		message := p.subscribeCommand(parameters, args.ChannelId, config, info)
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
		_, text, err := p.GetToDo(info)
		if err != nil {
			p.API.LogError("can't get todo in command", "err", err.Error())
			return p.getCommandResponse(args, "Encountered an error getting your to do items."), nil
		}
		return p.getCommandResponse(args, text), nil
	case "me":
		gitUser, err := p.GitlabClient.GetUserDetails(info)
		if err != nil {
			return p.getCommandResponse(args, "Encountered an error getting your GitLab profile."), nil
		}

		text := fmt.Sprintf("You are connected to GitLab as:\n# [![image](%s =40x40)](%s) [%s](%s)", gitUser.AvatarURL, gitUser.WebsiteURL, gitUser.Username, gitUser.WebsiteURL)
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
					p.API.LogError("can't store GitLab to user id mapping", "err", err.Error())
					return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
				}
				if err := p.storeGitlabIDToUserIDMapping(info.GitlabUsername, info.GitlabUserID); err != nil {
					p.API.LogError("can't store GitLab to GitLab id mapping", "err", err.Error())
					return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
				}
			} else if err := p.deleteGitlabToUserIDMapping(info.GitlabUsername); err != nil {
				p.API.LogError("can't delete GitLab username in kvstore", "err", err.Error())
				return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
			}
			info.Settings.Notifications = value
		case SettingReminders:
			info.Settings.DailyReminder = value
		default:
			return p.getCommandResponse(args, "Unknown setting."), nil
		}

		if err := p.storeGitlabUserInfo(info); err != nil {
			p.API.LogError("can't store user info after update by command", "err", err.Error())
			return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
		}

		return p.getCommandResponse(args, "Settings updated."), nil

	case "webhook":
		message := p.webhookCommand(parameters, info, config.EnablePrivateRepo)
		response := p.getCommandResponse(args, message)
		return response, nil

	default:
		return p.getCommandResponse(args, unknownActionMessage), nil
	}
}

// webhookCommand processes the /gitlab webhook commands
func (p *Plugin) webhookCommand(parameters []string, info *gitlab.UserInfo, enablePrivateRepo bool) string {
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
		group, project, err := p.GitlabClient.ResolveNamespaceAndProject(info, namespace, enablePrivateRepo)
		if err != nil {
			return err.Error()
		}

		var webhookInfo []*gitlab.WebhookInfo
		if project != "" {
			webhookInfo, err = p.GitlabClient.GetProjectHooks(info, group, project)
			if err != nil {
				if strings.Contains(err.Error(), projectNotFoundError) {
					return projectNotFoundMessage + namespace
				}
				return err.Error()
			}
		} else {
			webhookInfo, err = p.GitlabClient.GetGroupHooks(info, group)
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

		siteURL := *p.API.GetConfig().ServiceSettings.SiteURL
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
		group, projectName, namespaceErr := p.GitlabClient.ResolveNamespaceAndProject(info, namespace, enablePrivateRepo)
		if namespaceErr != nil {
			return namespaceErr.Error()
		}
		// If project scope
		if projectName != "" {
			project, err := p.GitlabClient.GetProject(info, group, projectName)
			if err != nil {
				return err.Error()
			}
			newWebhook, err := p.GitlabClient.NewProjectHook(info, project.ID, hookOptions)
			if err != nil {
				return err.Error()
			}
			return fmt.Sprintf("Webhook Created:\n%s", newWebhook.String())
		}
		// If webhook is group scoped
		newWebhook, err := p.GitlabClient.NewGroupHook(info, group, hookOptions)
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
	deleted, err := p.Unsubscribe(channelID, normalizedPath)
	if err != nil {
		p.API.LogError("can't unsubscribe channel in command", "err", err.Error())
		return "Encountered an error trying to unsubscribe. Please try again.", nil
	}

	if !deleted {
		return "Subscription not found, please check repository name.", nil
	}

	return fmt.Sprintf("Successfully deleted subscription for %s.", normalizedPath), nil
}

// subscriptionsListCommand list GitLab subscriptions in a channel
func (p *Plugin) subscriptionsListCommand(channelID string) string {
	var txt string
	subs, err := p.GetSubscriptionsByChannel(channelID)
	if err != nil {
		txt = err.Error()
		return txt
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
func (p *Plugin) subscriptionsAddCommand(info *gitlab.UserInfo, config *configuration, fullPath, channelID, features string) string {
	var err error
	namespace, project, err := p.GitlabClient.
		ResolveNamespaceAndProject(info, fullPath, config.EnablePrivateRepo)

	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) {
			return "Resource with such path is not found."
		} else if errors.Is(err, gitlab.ErrPrivateResource) {
			return "Requested resource is private."
		}
		p.API.LogError(
			"unable to resolve subscription namespace and project name",
			"err", err.Error(),
		)
		return err.Error()
	}

	if subscribeErr := p.Subscribe(info, namespace, project, channelID, features); subscribeErr != nil {
		p.API.LogError(
			"failed to subscribe",
			"namespace", namespace,
			"project", project,
			"err", subscribeErr.Error(),
		)
		return subscribeErr.Error()
	}
	var hasHook bool
	if project != "" {
		hasHook, err = p.HasProjectHook(info, namespace, project)
		if err != nil {
			return fmt.Sprintf(
				"Unable to determine status of Webhook. See [setup instructions](%s) to validate.",
				webhookHowToURL,
			)
		}
	} else {
		hasHook, err = p.HasGroupHook(info, namespace)
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
	return fmt.Sprintf("Successfully subscribed to %s.%s", fullPath, hookStatusMessage)
}

// subscribeCommand process the /gitlab subscribe command.
// It returns a message and handles all errors my including helpful information in the message
func (p *Plugin) subscribeCommand(parameters []string, channelID string, config *configuration, info *gitlab.UserInfo) string {
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

		return p.subscriptionsAddCommand(info, config, fullPath, channelID, features)
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

func getAutocompleteData() *model.AutocompleteData {
	gitlabCommand := model.NewAutocompleteData("gitlab", "[command]", "Available commands: connect, disconnect, todo, subscribe, unsubscribe, me, settings, webhook")

	connect := model.NewAutocompleteData("connect", "", "Connect your GitLab account")
	gitlabCommand.AddCommand(connect)

	disconnect := model.NewAutocompleteData("disconnect", "", "disconnect your GitLab account")
	gitlabCommand.AddCommand(disconnect)

	todo := model.NewAutocompleteData("todo", "", "Get a list of unread messages and merge requests awaiting your review")
	gitlabCommand.AddCommand(todo)

	subscriptions := model.NewAutocompleteData("subscriptions", "[command]", "Available commands: Add, List, Delete")

	subscriptionsList := model.NewAutocompleteData(commandList, "", "List current channel subscriptions")
	subscriptions.AddCommand(subscriptionsList)

	subscriptionsAdd := model.NewAutocompleteData(commandAdd, "owner[/repo] [features]", "Subscribe the current channel to receive notifications from a project")
	subscriptionsAdd.AddTextArgument("Project path: includes user or group name with optional slash project name", "owner[/repo]", "")
	subscriptionsAdd.AddTextArgument("Features: comma-delimited list of features to subscribe to", "[issues,][merges,][pushes,][issue_comments,][merge_request_comments,][pipeline,][tag,][pull_reviews,][label:<labelName>]", "")
	subscriptions.AddCommand(subscriptionsAdd)

	subscriptionsDelete := model.NewAutocompleteData(commandDelete, "owner[/repo]", "Unsubscribe the current channel from a repository")
	subscriptionsDelete.AddTextArgument("Project path: includes user or group name with optional slash project name", "owner[/repo]", "")
	subscriptions.AddCommand(subscriptionsDelete)

	gitlabCommand.AddCommand(subscriptions)

	me := model.NewAutocompleteData("me", "", "Displays the connected GitLab account")
	gitlabCommand.AddCommand(me)

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
	gitlabCommand.AddCommand(settings)

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

	gitlabCommand.AddCommand(webhook)

	help := model.NewAutocompleteData("help", "", "Display GiLab Plug Help.")
	gitlabCommand.AddCommand(help)

	return gitlabCommand
}
