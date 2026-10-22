// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {execFileSync} from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';

import './env';

import type {Client4} from '@mattermost/client';
import {testConfig} from '@mattermost/playwright-lib';

import {loginAdmin} from './admin';
import {ENCRYPTION_KEY, PLUGIN_ID, TEAM_NAME, WEBHOOK_SECRET} from './constants';
import {loadCredentials} from './credentials';
import {GitlabApi} from './gitlab_api';
import {installInstance} from './instance';

const DIST_DIR = path.resolve(__dirname, '../../../dist');
const E2E_DIR = path.resolve(__dirname, '..');
const DISPOSABLE_SERVER_URL = 'http://mattermost:8065';

type PluginSettingValues = Record<string, unknown>;

export function findPluginBundle(): string {
    const bundles = fs.existsSync(DIST_DIR) ? fs.readdirSync(DIST_DIR).filter((f) => f.startsWith(PLUGIN_ID) && f.endsWith('.tar.gz')) : [];
    if (bundles.length === 0) {
        throw new Error(`No plugin bundle found in ${DIST_DIR}. Run \`make dist\` from the repo root first.`);
    }

    return bundles.
        map((f) => path.join(DIST_DIR, f)).
        sort((a, b) => fs.statSync(b).mtimeMs - fs.statSync(a).mtimeMs)[0];
}

export async function installPlugin(client: Client4, bundlePath: string) {
    const bytes = fs.readFileSync(bundlePath);
    const file = new File([bytes], path.basename(bundlePath), {type: 'application/gzip'});
    await client.uploadPlugin(file, true);
}

export async function restartPlugin(client: Client4) {
    await client.disablePlugin(PLUGIN_ID);
    await client.enablePlugin(PLUGIN_ID);
    await waitForPlugin(client);
}

async function waitForPlugin(client: Client4) {
    const deadline = Date.now() + 20000;
    while (Date.now() < deadline) {
        const plugins = await client.getPlugins();
        if (plugins.active.some((plugin) => plugin.id === PLUGIN_ID)) {
            return;
        }
        // eslint-disable-next-line no-await-in-loop
        await new Promise((resolve) => setTimeout(resolve, 500));
    }
    throw new Error(`Timed out waiting for ${PLUGIN_ID} to become active`);
}

async function patchPluginSettings(client: Client4, patch: PluginSettingValues) {
    const config = await client.getConfig();
    const plugins = (config.PluginSettings.Plugins ?? {}) as Record<string, PluginSettingValues>;
    plugins[PLUGIN_ID] = {...(plugins[PLUGIN_ID] ?? {}), ...patch};
    config.PluginSettings.Plugins = plugins as typeof config.PluginSettings.Plugins;
    await client.updateConfig(config);
}

// OAuth for connecting accounts comes from the instance installed by
// installInstance. The legacy client id and secret are still required: the
// plugin's configuration check rejects authenticated commands without them, and
// its GitLab API client reads gitlaburl from these settings.
export async function configurePlugin(client: Client4) {
    const credentials = loadCredentials();
    await patchPluginSettings(client, {
        gitlaburl: credentials.gitlab_url,
        gitlaboauthclientid: credentials.client_id,
        gitlaboauthclientsecret: credentials.client_secret,
        encryptionkey: ENCRYPTION_KEY,
        webhooksecret: WEBHOOK_SECRET,
        enableprivaterepo: true,
        usepreregisteredapplication: false,
    });
}

export function bootstrapGitlab() {
    execFileSync(path.join(E2E_DIR, 'scripts/ensure-hosts.sh'), {stdio: 'inherit'});
    execFileSync(path.join(E2E_DIR, 'scripts/bootstrap-gitlab.sh'), {stdio: 'inherit'});
}

// ensureTeam keeps a single team so the channel-header GitLab buttons render.
// Those buttons are hidden once the user belongs to more than one team.
export async function ensureTeam(client: Client4) {
    const teams = await client.getMyTeams();
    if (!teams.some((team) => team.name === TEAM_NAME)) {
        await client.createTeam({
            name: TEAM_NAME,
            display_name: 'GitLab E2E',
            type: 'O',
        } as Parameters<Client4['createTeam']>[0]);
    }

    const others = (await client.getMyTeams()).filter((team) => team.name !== TEAM_NAME);
    if (others.length === 0) {
        return;
    }

    const isDisposableServer = new URL(testConfig.baseURL).origin === DISPOSABLE_SERVER_URL;
    if (!isDisposableServer && process.env.PW_DELETE_OTHER_TEAMS !== 'true') {
        throw new Error(`Refusing to delete teams ${others.map((team) => team.name).join(', ')} on ${testConfig.baseURL}. ` +
            'Leave them first, or set PW_DELETE_OTHER_TEAMS=true if this server is disposable.');
    }
    await Promise.all(others.map((team) => client.deleteTeam(team.id)));
}

async function resetGitlabState() {
    const credentials = loadCredentials();
    await new GitlabApi(credentials).closeItemsInvolving(credentials.e2e_user_id);
    await GitlabApi.asConnectedUser(credentials).markAllTodosDone();
}

export async function prepareServer() {
    const {client} = await loginAdmin();
    await ensureTeam(client);
    bootstrapGitlab();

    const bundle = findPluginBundle();
    // eslint-disable-next-line no-console
    console.log(`Installing ${bundle}`);
    await installPlugin(client, bundle);
    await configurePlugin(client);
    await restartPlugin(client);
    await installInstance();

    // Runs once the sysadmin is disconnected, so closing old items sends no DMs into the wizard channel.
    await resetGitlabState();
}
