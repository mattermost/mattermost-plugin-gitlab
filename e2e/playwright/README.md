# GitLab plugin E2E tests

Playwright suite that drives the plugin against a disposable Mattermost server and a GitLab Community Edition container. GitLab users, the trusted OAuth application, and webhook settings are created with `gitlab-rails runner`. Groups, projects, issues, and merge requests are created by the specs through the GitLab API.

Built on [`@mattermost/playwright-lib`](https://www.npmjs.com/package/@mattermost/playwright-lib) (fixtures, page objects, and admin bootstrap), so no `mattermost` monorepo checkout is needed.

## Host names

The browser and both containers share two hostnames. Add this line to `/etc/hosts` once:

```text
127.0.0.1 mattermost gitlab.test
```

`scripts/ensure-hosts.sh` checks that both names resolve and prints the line when they do not. It does not edit `/etc/hosts`.

The suite needs Node.js 20 or newer. GitLab needs about 4 GB of RAM. The first boot takes several minutes.

## Running

From the repo root:

```sh
make e2e-env-up    # Mattermost 12.0, Postgres, GitLab CE, and a GitLab runner
make e2e              # build the plugin bundle, install deps, run the suite
make e2e-env-down  # stop and wipe both servers
```

Or from this directory, after `make dist` and `make e2e-env-up`:

```sh
npm ci && npx playwright install chromium
npm test              # headless
npm run test:headed   # watch the browser
npm run test:ui       # Playwright UI mode
npm run show-report   # open the last HTML report
```

`global_setup.ts` creates the sysadmin, a single `gitlab-e2e` team (a second team would move the GitLab buttons out of the channel header), and a regular member, `gitlab-e2e-user`, that the specs act as. Only the install wizard and the `/gitlab instance` checks use the sysadmin. Setup then bootstraps GitLab, uploads the newest `dist/*.tar.gz`, and writes the GitLab URL and OAuth credentials into the plugin config. It then runs `/gitlab instance install` and completes the wizard in the GitLab bot DM with the same OAuth application, so `gitlab-e2e` becomes the default instance. The suite is safe to rerun against the same server: the bootstrap recreates the OAuth application and the runner registration, and setup disconnects both Mattermost users and uninstalls the old instance before installing it again. It also closes any issues and merge requests still assigned to or awaiting review from `gitlab-e2e`, and clears its to-dos, so an interrupted run does not skew the sidebar counts. The first test then reconnects through the OAuth popup.

Setup deletes every other team the sysadmin belongs to. It only does so on `http://mattermost:8065` unless `PW_DELETE_OTHER_TEAMS=true` is set.

The bootstrap also creates a personal access token for `gitlab-e2e`. Specs use it to act as the connected user, for example to approve merge requests or clear to-dos. Root performs most GitLab actions so that the connected user receives the DMs.

GitLab Community Edition has no group webhooks, so the group-subscription test adds its webhook on the project inside the group.

### Coverage

| Spec | Covers |
| --- | --- |
| `connect` | OAuth connect and disconnect |
| `commands` | `help`, `about`, `me`, `instance list`, `connect`, `settings`, `todo`, and usage errors |
| `subscriptions` | Add, list, delete, custom features, validation, permissions, `webhook list` |
| `channel_events` | Issue, merge request, comment, push, tag, and release posts |
| `feature_filters` | Label filters, feature exclusion, confidential issues |
| `dm_notifications` | Assignment, comment, mention, review, approval, merge DMs, and muting with `settings` |
| `notifications` | Project and group subscriptions, plus the assigned issue and merge request lists |
| `sidebar` | Review and to-do lists and counts |
| `post_actions` | Create GitLab issue and attach to issue from the post menu, and `/gitlab issue create` |
| `webapp_integrations` | Link tooltips and the profile popover attribute |
| `pipelines` | `/gitlab pipelines run`, plus pipeline, job, and deployment posts on the runner |

`pipelines` includes a `fixme` test for the failed-pipeline DM. The plugin does not deliver that DM to the pipeline author today.

### CI

`.github/workflows/e2e.yml` runs the suite on every pull request except those that only change Markdown files or files under `docs/`, weekly on Mondays at 06:00 UTC, and on demand from the Actions tab. It boots the stack on the runner while the plugin builds, then runs the full suite. When a run fails, the Playwright report, traces, and container logs are uploaded as the `e2e-results-*` artifact.

### Environment

| Variable | Default |
| --- | --- |
| `PW_BASE_URL` | `http://mattermost:8065` |
| `PW_ADMIN_USERNAME` / `PW_ADMIN_PASSWORD` | `sysadmin` / `Sys@dmin-sample1` |
| `PW_HEADLESS` | `true` |
| `PW_REGULAR_USER_PASSWORD` | `gitlab-e2e-user`'s password. Required unless `PW_BASE_URL` is `http://mattermost:8065` |
| `PW_DELETE_OTHER_TEAMS` | unset (only deletes other teams on `http://mattermost:8065`) |
| `MM_IMAGE` (docker compose) | `mattermost/mattermost-enterprise-edition:12.0` |
| `GITLAB_IMAGE` (docker compose) | `gitlab/gitlab-ce:18.11.12-ce.0` |
| `GITLAB_RUNNER_IMAGE` (docker compose) | `gitlab/gitlab-runner:alpine-v18.11.4` |
| `GITLAB_ROOT_PASSWORD` | `Kv7-Qm4n-Wx2p-L9` |

GitLab rejects passwords that contain a predictable word or part of the account name, username, or email. The defaults above avoid that check. The GitLab user the suite connects is `gitlab-e2e`. Its password is printed into `.gitlab-credentials.json` by the bootstrap and is not a secret outside this local stack.
