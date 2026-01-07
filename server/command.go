// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/command"
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
	* confidential_issues - includes new and closed confidential issues
	* jobs - includes jobs status updates
	* merges - includes new and closed merge requests
    * pushes - includes pushes
	* issue_comments - includes new issue comments
	* merge_request_comments - include new merge-request comments
	* pipeline - includes pipeline runs
	* tag - include tag creation
    * pull_reviews - includes merge request reviews
	* label:"<label-1-name>","<label-2-name>" - must include "merges" or "issues" in feature list when using labels
	* deployments - includes deployments
	* releases - includes releases
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
	 * DeploymentEvents
	 * ReleaseEvents
	 * SSLverification
  * |url| is the URL that will be called when triggered. Defaults to this plugins URL
  * |token| Secret token. Defaults to secret token used in plugin's settings.
* |/gitlab about| - Display build information about the plugin
`
const (
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

	invalidSubscribeSubCommand           = "Invalid subscribe command. Available commands are add, delete, and list"
	missingOrgOrRepoFromSubscribeCommand = "Please provide the owner[/repo]"

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
		AutoCompleteDesc:     "Available commands: connect, disconnect, instance, todo, subscriptions, me, pipelines, settings, webhook, setup, help, about",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     p.getAutocompleteData(config),
		AutocompleteIconData: iconData,
	}, nil
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string, isEphemeralPost bool) {
	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: args.ChannelId,
		RootId:    args.RootId,
		Message:   text,
	}

	if isEphemeralPost {
		p.client.Post.SendEphemeralPost(args.UserId, post)
		return
	}

	if err := p.client.Post.CreatePost(post); err != nil {
		p.client.Log.Error("Failed to create post", "error", err.Error())
	}
}

func (p *Plugin) getCommandResponse(args *model.CommandArgs, text string, isEphemeralPost bool) *model.CommandResponse {
	p.postCommandResponse(args, text, isEphemeralPost)
	return &model.CommandResponse{}
}

type authenticatedCommandHandlerFunc func(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError)

type unauthenticatedCommandHandlerFunc func(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError)

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

	defer p.recoverFromPanic(args)

	unauthenticatedHandlers := map[string]unauthenticatedCommandHandlerFunc{
		"about":    p.handleAbout,
		"setup":    p.handleSetup,
		"instance": p.handleInstance,
		"connect":  p.handleConnect,
		"help":     p.handleHelp,
		"":         p.handleHelp,
	}
	if handler, ok := unauthenticatedHandlers[action]; ok {
		return handler(args, parameters)
	}

	config := p.getConfiguration()
	if err := config.IsValid(); err != nil {
		return p.handleConfigError(args, err)
	}

	info, apiErr := p.getGitlabUserInfoByMattermostID(args.UserId)
	if apiErr != nil {
		return p.handleUserNotConnected(args, apiErr)
	}

	authenticatedHandlers := map[string]authenticatedCommandHandlerFunc{
		"subscriptions": p.handleSubscribe,
		"subscription":  p.handleSubscribe,
		"subscribe":     p.handleSubscribe,
		"unsubscribe":   p.handleUnsubscribe,
		"disconnect":    p.handleDisconnect,
		"todo":          p.handleTodo,
		"issue":         p.handleIssue,
		"me":            p.handleMe,
		"settings":      p.handleSettings,
		"webhook":       p.handleWebhookHandler,
		"pipelines":     p.handlePipelines,
	}
	if handler, ok := authenticatedHandlers[action]; ok {
		return handler(ctx, args, parameters, info)
	}

	return p.getCommandResponse(args, unknownActionMessage, true), nil
}

func (p *Plugin) handleConfigError(args *model.CommandArgs, err error) (*model.CommandResponse, *model.AppError) {
	isSysAdmin, sysErr := p.isAuthorizedSysAdmin(args.UserId)
	var text string
	switch {
	case sysErr != nil:
		text = "Error checking user's permissions"
		p.client.Log.Warn(text, "error", sysErr.Error())
	case isSysAdmin:
		text = "Before using this plugin, you'll need to configure it by running `/gitlab setup`"
	default:
		text = "Please contact your system administrator to configure the GitLab plugin."
	}

	p.postCommandResponse(args, text, true)
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleInstance(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	if len(parameters) < 1 {
		return p.getCommandResponse(args, "Please specify the instance command.", true), nil
	}

	switch parameters[0] {
	case "install":
		return p.handleInstallInstance(args, parameters[1:])
	case "uninstall":
		return p.handleUnInstallInstance(args, parameters[1:])
	case "set-default":
		return p.handleSetDefaultInstance(args, parameters[1:])
	case "list":
		return p.handleListInstance(args, parameters[1:])
	default:
		return p.getCommandResponse(args, "Unknown instance command. Available commands: install, uninstall, set-default, list", true), nil
	}
}

func (p *Plugin) handleInstallInstance(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	userID := args.UserId
	isSysAdmin, err := p.isAuthorizedSysAdmin(userID)
	if err != nil {
		p.client.Log.Warn("Failed to check if user is System Admin", "error", err.Error())
		p.postCommandResponse(args, "Error checking user's permissions", true)
		return &model.CommandResponse{}, nil
	}

	if !isSysAdmin {
		p.postCommandResponse(args, "Only System Admins are allowed to set up the plugin.", true)
		return &model.CommandResponse{}, nil
	}

	err = p.flowManager.StartOauthWizard(userID)
	if err != nil {
		p.postCommandResponse(args, err.Error(), true)
	}

	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleUnInstallInstance(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	if len(parameters) < 1 {
		return p.getCommandResponse(args, "Please specify the instance name.", true), nil
	}

	instanceName := strings.Join(parameters, " ")

	err := p.uninstallInstance(instanceName)
	if err != nil {
		return p.getCommandResponse(args, err.Error(), true), nil
	}

	return p.getCommandResponse(args, fmt.Sprintf("Instance '%s' has been uninstalled.", instanceName), true), nil
}

func (p *Plugin) handleSetDefaultInstance(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	if len(parameters) < 1 {
		return p.getCommandResponse(args, "Please specify the instance name.", true), nil
	}

	instanceName := strings.Join(parameters, " ")
	err := p.setDefaultInstance(instanceName)
	if err != nil {
		return p.getCommandResponse(args, err.Error(), true), nil
	}

	return p.getCommandResponse(args, fmt.Sprintf("Instance '%s' has been set as the default.", instanceName), true), nil
}

func (p *Plugin) handleListInstance(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	instanceDetailMap, err := p.getInstanceConfigMap()
	if err != nil {
		p.client.Log.Warn("Failed to get instance list", "error", err.Error())
		return p.getCommandResponse(args, "Error retrieving instance list.", true), nil
	}

	if len(instanceDetailMap) == 0 {
		return p.getCommandResponse(args, "No GitLab instances are currently installed.", true), nil
	}

	var builder strings.Builder
	builder.WriteString("### Installed GitLab Instances\n")
	builder.WriteString("| Instance Name | Instance URL |\n")
	builder.WriteString("|--------------|--------------|\n")
	for name, instanceConfiguration := range instanceDetailMap {
		builder.WriteString(fmt.Sprintf("| %s | %s |\n", name, instanceConfiguration.GitlabURL))
	}

	return p.getCommandResponse(args, builder.String(), true), nil
}

func (p *Plugin) handleUserNotConnected(args *model.CommandArgs, apiErr *APIErrorResponse) (*model.CommandResponse, *model.AppError) {
	text := "Unknown error."
	if apiErr.ID == APIErrorIDNotConnected {
		text = "You must connect your account to GitLab first. Either click on the GitLab logo in the bottom left of the screen or enter `/gitlab connect`."
	}
	return p.getCommandResponse(args, text, true), nil
}

func (p *Plugin) handleAbout(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	text, err := command.BuildInfo(model.Manifest{
		Id:      manifest.Id,
		Version: manifest.Version,
		Name:    manifest.Name,
	})
	if err != nil {
		text = errors.Wrap(err, "failed to get build info").Error()
	}
	p.postCommandResponse(args, text, true)
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleConnect(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	if len(parameters) < 1 {
		return p.getCommandResponse(args, "Please specify the instance name.", true), nil
	}

	// Set the default instance for the user before connecting
	instanceName := parameters[0]
	err := p.setDefaultInstance(instanceName)
	if err != nil {
		return p.getCommandResponse(args, err.Error(), true), nil
	}

	pluginURL := getPluginURL(p.client)
	if pluginURL == "" {
		return p.getCommandResponse(args, "Encountered an error connecting to GitLab.", true), nil
	}
	resp := p.getCommandResponse(args, fmt.Sprintf("[Click here to link your GitLab account.](%s/oauth/connect)", pluginURL), true)
	return resp, nil
}

func (p *Plugin) handleHelp(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	text := "###### Mattermost GitLab Plugin - Slash Command Help\n" + strings.ReplaceAll(commandHelp, "|", "`")
	return p.getCommandResponse(args, text, true), nil
}

