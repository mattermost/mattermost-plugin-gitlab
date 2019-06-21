# Mattermost GitLab Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/manland/mattermost-plugin-gitlab/master.svg)

A GitLab plugin for Mattermost. The plugin is currently in beta.

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

__Requires Mattermost 5.10+ and Gitlab 11.2+ see [Compatibility](#Compatibility) for others versions__

1. Install the plugin
    1. Download the tar.gz file of the latest version of the plugin from the [GitHub releases page](https://github.com/mattermost/mattermost-plugin-gitlab/releases)
    2. In Mattermost, go the System Console -> Plugins -> Management
    3. Upload the plugin
2. Register a GitLab OAuth app
    1. Go to https://gitlab.com/profile/applications or https://gitlab.yourdomain.com/profile/applications
        * Use "Mattermost GitLab Plugin - <your company name>" as the name
        * Use "https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/oauth/complete" as the authorization callback URL, replacing `https://your-mattermost-url.com` with your Mattermost URL
        * Check `api` and `read_user` in scopes
        * Submit and copy the Client ID and Secret
    2. In Mattermost, go to System Console -> Plugins -> GitLab
        * Fill in the Gitlab URL, Client ID and Secret and save the settings
3. Create a GitLab webhook
    1. In Mattermost, go to the System Console -> Plugins -> GitLab and copy the "Webhook Secret"
    2. Go to the settings page of your GitLab project and click on "Integrations" in the sidebar
        * Use "https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/webhook" as the URL, replacing `https://your-mattermost-url.com` with your Mattermost URL
        * Paste the webhook secret you copied before into the secret token field
        * Select all events
    3. Save the webhook
    4. __Note for each project you want to receive notifications for or subscribe to, you must create a webhook__
4. Generate an at rest encryption key
    1. Go to the System Console -> Plugins -> GitLab and click "Regenerate" under "At Rest Encryption Key"
    2. Save the settings
4. (Optional) Lock the plugin to a GitLab group
    * Go to System Console -> Plugins -> GitLab and set the GitLab Group field to the name of your GitLab group
4. (Optional) Enable private repositories
    * Go to System Console -> Plugins -> GitLab and set Enable Private Repositories to true
    * Note that if you do this after users have already connected their accounts to GitLab they will need to disconnect and reconnect their accounts to be able to use private repositories
5. Enable the plugin
    * Go to System Console -> Plugins -> Management and click "Enable" underneath the GitLab plugin
6. Test it out
    * In Mattermost, run the slash command `/gitlab connect`

## Compatibility

| Mattermost-Plugin-Gitlab | Mattermost | Gitlab |
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
