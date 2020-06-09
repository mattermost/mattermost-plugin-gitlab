package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const commandHelp = `* |/gitlab connect| - Connect your Mattermost account to your GitLab account
* |/gitlab disconnect| - Disconnect your Mattermost account from your GitLab account
* |/gitlab todo| - Get a list of unread messages and merge requests awaiting your review
* |/gitlab subscribe list| - Will list the current channel subscriptions
* |/gitlab subscribe owner[/repo] [features]| - Subscribe the current channel to receive notifications about opened merge requests and issues for a group or repository
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
* |/gitlab unsubscribe owner/repo| - Unsubscribe the current channel from a repository
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
  * |token| Secret token. Defaults to secrete token used in plugin's settings.
`
const (
	webhookHowToURL               = "https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook"
	inboundWebhookURL             = "plugins/com.github.manland.mattermost-plugin-gitlab/webhook"
	unknownActionMessage          = "Unknown action, please use `/gitlab help` to see all actions available."
	newWebhookEmptySiteURLmessage = "Unable to create webhook. The Mattermot Site URL is not set. " +
		"Set it in the Admin Console or rerun /gitlab webhook add group/project URL including the desired URL."
)

const (
	groupNotFoundError   = "404 {message: 404 Group Not Found}"
	groupNotFoundMessage = "Unable to find GitLab group: "
)

const (
	projectNotFoundError   = "404 {message: 404 Project Not Found}"
	projectNotFoundMessage = "Unable to find project with namespace: "
)

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "gitlab",
		DisplayName:      "GitLab",
		Description:      "Integration with GitLab.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, todo, me, settings, subscribe, unsubscribe, help",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.BotUserID,
		ChannelId: args.ChannelId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func (p *Plugin) getCommandResponse(args *model.CommandArgs, text string) *model.CommandResponse {
	p.postCommandResponse(args, text)
	return &model.CommandResponse{}
}

//ExecuteCommand is the entrypoint for /gitlab commands. It returns a message to display to the user or an error.
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
		text := "###### Mattermost GitLab Plugin - Slash Command Help\n" + strings.Replace(commandHelp, "|", "`", -1)
		return p.getCommandResponse(args, text), nil
	}

	info, apiErr := p.getGitlabUserInfoByMattermostID(args.UserId)
	if apiErr != nil {
		text := "Unknown error."
		if apiErr.ID == API_ERROR_ID_NOT_CONNECTED {
			text = "You must connect your account to GitLab first. Either click on the GitLab logo in the bottom left of the screen or enter `/gitlab connect`."
		}
		return p.getCommandResponse(args, text), nil
	}

	config := p.getConfiguration()

	switch action {
	case "subscribe":
		message := p.subscribeCommand(parameters, args.ChannelId, config, info)
		response := p.getCommandResponse(args, message)
		return response, nil
	case "unsubscribe":

		if len(parameters) == 0 {
			return p.getCommandResponse(args, "Please specify a repository."), nil
		}

		fullPath := normalizePath(parameters[0], config.GitlabURL)
		if deleted, err := p.Unsubscribe(args.ChannelId, fullPath); err != nil {
			p.API.LogError("can't unsubscribe channel in command", "err", err.Error())
			return p.getCommandResponse(args, "Encountered an error trying to unsubscribe. Please try again."), nil
		} else if !deleted {
			return p.getCommandResponse(args, "Subscription not found, please check repository name."), nil
		}

		return p.getCommandResponse(args, fmt.Sprintf("Successfully unsubscribed from %s.", fullPath)), nil

	case "disconnect":
		p.disconnectGitlabAccount(args.UserId)
		return p.getCommandResponse(args, "Disconnected your GitLab account."), nil
	case "todo":
		text, err := p.GetToDo(info)
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
		if strValue == SETTING_ON {
			value = true
		} else if strValue != SETTING_OFF {
			return p.getCommandResponse(args, "Invalid value. Accepted values are: \"on\" or \"off\"."), nil
		}

		if setting == SETTING_NOTIFICATIONS {
			if value {
				if err := p.storeGitlabToUserIDMapping(info.GitlabUsername, info.UserID); err != nil {
					p.API.LogError("can't store GitLab user id mapping", "err", err.Error())
					return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
				}
			} else {
				if err := p.API.KVDelete(info.GitlabUsername + GITLAB_USERNAME_KEY); err != nil {
					p.API.LogError("can't delete GitLab username in kvstore", "err", err.Error())
					return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
				}
			}

			info.Settings.Notifications = value
		} else if setting == SETTING_REMINDERS {
			info.Settings.DailyReminder = value
		} else {
			return p.getCommandResponse(args, "Unknown setting."), nil
		}

		if err := p.storeGitlabUserInfo(info); err != nil {
			p.API.LogError("can't store user info after update by command", "err", err.Error())
			return p.getCommandResponse(args, "Unknown error please retry or ask to an administrator to look at logs"), nil
		}

		return p.getCommandResponse(args, "Settings updated."), nil

	case "webhook":
		message := p.webhookCommand(parameters, info)
		response := p.getCommandResponse(args, message)
		return response, nil

	default:
		return p.getCommandResponse(args, unknownActionMessage), nil
	}
}

