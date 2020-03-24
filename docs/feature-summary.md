# Feature Summary

## GitLab to Mattermost Notifications

### Channel Subscriptions

Notify your team of the latest updates by sending notifications from your GitLab group or repository to Mattermost channels.  When team members log in the first time to Mattermost each day, they can get a post letting them know what issues and merge requests need their attention. You can specify which events trigger a notification. They can see:

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



![image](.gitbook/assets/image.png)



### Personal Notifications: GitLab Bot

Each user in Mattermost is connected with their own personal GitLab account.  Users can get a direct message in Mattermost when someone mentions them, requests their review, comments on or modifies one of their merge requests/issues, or assigns them on GitLab.



### Sidebar Buttons

Team members can stay up-to-date with how many reviews, unread messages, assignments and open merge requests they have by using buttons in the Mattermost sidebar.



## Mattermost Commands

Interact with the GitLab plugin using the `/gitlab` slash command

### Subscribe to a repository

Use `/gitlab subscribe` to subscribe a Mattermost channel to receive posts for new merge requests and/or issues in a GitLab repository

### Get to-do items

Use `/gitlab todo` to get an ephemeral message with items to do in GitLab

### Update Settings

Use `/gitlab settings` to update your settings for the plugin

### And more ...

Run `/gitlab help` to see what else the slash command can do

