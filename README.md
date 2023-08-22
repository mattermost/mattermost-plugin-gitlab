# Mattermost GitLab Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-gitlab/master.svg)](https://circleci.com/gh/mattermost/mattermost-plugin-gitlab)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-gitlab/master.svg)](https://codecov.io/gh/mattermost/mattermost-plugin-gitlab)
[![Release](https://img.shields.io/github/v/release/mattermost/mattermost-plugin-gitlab)](https://github.com/mattermost/mattermost-plugin-gitlab/releases/latest)
[![HW](https://img.shields.io/github/issues/mattermost/mattermost-plugin-gitlab/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/mattermost/mattermost-plugin-gitlab/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

A GitLab plugin for Mattermost. This plugin supports a two-way integration between Mattermost and GitLab. This plugin supports Software-as-a-Service (SaaS) or on-premises versions of GitLab.

![GitLab Plugin screenshot](https://user-images.githubusercontent.com/13119842/69115984-96b3ff80-0a58-11ea-92a3-9176b6b05a89.png)

Originally developed by [Romain Maneschi](https://github.com/manland). This project is a fork of the [mattermost-plugin-github](https://github.com/mattermost/mattermost-plugin-github). Thanks to all contributors of it.

**Maintainer:** [@mickmister](https://github.com/mickmister)

**Co-Maintainer:** [@hanzei](https://github.com/hanzei)

## Feature summary of GitLab to Mattermost notifications

### Channel subscriptions

Notify your team of the latest updates by sending notifications from your GitLab group or repository to Mattermost channels. When team members log in the first time to Mattermost each day, they can get a post letting them know what issues and merge requests need their attention. They can also get a refresh of new events by selecting **Refresh** from every webhook configured in GitLab.

You can specify which events trigger a notification. They can see:

- issues - includes new and closed issues
- merges - includes new and closed merge requests
- pushes - includes pushes
- issue_comments - includes new issue comments
- merge_request_comments - include new merge-request comments
- pipeline - include pipeline
- tag - include tag creation
- pull_reviews - includes merge request reviews
- label:"<labelname>" - must include "merges" or "issues" in feature list when using a label
- Defaults to "merges,issues,tag"

### Personal notifications: GitLab bot

Each user in Mattermost is connected with their own personal GitLab account. Users can get a direct message in Mattermost when someone mentions them, requests their review, comments on, or modifies one of their merge requests/issues, or assigns them on GitLab.

### Sidebar buttons

Team members can stay up-to-date with how many reviews, todos, assigned issues, and assigned merge requests they have by using buttons in the Mattermost sidebar.

## Admin guide
  
Get started by installing the GitLab plugin from the Marketplace.

### Prerequisites

* The GitLab plugin is included in the Plugin Marketplace in Mattermost v5.16 and above.
* For Mattermost v5.13 and earlier, a manual install is necessary.

## Installation
  
### Marketplace installation

1. Go to **Main Menu > Plugin Marketplace** in Mattermost.
2. Search for "gitlab" or manually find the plugin from the list and select **Install**.
3. After the plugin has downloaded and been installed, select the **Configure** button.

### Manual installation

If your server doesn't have access to the internet, you can download the latest [plugin binary release](https://github.com/mattermost/mattermost-plugin-gitlab/releases) and upload it to your server via **System Console > Plugins > Plugin Management**. The releases on this page are the same used by the Marketplace. 

See the [GitLab plugin release page](https://github.com/mattermost/mattermost-plugin-gitlab/releases) for compatibility considerations.

## Configuration

### Step 1: Register an OAuth application in GitLab

1. Go to https://gitlab.com/-/profile/applications or https://gitlab.yourdomain.com/-/profile/applications to register an OAuth app.
1. Set the following values:
   - **Name**: `Mattermost GitLab Plugin - <your company name>`
   - **Redirect URI**: `https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/oauth/complete`, replacing `https://your-mattermost-url.com` with your Mattermost URL
1. Select `api` and `read_user` in **Scopes**.
1. Save the application. Copy the **Application ID** and **Secret** fields in the resulting screen.
1. In Mattermost, go to **Plugins Marketplace > GitLab > Configure**, and enter the **GitLab URL**, **GitLab OAuth Client ID**, and **GitLab OAuth Client Secret**.

### Step 2: Configure the plugin in Mattermost

1. Go to **System Console > Plugins > GitLab** and do the following:
    - Generate a new value for **Webhook Secret**.
    - Generate a new value for **At Rest Encryption Key**.
    - (Optional) **GitLab Group**: Lock the plugin to a single GitLab group by setting this field to the name of your GitLab group.
    - (Optional) **Enable Private Repositories**: Allow the plugin to receive notifications from private repositories by setting this value to `true`. When enabled, existing users must reconnect their accounts to gain access to private project. Affected users will be notified by the plugin once private repositories are enabled.
1. Select **Save**.
1. Go to **Plugins Marketplace > GitLab > Configure > Enable Plugin** and select **Enable** to enable the GitLab plugin.

### Step 3: Connect your GitLab accounts

Run the `/gitlab connect` slash command to connect your Mattermost account with GitLab.

### Step 4: Subscribe to projects and groups

For each project you want to receive notifications for or subscribe to, you must create a webhook. Run the subscribe slash command to watch events sent from GitLab.

``/gitlab subscriptions add group[/project]``

Run the webhook slash command to have GitLab send events to Mattermost. 

``/gitlab webhook add group[/project]``

For versions prior to 1.2: 

1. In GitLab, go to the project you want to subscribe to, select **Settings > Integrations** in the sidebar.
2. Set the following values:
   - **URL**: `https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/webhook`, replacing `https://your-mattermost-url.com` with your Mattermost URL. Ensure that you add `/plugins/com.github.manland.mattermost-plugin-gitlab/webhook` to the URL or the webhook won't work.
   - **Secret Token**: The webhook secret you copied previously.
3. Select all the events in **Triggers**.
4. Add the webhook.

### Compatibility

| Mattermost-Plugin-Gitlab| Mattermost | GitLab |
|:-----------------------:|:----------:|:------:|
|        0.3.0            |     5.10+  |  11.2+ |
|        0.2.0            |     5.8+   |  11.2+ |
|        0.1.0            |     5.8+   |  11.2+ |

### Update the plugin

When a new version of the plugin is released to the **Plugin Marketplace**, the system will display a prompt asking you to update your current version of the GitLab plugin to the newest one. There may be a warning shown if there is a major version change that **may** affect the installation. Generally, updates are seamless and don't interrupt the user experience in Mattermost.
  
## Mattermost commands user guide

Interact with the GitLab plugin using the `/gitlab` slash command.

### Subscribe to/unsubscribe from a repository

Use `/gitlab subscriptions add owner[/repo] [features]` to subscribe a Mattermost channel to receive posts for new merge requests and/or issues, or other features (as listed above), from a GitLab repository. Ensure that the webhook is configured, otherwise this will not work properly.

Use `/gitlab subscriptions delete owner/repo` to unsubscribe from it.  

`/gitlab subscriptions list` lists what you have subscribed to.

### Connect to/disconnect from GitLab

Connect your Mattermost account to your GitLab account using `/gitlab connect` and disconnect it using`/gitlab disconnect`. 

`/gitlab me` displays the connected GitLab account.

### Get "To Do" items

Use `/gitlab todo` to get a list of todos, assigned issues, assigned merge requests and merge requests awaiting your review.

### Update settings

Use `/gitlab settings [setting] [value]` to update your settings for the plugin.  There are two settings:

- To turn **personal notifications** `on` or `off.
- To turn **reminders** `on` or `off` for when you connect for the first time each day.  

### And more...

Run `/gitlab help` to see what else the slash command can do.

**Tip**: Don't forget to add a webhook in GitLab!

## Development
  
This plugin contains both a server and web app portion. Read our documentation about the [Developer Workflow](https://developers.mattermost.com/extend/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/extend/plugins/developer-setup/) for more information about developing and extending plugins.
  
## Help wanted!

If you're interested in joining our community of developers who contribute to Mattermost - check out the current set of issues [that are being requested](https://github.com/mattermost/mattermost-plugin-gitlab/issues?q=is%3Aissue+is%3Aopen+label%3AEnhancement).

You can also find issues labeled ["Help Wanted"](https://github.com/mattermost/mattermost-plugin-gitlab/issues?q=is%3Aissue+is%3Aopen+label%3A%22Help+Wanted%22) in the GitLab plugin repository that we have laid out the primary requirements for and could use some coding help from the community.

## Help and support

For Mattermost customers - please open a [support case](https://mattermost.zendesk.com/hc/en-us/requests/new) to ensure your issue is tracked properly.

For Questions, suggestions, and help - please find us on our forum at [https://forum.mattermost.org/c/plugins](https://forum.mattermost.org/c/plugins).

Alternatively, join our public Mattermost server and join the [Integrations and Apps channel](https://community-daily.mattermost.com/core/channels/integrations).

## Feedback and feature requests

Feel free to create a GitHub issue or [join the GitLab Plugin channel on our community Mattermost instance](https://community.mattermost.com/core/channels/plugin-gitlab) to discuss.

Share your thoughts in the [Plugin: GitLab Channel](https://community-daily.mattermost.com/core/channels/gitlab-plugin) on our Mattermost community!
