package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-server/plugin"

	"github.com/mattermost/mattermost-server/model"
)

const COMMAND_HELP = `* |/gitlab connect| - Connect your Mattermost account to your Gitlab account
* |/gitlab disconnect| - Disconnect your Mattermost account from your Gitlab account
* |/gitlab todo| - Get a list of unread messages and pull requests awaiting your review
* |/gitlab subscribe list| - Will list the current channel subscriptions
* |/gitlab subscribe owner [features]| - Subscribe the current channel to all available repositories within an organization and receive notifications about opened pull requests and issues
* |/gitlab subscribe owner/repo [features]| - Subscribe the current channel to receive notifications about opened pull requests and issues for a repository
  * |features| is a comma-delimited list of one or more the following:
    * issues - includes new and closed issues
	* pulls - includes new and closed pull requests
    * pushes - includes pushes
	* issue_comments - includes new issue comments
	* merge_request_comments - include new merge-request comments
	* pipeline - include pipeline
	* tag - include tag creation
    * pull_reviews - includes pull request reviews
	* label:"<labelname>" - must include "pulls" or "issues" in feature list when using a label
    * Defaults to "pulls,issues,tag"
* |/gitlab unsubscribe owner/repo| - Unsubscribe the current channel from a repository
* |/gitlab me| - Display the connected Gitlab account
* |/gitlab settings [setting] [value]| - Update your user settings
  * |setting| can be "notifications" or "reminders"
  * |value| can be "on" or "off"`

func getCommand() *model.Command {
	return &model.Command{
		Trigger:          "gitlab",
		DisplayName:      "Gitlab",
		Description:      "Integration with Gitlab.",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: connect, disconnect, todo, me, settings, subscribe, unsubscribe, help",
		AutoCompleteHint: "[command]",
	}
}

func (p *Plugin) getCommandResponse(responseType, text string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: responseType,
		Text:         text,
		Username:     GITLAB_USERNAME,
		IconURL:      p.getConfiguration().ProfileImageURL,
		Type:         model.POST_DEFAULT,
	}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	split := strings.Fields(args.Command)
	command := split[0]
	parameters := []string{}
	action := ""
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
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Encountered an error connecting to Gitlab."), nil
		}

		resp := p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("[Click here to link your Gitlab account.](%s/plugins/%s/oauth/connect)", *config.ServiceSettings.SiteURL, manifest.Id))
		return resp, nil
	}

	info, apiErr := p.getGitlabUserInfoByMattermostID(args.UserId)
	if apiErr != nil {
		text := "Unknown error."
		if apiErr.ID == API_ERROR_ID_NOT_CONNECTED {
			text = "You must connect your account to Gitlab first. Either click on the Gitlab logo in the bottom left of the screen or enter `/gitlab connect`."
		}
		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	}

	client := p.gitlabConnect(*info.Token)

	switch action {
	case "subscribe":
		config := p.getConfiguration()
		features := "pulls,issues,tag"

		txt := ""
		if len(parameters) == 0 {
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Please specify a repository or 'list' command."), nil
		} else if len(parameters) == 1 && parameters[0] == "list" {
			subs, err := p.GetSubscriptionsByChannel(args.ChannelId)
			if err != nil {
				return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, err.Error()), nil
			}

			if len(subs) == 0 {
				txt = "Currently there are no subscriptions in this channel"
			} else {
				txt = "### Subscriptions in this channel\n"
			}
			for _, sub := range subs {
				txt += fmt.Sprintf("* `%s` - %s\n", sub.Repository, sub.Features)
			}
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, txt), nil
		} else if len(parameters) > 1 {
			features = strings.Join(parameters[1:], " ")
		}

		_, owner, repo := parseOwnerAndRepo(parameters[0], config.EnterpriseBaseURL)
		if repo == "" {
			if err := p.SubscribeGroup(client, args.UserId, owner, args.ChannelId, features); err != nil {
				return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, err.Error()), nil
			}

			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Successfully subscribed to organization %s.", owner)), nil
		}

		if err := p.Subscribe(client, args.UserId, owner, repo, args.ChannelId, features); err != nil {
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, err.Error()), nil
		}

		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Successfully subscribed to %s.", repo)), nil
	case "unsubscribe":
		if len(parameters) == 0 {
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Please specify a repository."), nil
		}

		repo := parameters[0]

		if err := p.Unsubscribe(args.ChannelId, repo); err != nil {
			p.API.LogError("can't unsubscribe channel in command", "err", err.Error())
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Encountered an error trying to unsubscribe. Please try again."), nil
		}

		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, fmt.Sprintf("Succesfully unsubscribed from %s.", repo)), nil
	case "disconnect":
		p.disconnectGitlabAccount(args.UserId)
		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Disconnected your Gitlab account."), nil
	case "todo":
		text, err := p.GetToDo(info, client)
		if err != nil {
			p.API.LogError("can't get todo in command", "err", err.Error())
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Encountered an error getting your to do items."), nil
		}
		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	case "me":
		gitUser, _, err := client.Users.CurrentUser()
		if err != nil {
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Encountered an error getting your Gitlab profile."), nil
		}

		text := fmt.Sprintf("You are connected to Gitlab as:\n# [![image](%s =40x40)](%s) [%s](%s)", gitUser.AvatarURL, gitUser.WebsiteURL, gitUser.Username, gitUser.WebsiteURL)
		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	case "help":
		text := "###### Mattermost Gitlab Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)
		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	case "":
		text := "###### Mattermost Gitlab Plugin - Slash Command Help\n" + strings.Replace(COMMAND_HELP, "|", "`", -1)
		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, text), nil
	case "settings":
		if len(parameters) < 2 {
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Please specify both a setting and value. Use `/gitlab help` for more usage information."), nil
		}

		setting := parameters[0]
		if setting != SETTING_NOTIFICATIONS && setting != SETTING_REMINDERS {
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Unknown setting."), nil
		}

		strValue := parameters[1]
		value := false
		if strValue == SETTING_ON {
			value = true
		} else if strValue != SETTING_OFF {
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Invalid value. Accepted values are: \"on\" or \"off\"."), nil
		}

		if setting == SETTING_NOTIFICATIONS {
			if value {
				if err := p.storeGitlabToUserIDMapping(info.GitlabUsername, info.UserID); err != nil {
					p.API.LogError("can't store gitlab user id mapping", "err", err.Error())
					return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Unknown error please retry or ask to an administrator to look at logs"), nil
				}
			} else {
				if err := p.API.KVDelete(info.GitlabUsername + GITLAB_USERNAME_KEY); err != nil {
					p.API.LogError("can't delete gitlab username in kvstore", "err", err.Error())
					return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Unknown error please retry or ask to an administrator to look at logs"), nil
				}
			}

			info.Settings.Notifications = value
		} else if setting == SETTING_REMINDERS {
			info.Settings.DailyReminder = value
		}

		if err := p.storeGitlabUserInfo(info); err != nil {
			p.API.LogError("can't store user info after update by command", "err", err.Error())
			return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Unknown error please retry or ask to an administrator to look at logs"), nil
		}

		return p.getCommandResponse(model.COMMAND_RESPONSE_TYPE_EPHEMERAL, "Settings updated."), nil
	}

	return &model.CommandResponse{}, nil
}
