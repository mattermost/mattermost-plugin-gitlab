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
| `pipelines` | `/gitlab pipelines run`, pipeline, job, and deployment posts on the runner, and the failed-pipeline DM |

### Writing tests

Specs follow the [Mattermost Playwright guidelines](https://github.com/mattermost/mattermost/blob/master/e2e-tests/playwright/CLAUDE.OPTIONAL.md):

- Every test has a JSDoc block with `@objective` (and `@precondition` when the setup is not the default), the `@gitlab_plugin` tag, and `// #` action and `// *` verification comments.
- Specs hold no selectors. Mattermost UI goes through `@mattermost/playwright-lib` page objects, and the plugin's UI through the components in `support/components/` (`GitlabRhs`, `GitlabSidebar`, `CreateIssueModal`, `AttachCommentModal`, `GitlabLinkTooltip`, `GitlabProfileAttribute`).
- Components locate elements by role, label, or text. Test ids are kept only for elements without an accessible name: the header counts, sidebar items, the connected username, subscription rows, and the link tooltip.

### CI

`.github/workflows/e2e.yml` runs the suite on every pull request except those that only change Markdown files or files under `docs/`, weekly on Mondays at 06:00 UTC, and on demand from the Actions tab. It boots the stack on the runner while the plugin builds, then runs the full suite. When a run fails, the Playwright report, traces, and container logs are uploaded as the `e2e-results-*` artifact.

`.github/workflows/e2e-upgrade.yml` runs the upgrade tests (below) daily at 04:00 UTC, on demand with a `from` input, and on pull requests that change the code reading stored data (`server/plugin.go`, `configuration.go`, `subscriptions.go`, `instance.go`, `utils.go`, `webhook.go`, `gitlab/user.go`, `subscription/`), `plugin.json`, the version tooling, or this suite. It is not a required check yet.

### Upgrade tests

Setting `E2E_UPGRADE_FROM` turns the run into an upgrade test: install an earlier release, create data through it, upgrade to the bundle in `dist/`, and check that the data still works. It catches stored data the new build can no longer read, tokens that no longer decrypt, subscriptions or webhooks that stop delivering, and settings lost in the upgrade. The specs live in `tests/upgrade/`, and a normal `npm test` never runs them.

```sh
make dist                              # from the repo root
cd e2e/playwright
E2E_UPGRADE_FROM=latest npm test       # or a release tag, e.g. E2E_UPGRADE_FROM=v2.0.0
```

**Which release.** `latest` is the newest published release with the same major version as the build and no newer than it. A customer upgrades along one release line, and a new major can raise `min_server_version`, so a release from another major is never picked. An explicit tag must meet the same rules, and its `min_server_version` must fit the test server. The bundle (the `linux-amd64` asset) is downloaded from GitHub into `results/upgrade/`. Set `GH_TOKEN` or log in with `gh` to avoid the unauthenticated rate limit.

**Which version the build has.** `make dist` takes the version from git tags: a tagged commit gets its tag, any other commit gets the latest reachable tag plus `+<hash>`. So on master the build is usually newer than the latest release (for example `2.0.0` → `2.1.0-rc1+7ee08ed`), and the run is a real version upgrade. Right after a final release is tagged, the build has the same version as that release with only a different hash. The run then still swaps new code in over the release's data, and the report annotates it as `relaxed` instead of `strict`.

The run is three Playwright projects, each depending on the one before, plus a cleanup:

| Project | Does |
| --- | --- |
| `upgrade-seed` | Installs the earlier release, configures it, and completes its install wizard. Connects the member through OAuth, subscribes Town Square to a new project with `issues,issue_comments,label:"upgrade"`, adds the webhook, and checks a labeled issue is posted. Writes what it created to `results/upgrade/upgrade-state.json`. |
| `upgrade-apply` | Uploads `dist/` over the release and restarts the plugin without touching its settings. Checks the running version is the `dist/` bundle's, in the same major and not older, and that the server logged no plugin errors during the upgrade (read from the `docker compose` logs, so skipped on other servers). |
| `upgrade-verify` | Checks the member is still connected and `/gitlab todo` works, so the stored token still decrypts. Checks the settings and the default instance carried over, the subscription and its features are still listed, and the webhook still delivers with the label filter applied. |
| `upgrade-cleanup` | The seed's teardown. Once the others finish, even after a failure, removes the Town Square subscription, the seeded project, and the state file. |

The seed creates everything through the plugin's own commands, OAuth flow, and wizard, never by writing to the KV store, so the stored data is in the earlier release's format. The plugin's KV store is not wiped first. For a clean slate, run `make e2e-env-down` and `make e2e-env-up`. The upgrade run reinstalls the plugin on the shared server, so do not run it while another run uses the same stack.

`OnActivate` runs no data migrations today, and the legacy `_gitlabtoken` migration predates every 2.x release, so there is no migrated data to check. If a change adds a migration, add a verify spec that checks the migrated data. Between v2.0.0 and master no stored data format has changed, so the run guards against future changes.

### Environment

| Variable | Default |
| --- | --- |
| `PW_BASE_URL` | `http://mattermost:8065` |
| `PW_ADMIN_USERNAME` / `PW_ADMIN_PASSWORD` | `sysadmin` / `Sys@dmin-sample1` |
| `PW_HEADLESS` | `true` |
| `PW_REGULAR_USER_PASSWORD` | `gitlab-e2e-user`'s password. Required unless `PW_BASE_URL` is `http://mattermost:8065` |
| `PW_DELETE_OTHER_TEAMS` | unset (only deletes other teams on `http://mattermost:8065`) |
| `E2E_UPGRADE_FROM` | unset. `latest` or a release tag runs the upgrade tests instead of the suite |
| `GH_TOKEN` / `GITHUB_TOKEN` | unset. Used to list releases and download the base bundle. Falls back to `gh auth token`, then to unauthenticated requests |
| `MM_IMAGE` (docker compose) | `mattermost/mattermost-enterprise-edition:12.0` |
| `GITLAB_IMAGE` (docker compose) | `gitlab/gitlab-ce:18.11.12-ce.0` |
| `GITLAB_RUNNER_IMAGE` (docker compose) | `gitlab/gitlab-runner:alpine-v18.11.4` |
| `GITLAB_ROOT_PASSWORD` | `Kv7-Qm4n-Wx2p-L9` |

GitLab rejects passwords that contain a predictable word or part of the account name, username, or email. The defaults above avoid that check. The GitLab user the suite connects is `gitlab-e2e`. Its password is printed into `.gitlab-credentials.json` by the bootstrap and is not a secret outside this local stack.
