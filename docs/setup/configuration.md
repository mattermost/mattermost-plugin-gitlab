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

For versions prior to 1.2: 

1. In GitLab, go to the project you want to subscribe to, select **Settings > Integrations** in the sidebar.
2. Set the following values:
   - **URL**: `https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/webhook`, replacing `https://your-mattermost-url.com` with your Mattermost URL. Ensure that you add `/plugins/com.github.manland.mattermost-plugin-gitlab/webhook` to the URL or the webhook won't work.
   - **Secret Token**: The webhook secret you copied previously.
3. Select all the events in **Triggers**.
4. Add the webhook.

## Compatibility

| Mattermost-Plugin-Gitlab| Mattermost | GitLab |
|:-----------------------:|:----------:|:------:|
|        0.3.0            |     5.10+  |  11.2+ |
|        0.2.0            |     5.8+   |  11.2+ |
|        0.1.0            |     5.8+   |  11.2+ |