func (p *Plugin) recoverFromPanic(args *model.CommandArgs) {
	if r := recover(); r != nil {
		p.client.Log.Warn("Recovered from a panic",
			"Command", args.Command,
			"UserId", args.UserId,
			"error", r,
			"stack", string(debug.Stack()))
		p.postCommandResponse(args, "An unexpected error occurred. Please try again later.", true)
		if *p.client.Configuration.GetConfig().ServiceSettings.EnableDeveloper {
			p.postCommandResponse(args, fmt.Sprintf("error: %v, \nstack:\n```%s```", r, string(debug.Stack())), true)
		}
	}
}

func (p *Plugin) handleSetup(args *model.CommandArgs, parameters []string) (*model.CommandResponse, *model.AppError) {
	userID := args.UserId
	isSysAdmin, err := p.isAuthorizedSysAdmin(userID)
	if err != nil {
		p.client.Log.Warn("Failed to check if user is System Admin", "error", err.Error())
		p.postCommandResponse(args, "Error checking user's permissions", true)
		return &model.CommandResponse{}, nil
	}

	if !isSysAdmin {
		p.postCommandResponse(args, "Only System Admins are allowed to set up the plugin.", true)
		return &model.CommandResponse{}, nil
	}

	if len(parameters) == 0 {
		err = p.flowManager.StartSetupWizard(userID, "")
	} else {
		switch parameters[0] {
		case "oauth":
			err = p.flowManager.StartOauthWizard(userID)
		case "webhook":
			err = p.flowManager.StartWebhookWizard(userID)
		case "announcement":
			err = p.flowManager.StartAnnouncementWizard(userID)
		default:
			p.postCommandResponse(args, fmt.Sprintf("Unknown subcommand %v", parameters[0]), true)
			return &model.CommandResponse{}, nil
		}
	}

	if err != nil {
		p.postCommandResponse(args, err.Error(), true)
	}

	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleSubscribe(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
	config := p.getConfiguration()
	message, isEphemeralPost := p.subscribeCommand(ctx, parameters, args.ChannelId, config, info)
	return p.getCommandResponse(args, message, isEphemeralPost), nil
}

func (p *Plugin) handleUnsubscribe(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
	config := p.getConfiguration()
	var message string
	var err error
	var isEphemeralPost bool
	if len(parameters) == 0 {
		message = specifyRepositoryMessage
	} else {
		message, isEphemeralPost, err = p.subscriptionDelete(info, config, parameters[0], args.ChannelId)
		if err != nil {
			message = err.Error()
		}
	}
	return p.getCommandResponse(args, message, isEphemeralPost), nil
}

func (p *Plugin) handleDisconnect(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
	p.disconnectGitlabAccount(args.UserId)
	return p.getCommandResponse(args, "Disconnected your GitLab account.", true), nil
}

func (p *Plugin) handleTodo(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
	_, text, err := p.GetToDo(ctx, info)
	if err != nil {
		p.client.Log.Warn("can't get todo in command", "err", err.Error())
		return p.getCommandResponse(args, "Encountered an error getting your todo items.", true), nil
	}
	return p.getCommandResponse(args, text, true), nil
}

func (p *Plugin) handleIssue(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
	message := p.handleIssueHelper(nil, args, parameters)
	if message != "" {
		p.postCommandResponse(args, message, true)
	}
	return &model.CommandResponse{}, nil
}

func (p *Plugin) handleMe(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
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
		return p.getCommandResponse(args, "Encountered an error getting your GitLab profile.", true), nil
	}

	text := fmt.Sprintf("You are connected to GitLab as:\n# [![image](%s =40x40)](%s) [%s](%s)", gitUser.AvatarURL, gitUser.WebURL, gitUser.Username, gitUser.WebsiteURL)
	return p.getCommandResponse(args, text, true), nil
}

