# Mattermost Gitlab Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/manland/mattermost-plugin-gitlab/master.svg)

A Gitlab plugin for Mattermost. The plugin is currently in beta.

## Todo

* [ ] __Daily reminders__ - the first time you log in to Mattermost each day, get a post letting you know what issues and merge requests need your attention
* [ ] __Notifications__ - get a direct message in Mattermost when someone mentions you, requests your review, comments on or modifies one of your merge requests/issues, or assigns you on Gitlab
* [ ] __Sidebar buttons__ - stay up-to-date with how many reviews, unread messages, assignments and open pull requests you have with buttons in the Mattermost sidebar
* [ ] __Slash commands__ - interact with the Gitlab plugin using the `/gitlab` slash command
    * __Subscribe to a respository__ - Use `/gitlab subscribe` to subscribe a Mattermost channel to receive posts for new pull requests and/or issues in a Gitlab repository
    * __Get to do items__ - Use `/gitlab todo` to get an ephemeral message with items to do in Gitlab
    * __Update settings__ - Use `/gitlab settings` to update your settings for the plugin
    * __And more!__ - Run `/gitlab help` to see what else the slash command can do
* [ ] __Supports Gitlab Enterprise__ - Works with SaaS and Enterprise versions of Gitlab

## Installation

__Requires Mattermost 5.2 or higher__

1. Install the plugin
    1. Download the latest version of the plugin from the GitHub releases page
    2. In Mattermost, go the System Console -> Plugins -> Management
    3. Upload the plugin
2. Register a Gitlab OAuth app
    1. Go to https://gitlab.com/profile/applications
        * Use "Mattermost Gitlab Plugin - <your company name>" as the name
        * Use "https://github.com/manland/mattermost-plugin-gitlab" as the homepage
        * Use "https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/oauth/complete" as the authorization callback URL, replacing `https://your-mattermost-url.com` with your Mattermost URL
        * Submit and copy the Client ID and Secret
    2. In Mattermost, go to System Console -> Plugins -> Gitlab
        * Fill in the Client ID and Secret and save the settings
3. Create a Gitlab webhook
    1. In Mattermost, go to the System Console -> Plugins -> Gitlab and copy the "Webhook Secret"
    2. Go to the settings page of your Gitlab group and click on "Webhooks" in the sidebar
        * Click "Add webhook"
        * Use "https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/webhook" as the payload URL, replacing `https://your-mattermost-url.com` with your Mattermost URL
        * Change content type to "application/json"
        * Paste the webhook secret you copied before into the secret field
        * Select the events: Issues, Issue comments, Pull requests, Pull request reviews, Pull request review comments, Pushes, Branch or Tag creation and Branch or Tag deletion
    3. Save the webhook
    4. __Note for each organization you want to receive notifications for or subscribe to, you must create a webhook__
4. Configure a bot account
    1. Create a new Mattermost user, through the regular UI or the CLI with the username "gitlab"
    2. Go to the System Console -> Plugins -> Gitlab and select this user in the User setting
    3. Save the settings
4. Generate an at rest encryption key
    1. Go to the System Console -> Plugins -> Gitlab and click "Regenerate" under "At Rest Encryption Key"
    2. Save the settings
4. (Optional) Lock the plugin to a Gitlab group
    * Go to System Console -> Plugins -> Gitlab and set the Gitlab Group field to the name of your Gitlab group
4. (Optional) Enable private repositories
    * Go to System Console -> Plugins -> Gitlab and set Enable Private Repositories to true
    * Note that if you do this after users have already connected their accounts to Gitlab they will need to disconnect and reconnect their accounts to be able to use private repositories
4. (Enterprise only) Set your Enterprise URLs
    * Go to System Console -> Plugins -> Gitlab and set the Enterprise Base URL and Enterprise Upload URL fields to your Gitlab Enterprise URLs, ex: `https://gitlab.example.com`
    * The Base and Upload URLs are often the same
5. Enable the plugin
    * Go to System Console -> Plugins -> Management and click "Enable" underneath the Gitlab plugin
6. Test it out
    * In Mattermost, run the slash command `/gitlab connect`

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

## Frequently Asked Questions

### How do I connect a repository instead of an organization?

Set up your Gitlab webhook from the repository instead of the organization. Notifications and subscriptions will then be sent only for repositories you create webhooks for.

The reminder and `/gitlab todo` will still search the whole organization, but only list items assigned to you.

## Feedback and Feature Requests

Feel free to create a GitHub issue or [join the Gitlab Plugin channel on our community Mattermost instance](https://pre-release.mattermost.com/core/channels/gitlab-plugin) to discuss.
