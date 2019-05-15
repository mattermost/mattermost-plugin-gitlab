# Changelog

ALl releases can be found at https://github.com/mattermost/mattermost-plugin-gitlab/releases.

## 0.4.0 - unreleased

- Add webhook in gitlab when user subscribe to group or project [#15](https://github.com/manland/mattermost-plugin-gitlab/pull/15)

## 0.3.0 - 2019-06-05

- Give feedback to user when subscription has unknown feature [#47](https://github.com/manland/mattermost-plugin-gitlab/pull/47)
- Implement `OnConfigurationChange` to have configuration changes without reload [#22](https://github.com/manland/mattermost-plugin-gitlab/issues/22)
- Give feedback to user when deleting an unknown subscription [#48](https://github.com/manland/mattermost-plugin-gitlab/pull/48)
- **Fix** Reject private group subscription when `Enable Private repository` is `False` [#52](https://github.com/manland/mattermost-plugin-gitlab/pull/52)
- **Fix** `settings.notifications=off` was not implemented [#50](https://github.com/manland/mattermost-plugin-gitlab/pull/50)
- `/gitlab help` is accessible before user is logged [#49](https://github.com/manland/mattermost-plugin-gitlab/pull/49)
- Clean configuration for plugin [#51](https://github.com/manland/mattermost-plugin-gitlab/pull/51)
- **Breaking** Configuration `EnterpriseBaseURL` become `GitlabURL` with `https://gitlab.com` as de fault value [#34](https://github.com/manland/mattermost-plugin-gitlab/pull/34)
- Use a bot account [#9](https://github.com/manland/mattermost-plugin-gitlab/pull/9)
- Rewrite all `pull request` to `merge request`
- Display coverage with [codecov](https://codecov.io) [#37](https://github.com/manland/mattermost-plugin-gitlab/issues/37)
- Add screenshot in [README.md](https://github.com/manland/mattermost-plugin-gitlab/blob/master/README.md) [#26](https://github.com/manland/mattermost-plugin-gitlab/issues/26)
- Backport [mattermost-plugin-sample](https://github.com/mattermost/mattermost-plugin-sample/) infra code : use go mod, rework makefile to use sub-module, repair coverage [#27](https://github.com/manland/mattermost-plugin-gitlab/issues/27)
- From `Gitlab` to `GitLab` [#31](https://github.com/manland/mattermost-plugin-gitlab/issues/31)

## 0.2.0 - 2019-05-06

- Send refresh to webapp of the author of events received by webhook [#25](https://github.com/manland/mattermost-plugin-gitlab/pull/25)
- Add all webhook events for pipeline (run, fail, success) [#24](https://github.com/manland/mattermost-plugin-gitlab/pull/24)
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