func (p *Plugin) handleSettings(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
	if len(parameters) < 2 {
		return p.getCommandResponse(args, "Please specify both a setting and value. Use `/gitlab help` for more usage information.", true), nil
	}

	setting := parameters[0]
	strValue := parameters[1]
	value := false
	if strValue == SettingOn {
		value = true
	} else if strValue != SettingOff {
		return p.getCommandResponse(args, "Invalid value. Accepted values are: \"on\" or \"off\".", true), nil
	}

	switch setting {
	case SettingNotifications:
		if value {
			if err := p.storeGitlabToUserIDMapping(info.GitlabUsername, info.UserID); err != nil {
				p.client.Log.Warn("can't store GitLab to user id mapping", "err", err.Error())
				return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs", true), nil
			}
			if err := p.storeGitlabIDToUserIDMapping(info.GitlabUsername, info.GitlabUserID); err != nil {
				p.client.Log.Warn("can't store GitLab to GitLab id mapping", "err", err.Error())
				return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs", true), nil
			}
		} else if err := p.deleteGitlabToUserIDMapping(info.GitlabUsername); err != nil {
			p.client.Log.Warn("can't delete GitLab username in kvstore", "err", err.Error())
			return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs", true), nil
		}
		info.Settings.Notifications = value
	case SettingReminders:
		info.Settings.DailyReminder = value
	default:
		return p.getCommandResponse(args, "Unknown setting.", true), nil
	}

	if err := p.storeGitlabUserInfo(info); err != nil {
		p.client.Log.Warn("can't store user info after update by command", "err", err.Error())
		return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs", true), nil
	}

	return p.getCommandResponse(args, "Settings updated.", true), nil
}

