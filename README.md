# Mattermost GitLab Plugin

[![Build Status](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-gitlab/master.svg)](https://circleci.com/gh/mattermost/mattermost-plugin-gitlab)
[![Code Coverage](https://img.shields.io/codecov/c/github/mattermost/mattermost-plugin-gitlab/master.svg)](https://codecov.io/gh/mattermost/mattermost-plugin-gitlab)
[![Release](https://img.shields.io/github/v/release/mattermost/mattermost-plugin-gitlab)](https://github.com/mattermost/mattermost-plugin-gitlab/releases/latest)
[![HW](https://img.shields.io/github/issues/mattermost/mattermost-plugin-gitlab/Up%20For%20Grabs?color=dark%20green&label=Help%20Wanted)](https://github.com/mattermost/mattermost-plugin-gitlab/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22Up+For+Grabs%22+label%3A%22Help+Wanted%22)

A GitLab plugin for Mattermost. This plugin supports a two-way integration between Mattermost and GitLab. This plugin supports Software-as-a-Service (SaaS) or on-premises versions of GitLab.

![GitLab Plugin screenshot](https://user-images.githubusercontent.com/13119842/69115984-96b3ff80-0a58-11ea-92a3-9176b6b05a89.png)

Originally developed by [Romain Maneschi](https://github.com/manland). This project is a fork of the [mattermost-plugin-github](https://github.com/mattermost/mattermost-plugin-github). Thanks to all contributors of it.

**Maintainer:** [@wiggin77](https://github.com/wiggin77)

**Co-Maintainer:** [@hanzei](https://github.com/hanzei)

See the [Mattermost Product Documentation](https://docs.mattermost.com/integrate/gitlab-interoperability.html) for details on installing, configuring, enabling, and using this Mattermost integration.

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

## Development
  
This plugin contains both a server and web app portion. Read our documentation about the [Developer Workflow](https://developers.mattermost.com/integrate/plugins/developer-workflow/) and [Developer Setup](https://developers.mattermost.com/integrate/plugins/developer-setup/) for more information about developing and extending plugins.
  
## Help wanted!

If you're interested in joining our community of developers who contribute to Mattermost - check out the current set of issues [that are being requested](https://github.com/mattermost/mattermost-plugin-gitlab/issues?q=is%3Aissue+is%3Aopen+label%3AEnhancement).

You can also find issues labeled ["Help Wanted"](https://github.com/mattermost/mattermost-plugin-gitlab/issues?q=is%3Aissue+is%3Aopen+label%3A%22Help+Wanted%22) in the GitLab plugin repository that we have laid out the primary requirements for and could use some coding help from the community.

## Help and support

For Mattermost customers - please open a [support case](https://mattermost.zendesk.com/hc/en-us/requests/new) to ensure your issue is tracked properly.

For Questions, suggestions, and help - please find us on our forum at [https://forum.mattermost.org/c/plugins](https://forum.mattermost.org/c/plugins).

Alternatively, join our public Mattermost server and join the [Integrations and Apps channel](https://community.mattermost.com/core/channels/integrations).

## Feedback and feature requests

Feel free to create a GitHub issue or [join the GitLab Plugin channel on our community Mattermost instance](https://community.mattermost.com/core/channels/plugin-gitlab) to discuss.

Share your thoughts in the [Plugin: GitLab Channel](https://community.mattermost.com/core/channels/gitlab-plugin) on our Mattermost community!

### Releasing new versions

The version of a plugin is determined at compile time, automatically populating a `version` field in the [plugin manifest](plugin.json):
* If the current commit matches a tag, the version will match after stripping any leading `v`, e.g. `1.3.1`.
* Otherwise, the version will combine the nearest tag with `git rev-parse --short HEAD`, e.g. `1.3.1+d06e53e1`.
* If there is no version tag, an empty version will be combined with the short hash, e.g. `0.0.0+76081421`.

To disable this behaviour, manually populate and maintain the `version` field.

## How to Release

To trigger a release, follow these steps:

1. **For Patch Release:** Run the following command:
    ```
    make patch
    ```
   This will release a patch change.

2. **For Minor Release:** Run the following command:
    ```
    make minor
    ```
   This will release a minor change.

3. **For Major Release:** Run the following command:
    ```
    make major
    ```
   This will release a major change.

4. **For Patch Release Candidate (RC):** Run the following command:
    ```
    make patch-rc
    ```
   This will release a patch release candidate.

5. **For Minor Release Candidate (RC):** Run the following command:
    ```
    make minor-rc
    ```
   This will release a minor release candidate.

6. **For Major Release Candidate (RC):** Run the following command:
    ```
    make major-rc
    ```
   This will release a major release candidate.