// webhookCommand processes the /gitlab webhook commands
func (p *Plugin) webhookCommand(parameters []string, info *gitlab.GitlabUserInfo) string {
	if len(parameters) < 1 {
		return unknownActionMessage
	}
	subCommand := parameters[0]

	switch subCommand {
	case "list":
		if len(parameters) != 2 {
			return unknownActionMessage
		}

		namespace := parameters[1]
		fullPath := strings.Split(namespace, "/")

		var webhookInfo []*gitlab.WebhookInfo
		var err error
		if len(fullPath) == 2 {
			owner := fullPath[0]
			repo := fullPath[1]

			webhookInfo, err = p.GitlabClient.GetProjectHooks(info, owner, repo)
			if err != nil {
				if strings.Contains(err.Error(), projectNotFoundError) {
					return projectNotFoundMessage + namespace
				}
				return err.Error()
			}

		} else {
			owner := fullPath[0]
			webhookInfo, err = p.GitlabClient.GetGroupHooks(info, owner)
			if err != nil {
				if strings.Contains(err.Error(), groupNotFoundError) {
					return groupNotFoundMessage + owner
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

	case "add":
		namespace := parameters[1]
		fullPath := strings.Split(namespace, "/")

		siteURL := *p.API.GetConfig().ServiceSettings.SiteURL
		if siteURL == "" {
			return newWebhookEmptySiteURLmessage
		}

		urlPath := fmt.Sprintf("%v/%s", siteURL, inboundWebhookURL)
		if len(parameters) > 3 {
			urlPath = parameters[3]
		}

		//default to all triggers unless specified
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

		//if project scope
		if len(fullPath) == 2 {
			owner := fullPath[0]
			repo := fullPath[1]

			project, _ := p.GitlabClient.GetProject(info, owner, repo)
			newWebhook, err := p.GitlabClient.NewProjectHook(info, project.ID, hookOptions)
			if err != nil {
				return err.Error()
			}
			return fmt.Sprintf("Webhook Created:\n%s", newWebhook.String())
		}
		// If webhook is group scoped
		if len(fullPath) == 1 {
			groupName := fullPath[0]

			newWebhook, err := p.GitlabClient.NewGroupHook(info, groupName, hookOptions)
			if err != nil {
				return err.Error()
			}
			return fmt.Sprintf("Webhook Created:\n%s", newWebhook.String())
		}
		return fmt.Sprintf("Invalid command")

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

// SubscribeCommand proccess the /gitlab subscribe command.
// It returns a message and handles all errors my including helpful information in the message
func (p *Plugin) subscribeCommand(parameters []string, channelID string, config *configuration, info *gitlab.GitlabUserInfo) string {
	features := "merges,issues,tag"

	txt := ""
	if len(parameters) == 0 {
		return "Please specify a repository or 'list' command."
	} else if len(parameters) == 1 && parameters[0] == "list" {
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
	} else if len(parameters) > 1 {
		features = strings.Join(parameters[1:], " ")
	}

	// Resolve namespace and project name
	fullPath := normalizePath(parameters[0], config.GitlabURL)

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

	// Create subscription
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
		//no web hook found
		hookStatusMessage = fmt.Sprintf(
			"\nA Webhook is needed, run ```/gitlab webhook add %s``` to create one now.",
			fullPath,
		)
	}

	return fmt.Sprintf("Successfully subscribed to %s.%s", fullPath, hookStatusMessage)
}