func (p *Plugin) handleWebhookHandler(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
	config := p.getConfiguration()
	message := p.webhookCommand(ctx, parameters, info, config.EnablePrivateRepo)
	return p.getCommandResponse(args, message, true), nil
}

func (p *Plugin) handlePipelines(ctx context.Context, args *model.CommandArgs, parameters []string, info *gitlab.UserInfo) (*model.CommandResponse, *model.AppError) {
	message := p.pipelinesCommand(ctx, parameters, args.ChannelId, info)
	return p.getCommandResponse(args, message, true), nil
}

func (p *Plugin) handleIssueHelper(_ *plugin.Context, args *model.CommandArgs, parameters []string) string {
	if len(parameters) == 0 {
		return "Invalid issue command. Available command is 'create'."
	}

	command := parameters[0]
	parameters = parameters[1:]

	switch {
	case command == "create":
		p.openIssueCreateModal(args.UserId, args.ChannelId, strings.Join(parameters, " "))
		return ""
	default:
		return fmt.Sprintf("This command is not implemented yet. Command: %v", command)
	}
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

		siteURL := getSiteURL(p.client)
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
	var confidentialNoteEvents, mergeRequestsEvents, jobEvents, pipelineEvents, wikiPageEvents, deploymentEvents, releaseEvents bool
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
		if all || strings.EqualFold(trigger, "DeploymentEvents") {
			deploymentEvents = true
		}
		if all || strings.EqualFold(trigger, "ReleaseEvents") {
			releaseEvents = true
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
		DeploymentEvents:         deploymentEvents,
		ReleaseEvents:            releaseEvents,
	}
}

