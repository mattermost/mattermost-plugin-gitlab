# Configuration

### Step 1: Register an OAuth application in GitLab

1. Go to https://gitlab.com/profile/applications or https://gitlab.yourdomain.com/profile/applications to register an OAuth app.
2. Set the following values:
   - **Name**: `Mattermost GitLab Plugin - <your company name>`
   - **Redirect URI**: `https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/oauth/complete`, replacing `https://your-mattermost-url.com` with your Mattermost URL
3. Select `api` and `read_user` in **Scopes**
4. Save the application
5. Copy the **Application ID** and **Secret** fields from the resulting screen
6. In Mattermost, go to **Plugins Marketplace > GitLab > Configure**
7. Enter the **GitLab URL**, **GitLab OAuth Client ID**, and **Gitlab OAuth Client Secret**

### Step 2: Configure plugin in Mattermost

1. Go to **Plugins Marketplace &gt; GitLab** and click the **Configure** button or go to **System Console > Plugins > GitLab** and do the following:
   1. Generate a new value for  `Webhook Secret` 
   2. Generate a new value for **At Rest Encryption Key**
     3. (Optional) **GitLab Group**: Lock the plugin to a single GitLab group by setting this field to the name of your GitLab group
     4. (Optional) **Enable Private Repositories**: Allow the plugin to receive notifications from private repositories by setting this value to true. When enabled, existing users must reconnect their accounts to gain access to a private project. Affected users will be notified by the plugin once private repositories are enabled.
   5. Click **Save**.
2. Go to the top of the screen and set **Enable Plugin** to `True`and then click **Save** to enable the GitLab plugin.

### Step 3: Configure Webhooks in GitLab

For each project you want to receive notifications for, or subscribe to, you must create a webhook.

1. In GitLab, go to the project you want to subscribe to, select **Settings** then **Integrations** in the sidebar.
2. Set the following values:
   - **URL**: `https://your-mattermost-url.com/plugins/com.github.manland.mattermost-plugin-gitlab/webhook`, replacing `https://your-mattermost-url.com` with your Mattermost URL.
   - **Secret Token**: the webhook secret you copied previously.
3. Select all the events in **Triggers**.
4. Add the webhook.

### Step 4: Test it

To test it, run the `/gitlab connect` slash command to connect your Mattermost account with GitLab.



If you face issues installing the plugin, see our [Frequently Asked Questions]() for troubleshooting help, or open an issue in the [Mattermost Forum](http://forum.mattermost.org).
