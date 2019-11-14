# Mattermost GitLab Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-gitlab/master.svg)](https://circleci.com/gh/mattermost/mattermost-plugin-gitlab)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-gitlab/master.svg)](https://codecov.io/gh/mattermost/mattermost-plugin-gitlab)

A GitLab plugin for Mattermost. The plugin is currently in beta.

Originally developed by [Romain Maneschi](https://github.com/manland).

![GitLab Plugin screenshot](https://user-images.githubusercontent.com/1492516/57301593-afbb1b80-70d9-11e9-9134-809e5cc69a45.png)

## Features

* __Daily reminders__ - the first time you log in to Mattermost each day, get a post letting you know what issues and merge requests need your attention
* __Notifications__ - get a direct message in Mattermost when someone mentions you, requests your review, comments on or modifies one of your merge requests/issues, or assigns you on GitLab
* __Sidebar buttons__ - stay up-to-date with how many reviews, unread messages, assignments and open merge requests you have with buttons in the Mattermost sidebar
* __Slash commands__ - interact with the GitLab plugin using the `/gitlab` slash command
    * __Subscribe to a respository__ - Use `/gitlab subscribe` to subscribe a Mattermost channel to receive posts for new merge requests and/or issues in a GitLab repository
    * __Get to do items__ - Use `/gitlab todo` to get an ephemeral message with items to do in GitLab
    * __Update settings__ - Use `/gitlab settings` to update your settings for the plugin
    * __And more!__ - Run `/gitlab help` to see what else the slash command can do
* __Supports GitLab On Premise__ - Works with SaaS and On Premise versions of GitLab

## Installation

In Mattermost 5.14 and later, the GitLab plugin is pre-packaged and no steps are required for installation. You can go directly to [Configuration](#configuration).

In Mattermost 5.13 and earlier, follow these steps:
1. Go to https://github.com/mattermost/mattermost-plugin-gitlab/releases to download the latest release file in zip or tar.gz format.
2. Upload the file through **System Console > Plugins > Management**, or manually upload it to the Mattermost server under plugin directory. See [documentation](https://docs.mattermost.com/administration/plugins.html#set-up-guide) for more details.

See [Compatibility](#Compatibility) for supported versions.

## Configuration

### Step 1: Register an OAuth application in GitLab
   
1. Go to https://gitlab.com/profile/applications or https://gitlab.yourdomain.com/profile/applications to register an OAuth app.
2. Set the following values:
   - **Name**: `Mattermost GitLab Plugin - <your company name>`
   - **Redirect URI**: `https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/oauth/complete`, replacing `https://your-mattermost-url.com` with your Mattermost URL
3. Select `api` and `read_user` in **Scopes**
4. Save the application. Copy the **Application ID* and **Secret** fields in the resulting screen.
2. In Mattermost, go to **System Console > Plugins > GitLab**, and enter the **GitLab URL**, **GitLab OAuth Client ID**, and **Gitlab OAuth Client Secret**

### Step 2: Create a GitLab webhook

__Note for each project you want to receive notifications for or subscribe to, you must create a webhook__

1. In Mattermost, go to **System Console > Plugins > GitLab**, generate a new value for **Webhook Secret**. Copy it as you will use it in a later step.
2. In GitLab, go to the project you want to subscribe to, select **Settings** then **Integrations** in the sidebar.
3. Set the following values:
   - **URL**: `https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/webhook`, replacing `https://your-mattermost-url.com` with your Mattermost URL
   - **Secret Token**: the webhook secret you copied previously
4. Select all the events in **Triggers**.
5. Add the webhook.

### Step 3: Configure plugin in Mattermost

1. Generate an at rest encryption key
   - Go to the System Console -> Plugins -> GitLab and click "Regenerate" under "At Rest Encryption Key"
   - Save the settings
2. (Optional) Lock the plugin to a GitLab group
   - Go to System Console -> Plugins -> GitLab and set the GitLab Group field to the name of your GitLab group
3. (Optional) Enable private repositories
   - Go to System Console -> Plugins -> GitLab and set Enable Private Repositories to true
   - Note that if you do this after users have already connected their accounts to GitLab they will need to disconnect and reconnect their accounts to be able to use private repositories
4. Enable the plugin
   - Go to System Console -> Plugins -> Management and click "Enable" underneath the GitLab plugin
5. Test it out
   - In Mattermost, run the slash command `/gitlab connect`

1. Go to **System Console > Plugins > GitLab** and do the following:
  - Generate a new value for **At Rest Encryption Key**.
  - (Optional) **GitLab Group**: Lock the plugin to a single GitLab group by setting this field to the name of your GitLab group.
  - (Optional) **Enable Private Repositories**: Allow the plugin to receive notifications from private repositories by setting this value to true.
    When enabled, existing users must reconnect their accounts to gain access to private project. Affected users will be notified by the plugin once private repositories are enabled.
2. Hit **Save**.
3. Go to **System Console > Plugins > Management** and click **Enable** to enable the GitLab plugin.

You're all set! To test it, run the `/gitlab connect` slash command to connect your Mattermost account with GitLab.

## Compatibility

| Mattermost-Plugin-Gitlab | Mattermost | GitLab |
|:-----------------------:|:----------:|:------:|
|        0.3.0            |     5.10+  |  11.2+ |
|        0.2.0            |     5.8+   |  11.2+ |
|        0.1.0            |     5.8+   |  11.2+ |

## Developing 

This plugin contains both a server and web app portion.

Use `make dist` to build distributions of the plugin that you can upload to a Mattermost server.

Use `make check-style` to check the style.

Use `make deploy` to deploy the plugin to your local server. Before running `make deploy` you need to set a few environment variables:

```
export MM_SERVICESETTINGS_SITEURL=http://localhost:8065
export MM_ADMIN_USERNAME=admin
export MM_ADMIN_PASSWORD=password
```

## Inspiration

This project is a fork of the [mattermost-plugin-github](https://github.com/mattermost/mattermost-plugin-github). Thanks to all contributors of it.

## Feedback and Feature Requests

Feel free to create a GitHub issue or [join the GitLab Plugin channel on our community Mattermost instance](https://pre-release.mattermost.com/core/channels/gitlab-plugin) to discuss.