func (p *Plugin) subscriptionDelete(userInfo *gitlab.UserInfo, config *configuration, fullPath, channelID string) (string, bool, error) {
	normalizedPath := normalizePath(fullPath, config.GitlabURL)
	deleted, updatedSubscriptions, err := p.Unsubscribe(channelID, normalizedPath)
	if err != nil {
		p.client.Log.Warn("can't unsubscribe channel in command", "err", err.Error())
		return "Encountered an error trying to unsubscribe. Please try again.", true, nil
	}

	if !deleted {
		return "Subscription not found, please check repository name.", true, nil
	}

	p.sendChannelSubscriptionsUpdated(updatedSubscriptions, channelID)

	baseURL := config.GitlabURL
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	owner := strings.Split(normalizedPath, "/")[0]
	remainingPath := strings.Split(normalizedPath, "/")[1:]

	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeout)
	defer cancel()

	var project *gitlabLib.Project
	var getProjectError error
	err = p.useGitlabClient(userInfo, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		//nolint:govet // Ignore variable shadowing warning
		resp, err := p.GitlabClient.GetProject(ctx, info, token, owner, strings.Join(remainingPath, "/"))
		if err != nil {
			getProjectError = err
		} else {
			project = resp
		}
		return nil
	})
	if project == nil || err != nil {
		if err != nil {
			p.client.Log.Warn("Can't get group in subscription delete", "err", err.Error(), "group", normalizedPath)
		}
	}

	var webhookMsg string
	if getProjectError == nil && project != nil {
		webhookMsg = fmt.Sprintf("\n Please delete the [webhook](%s) for this subscription unless it's required for other subscriptions.", fmt.Sprintf("%s%s/-/hooks", baseURL, normalizedPath))
	} else {
		var group *gitlabLib.Group
		var getGroupError error
		err = p.useGitlabClient(userInfo, func(info *gitlab.UserInfo, token *oauth2.Token) error {
			//nolint:govet // Ignore variable shadowing warning
			resp, err := p.GitlabClient.GetGroup(ctx, info, token, owner, strings.Join(remainingPath, "/"))
			if err != nil {
				getGroupError = err
			} else {
				group = resp
			}
			return nil
		})
		if group == nil || err != nil {
			if err != nil {
				p.client.Log.Warn("Can't get project in subscription delete", "err", err.Error(), "project", normalizedPath)
			}
		}
		if getGroupError == nil && group != nil {
			webhookMsg = fmt.Sprintf("\n Please delete the [webhook](%s) for this subscription unless it's required for other subscriptions.", fmt.Sprintf("%sgroups/%s/-/hooks", baseURL, normalizedPath))
		} else {
			webhookMsg = "\n Please delete the webhook for this subscription unless it's required for other subscriptions."
		}
	}

	unsubscribeMessage := fmt.Sprintf("Successfully deleted subscription for %s.", fmt.Sprintf("[%s](%s)", normalizedPath, baseURL+normalizedPath))
	unsubscribeMessage += webhookMsg

	return unsubscribeMessage, false, nil
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

	// Only check the permissions for a project if the project subscription is created (Not a group or a subgroup subscription)
	if project != "" {
		if hasPermission := p.permissionToProject(ctx, info.UserID, namespace, project); !hasPermission {
			msg := "You don't have the permissions to create subscriptions for this project."
			p.client.Log.Warn(msg)
			return msg
		}
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
	hasHookError := false
	if project != "" {
		hasHook, err = p.HasProjectHook(ctx, info, namespace, project)
		if err != nil {
			p.client.Log.Debug("Unable to fetch project webhook data", "Error", err.Error())
			hasHookError = true
		}
	} else {
		hasHook, err = p.HasGroupHook(ctx, info, namespace)
		if err != nil {
			p.client.Log.Debug("Unable to fetch group webhook data", "Error", err.Error())
			hasHookError = true
		}
	}

	hookErrorMessage := ""
	if hasHookError {
		hookErrorMessage = "\n**Note:** We are unable to determine the webhook status for this project. Please contact your project administrator"
	}

	var hookStatusMessage string
	if !hasHook {
		// no web hook found
		hookStatusMessage = fmt.Sprintf("\nA Webhook is needed, run ```/gitlab webhook add %s``` to create one now.%s", fullPath, hookErrorMessage)
	}

	p.sendChannelSubscriptionsUpdated(updatedSubscriptions, channelID)

	return fmt.Sprintf("Successfully subscribed to %s.%s", fullPath, hookStatusMessage)
}

