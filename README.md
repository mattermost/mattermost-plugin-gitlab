# Mattermost/GitLab Integration

The GitLab/Mattermost plugin documentation is currently being updated and relocated to a new location: https://mattermost.gitbook.io/gitlab-plugin/ - let us know your thoughts on the new format in the [Plugin: GitLab Channel](https://community-daily.mattermost.com/core/channels/gitlab-plugin) on our Mattermost community!

# Mattermost GitLab Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-gitlab/master.svg)](https://circleci.com/gh/mattermost/mattermost-plugin-gitlab)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-gitlab/master.svg)](https://codecov.io/gh/mattermost/mattermost-plugin-gitlab)
[![Release](https://img.shields.io/github/v/release/mattermost/mattermost-plugin-gitlab)](https://github.com/mattermost/mattermost-plugin-gitlab/releases/latest)
[![HW](https://img.shields.io/github/issues/mattermost/mattermost-plugin-gitlab/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/mattermost/mattermost-plugin-gitlab/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

**Maintainer:** [@iomodo](https://github.com/iomodo)
**Co-Maintainer:** [@hanzei](https://github.com/hanzei)

A GitLab plugin for Mattermost. Originally developed by [Romain Maneschi](https://github.com/manland).

![GitLab Plugin screenshot](https://user-images.githubusercontent.com/13119842/69115984-96b3ff80-0a58-11ea-92a3-9176b6b05a89.png)

## Features

* __Daily reminders__ - The first time you log in to Mattermost each day, get a post letting you know what issues and merge requests need your attention.
* __Notifications__ - Receive a direct message in Mattermost when someone mentions you, requests your review, comments on, or modifies one of your merge requests/issues, or assigns you on GitLab.
* __Sidebar buttons__ - Stay up to date with how many reviews, unread messages, assignments, and open merge requests you have with buttons in the Mattermost sidebar.
* __Slash commands__ - Interact with the GitLab plugin using the `/gitlab` slash command.
    * __Subscribe to a repository__ - Use `/gitlab subscriptions add` to subscribe a Mattermost channel to receive posts for new merge requests and/or issues in a GitLab repository.
    * __Get to do items__ - Use `/gitlab todo` to get an ephemeral message with items to do in GitLab.
    * __Update settings__ - Use `/gitlab settings` to update your settings for the plugin.
    * __And more!__ - Run `/gitlab help` to see what else the slash command can do.
* __Supports GitLab On Premise__ - Works with SaaS and on-prem versions of GitLab.

## Installation

From Mattermost 5.16 and later, the GitLab plugin is included in the Plugin Marketplace which can be accessed from **Main Menu > Plugins Marketplace**. You can install the GitLab plugin and then configure it via the [Plugin Marketplace "Configure" button](#configuration).

In Mattermost 5.13 and earlier, follow these steps:

1. Go to https://github.com/mattermost/mattermost-plugin-gitlab/releases to download the latest release file in zip or tar.gz format.
2. Upload the file through **System Console > Plugins > Management**, or manually upload it to the Mattermost server under plugin directory. See [documentation](https://docs.mattermost.com/administration/plugins.html#set-up-guide) for more details.

See [Compatibility](#Compatibility) for supported versions.

## Configuration

### Step 1: Register an OAuth Application in GitLab

1. Go to https://gitlab.com/profile/applications or https://gitlab.yourdomain.com/profile/applications to register an OAuth app.
1. Set the following values:
   - **Name**: `Mattermost GitLab Plugin - <your company name>`
   - **Redirect URI**: `https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/oauth/complete`, replacing `https://your-mattermost-url.com` with your Mattermost URL
1. Select `api` and `read_user` in **Scopes**.
1. Save the application. Copy the **Application ID** and **Secret** fields in the resulting screen.
1. In Mattermost, go to **Plugins Marketplace > GitLab > Configure**, and enter the **GitLab URL**, **GitLab OAuth Client ID**, and **GitLab OAuth Client Secret**.

### Step 2: Configure the Plugin in Mattermost

1. Go to **System Console > Plugins > GitLab** and do the following:
  - Generate a new value for **Webhook Secret**.
  - Generate a new value for **At Rest Encryption Key**.
  - (Optional) **GitLab Group**: Lock the plugin to a single GitLab group by setting this field to the name of your GitLab group.
  - (Optional) **Enable Private Repositories**: Allow the plugin to receive notifications from private repositories by setting this value to `true`. When enabled, existing users must reconnect their accounts to gain access to private project. Affected users will be notified by the plugin once private repositories are enabled.
1. Hit **Save**.
1. Go to **Plugins Marketplace > GitLab > Configure > Enable Plugin** and click **Enable** to enable the GitLab plugin.

### Step 3: Connect Your GitLab Accounts

Run the `/gitlab connect` slash command to connect your Mattermost account with GitLab.

### Step 4: Subscribe to Projects and Groups

__Note for each project you want to receive notifications for or subscribe to, you must create a webhook.__

Run the subscribe slash command to watch events sent from GitLab.

``/gitlab subscriptions add group[/project]``

Run the webhook slash command to have GitLab send events to Mattermost. 

``/gitlab webhook add group[/project]``

## Compatibility

| Mattermost-Plugin-Gitlab| Mattermost | GitLab |
|:-----------------------:|:----------:|:------:|
|        0.3.0            |     5.10+  |  11.2+ |
|        0.2.0            |     5.8+   |  11.2+ |
|        0.1.0            |     5.8+   |  11.2+ |

## Development

This plugin contains both a server and web app portion. Read our documentation about the [Developer Workflow](https://developers.mattermost.com/extend/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/extend/plugins/developer-setup/) for more information about developing and extending plugins.

## Inspiration

This project is a fork of the [mattermost-plugin-github](https://github.com/mattermost/mattermost-plugin-github). Thanks to all contributors of it.

## Feedback and Feature Requests

Feel free to create a GitHub issue or [join the GitLab Plugin channel on our community Mattermost instance](https://community.mattermost.com/core/channels/plugin-gitlab) to discuss.
