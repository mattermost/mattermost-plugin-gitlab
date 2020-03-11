package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const COMMAND_HELP = `* |/gitlab connect| - Connect your Mattermost account to your GitLab account
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
  * |value| can be "on" or "off"`

const webhookHowToURL = "https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook"

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
		text := "###### Mattermost GitLab Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)
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

	default:
		return p.getCommandResponse(args, "Unknown action, please use `/gitlab help` to see all actions available."), nil
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
	var hookStatusMessage string
	hasHook, err := p.HasProjectHook(info, namespace, project)
	if err == nil {
		if hasHook {
			//web hook found
			return fmt.Sprintf("Successfully subscribed to %s.", fullPath)
		}
		//no web hook found
		hookStatusMessage = fmt.Sprintf(
			"Please [setup a WebHook](%s/%s/%s/hooks) in GitLab to complete integration. See [setup instructions](%s) for more info.",
			config.GitlabURL,
			namespace,
			project,
			webhookHowToURL,
		)
	} else {
		//unable to get web hook info
		hookStatusMessage = fmt.Sprintf(
			"Unable to determine status of Webhook. See [setup instructions](%s) to validate.",
			webhookHowToURL,
		)
	}

	return fmt.Sprintf("Successfully subscribed to %s. %s", fullPath, hookStatusMessage)
}