// subscribeCommand process the /gitlab subscribe command.
// It returns a message and handles all errors my including helpful information in the message
func (p *Plugin) subscribeCommand(ctx context.Context, parameters []string, channelID string, config *configuration, info *gitlab.UserInfo) (string, bool) {
	if len(parameters) == 0 {
		return invalidSubscribeSubCommand, true
	}

	subcommand := parameters[0]

	switch subcommand {
	case commandList:
		return p.subscriptionsListCommand(channelID), true
	case commandAdd:
		features := "merges,issues,tag"
		if len(parameters) < 2 {
			return missingOrgOrRepoFromSubscribeCommand, true
		} else if len(parameters) > 2 {
			features = strings.Join(parameters[2:], " ")
		}
		// Resolve namespace and project name
		fullPath := normalizePath(parameters[1], config.GitlabURL)

		return p.subscriptionsAddCommand(ctx, info, config, fullPath, channelID, features), false
	case commandDelete:
		if len(parameters) < 2 {
			return specifyRepositoryMessage, true
		}

		message, isEphemeralPost, err := p.subscriptionDelete(info, config, parameters[1], channelID)
		if err != nil {
			return err.Error(), true
		}
		return message, isEphemeralPost
	default:
		return invalidSubscribeSubCommand, true
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

	foundPipelineSubscription := false
	subs, err := p.GetSubscriptionsByChannel(channelID)
	if err != nil {
		p.client.Log.Warn("Failed to get subscriptions for the channel", "channel_id", channelID, "error", err.Error())
		return txt
	}

	for _, sub := range subs {
		if sub.Repository == namespace && sub.Pipeline() {
			foundPipelineSubscription = true
			break
		}
	}

	if !foundPipelineSubscription {
		txt += fmt.Sprintf("\n\n**Note:** This channel is currently not subscribed to pipeline event for `%s`. Run the command below if would you like to create a subscription.\n\n`/gitlab subscriptions add %s pipeline`", namespace, namespace)
	}

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

func (p *Plugin) getAutocompleteData(config *configuration) *model.AutocompleteData {
	if !config.IsOAuthConfigured() {
		gitlab := model.NewAutocompleteData("gitlab", "[command]", "Available commands: setup, about")

		setup := model.NewAutocompleteData("setup", "", "Set up the GitLab plugin")
		gitlab.AddCommand(setup)

		about := command.BuildInfoAutocomplete("about")
		gitlab.AddCommand(about)

		return gitlab
	}

	gitlab := model.NewAutocompleteData("gitlab", "[command]", "Available commands: connect, disconnect, todo, subscriptions, me, pipelines, settings, webhook, instance, setup, help, about")

	connect := model.NewAutocompleteData("connect", "", "Connect your GitLab account")
	connect.AddStaticListArgument("Instance Name", true, p.getConnectInstanceAutoCompleteData())
	gitlab.AddCommand(connect)

	disconnect := model.NewAutocompleteData("disconnect", "", "disconnect your GitLab account")
	gitlab.AddCommand(disconnect)

	instance := model.NewAutocompleteData("instance", "[command]", "Install, Uninstall, List, Set-Default Instance")

	install := model.NewAutocompleteData("install", "", "Install GitLab Instance")
	instance.AddCommand(install)

	setDefault := model.NewAutocompleteData("set-default", "", "Set the default GitLab instance to use")
	setDefault.AddStaticListArgument("Instance Name", true, p.getDefaultInstanceAutoCompleteData())
	instance.AddCommand(setDefault)

	uninstall := model.NewAutocompleteData("uninstall", "", "Uninstall GitLab Instance")
	uninstall.AddStaticListArgument("Instance Name", true, p.getUninstallInstanceAutoCompleteData())
	instance.AddCommand(uninstall)

	list := model.NewAutocompleteData("list", "", "List all installed GitLab instances")
	instance.AddCommand(list)

	gitlab.AddCommand(instance)

	todo := model.NewAutocompleteData("todo", "", "Get a list of todos, assigned issues, assigned merge requests and merge requests awaiting your review")
	gitlab.AddCommand(todo)

	issue := model.NewAutocompleteData("issue", "[command]", "Available commands: create")
	gitlab.AddCommand(issue)

	issueCreate := model.NewAutocompleteData("create", "[title]", "Open a dialog to create a new issue in Gitlab, using the title if provided")
	issue.AddCommand(issueCreate)

	subscriptions := model.NewAutocompleteData("subscriptions", "[command]", "Available commands: Add, List, Delete")

	subscriptionsList := model.NewAutocompleteData(commandList, "", "List current channel subscriptions")
	subscriptions.AddCommand(subscriptionsList)

	subscriptionsAdd := model.NewAutocompleteData(commandAdd, "owner[/repo] [features]", "Subscribe the current channel to receive notifications from a project")
	subscriptionsAdd.AddTextArgument("Project path: includes user or group name with optional slash project name", "owner[/repo]", "")
	subscriptionsAdd.AddTextArgument("comma-delimited list of features to subscribe to: issues, confidential_issues, merges, pushes, issue_comments, merge_request_comments, pipeline, tag, pull_reviews, label:<labelName>, deployments, releases", "[features] (optional)", `/[^,-\s]+(,[^,-\s]+)*/`)
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

func (p *Plugin) getDefaultInstanceAutoCompleteData() []model.AutocompleteListItem {
	return buildInstanceAutocompleteItems(p.getInstanceList(), "Set '%s' as the default instance")
}

func (p *Plugin) getUninstallInstanceAutoCompleteData() []model.AutocompleteListItem {
	return buildInstanceAutocompleteItems(p.getInstanceList(), "Uninstall '%s' instance")
}

func (p *Plugin) getConnectInstanceAutoCompleteData() []model.AutocompleteListItem {
	return buildInstanceAutocompleteItems(p.getInstanceList(), "Connect your Mattermost account to '%s' instance")
}
