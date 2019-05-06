# Changelog

The changelog can be found at https://github.com/mattermost/mattermost-plugin-gitlab/releases.

## 0.3.0 - unreleased

## 0.2.0 - 2019-05-06

- Send refresh to webapp of the author of events received by webhook [#19](https://github.com/manland/mattermost-plugin-gitlab/pull/19)
- Add all webhook events for pipeline (run, fail, success) [#17](https://github.com/manland/mattermost-plugin-gitlab/pull/17)
- Finish group restriction [#21](https://github.com/manland/mattermost-plugin-gitlab/pull/21)
- Finish private repositories on/off [#18](https://github.com/manland/mattermost-plugin-gitlab/pull/18)
- Finish all webhook implementation [#16](https://github.com/manland/mattermost-plugin-gitlab/pull/16): 
    - MergeEvent
    - IssueEvent
    - IssueCommentEvent
    - MergeCommentEvent
    - PushEvent
    - PipelineEvent
    - TagEvent

## 0.1.0 - 2019-04-17

- Get all code from [mattermost-plugin-github](https://github.com/mattermost/mattermost-plugin-github/) and adapt it for gitlab
