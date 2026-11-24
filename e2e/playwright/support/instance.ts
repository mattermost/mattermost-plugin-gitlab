// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './env';

import type {Client4} from '@mattermost/client';
import {ChannelsPage, TestBrowser, testConfig} from '@mattermost/playwright-lib';
import {chromium, expect, type Page} from '@playwright/test';

import {loginAdmin, loginRegularUser} from './admin';
import {BOT_USERNAME, INSTANCE_NAME, PLUGIN_ID, TEAM_NAME, TOWN_SQUARE} from './constants';
import {loadCredentials} from './credentials';

const WIZARD_STEP_TIMEOUT = 15000;

async function executeCommand(client: Client4, command: string) {
    const team = await client.getTeamByName(TEAM_NAME);
    const channel = await client.getChannelByName(team.id, TOWN_SQUARE);
    await client.executeCommand(command, {channel_id: channel.id, team_id: team.id});
}

async function defaultInstanceName(client: Client4) {
    const config = await client.getConfig();
    const settings = config.PluginSettings.Plugins?.[PLUGIN_ID] as Record<string, unknown> | undefined;
    return settings?.defaultinstancename;
}

// Earlier runs leave identical wizard prompts in the DM, so each step waits for the newest post.
async function expectWizardStep(channelsPage: ChannelsPage, text: string) {
    await channelsPage.centerView.waitUntilLastPostContains(text, WIZARD_STEP_TIMEOUT);
}

async function clickWizardButton(channelsPage: ChannelsPage, text: string, name: string) {
    await expectWizardStep(channelsPage, text);
    const post = await channelsPage.centerView.getLastPost();
    await post.container.getByRole('button', {name, exact: true}).click();
}

async function submitDialog(page: Page, fields: Record<string, string>) {
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    for (const [placeholder, value] of Object.entries(fields)) {
        // eslint-disable-next-line no-await-in-loop
        await dialog.getByPlaceholder(placeholder).fill(value);
    }
    await dialog.getByRole('button', {name: 'Save & continue'}).click();
    await expect(dialog).toBeHidden();
}

// installInstance walks the `/gitlab instance install` wizard in the GitLab bot DM
// with the OAuth application created by the GitLab bootstrap. The bootstrap
// recreates that application on every run, so an existing instance is removed first
// to keep its client secret current.
export async function installInstance() {
    const credentials = loadCredentials();
    const {client, user} = await loginAdmin();

    // The bootstrap recreates the OAuth application, which revokes tokens from an earlier connection.
    await executeCommand(client, '/gitlab disconnect');
    await executeCommand((await loginRegularUser()).client, '/gitlab disconnect');
    await executeCommand(client, `/gitlab instance uninstall ${INSTANCE_NAME}`);
    await executeCommand(client, '/gitlab instance install');

    const browser = await chromium.launch({headless: testConfig.headless});
    try {
        const {page} = await new TestBrowser(browser).login(user);
        const channelsPage = new ChannelsPage(page);
        await page.goto(`${testConfig.baseURL}/${TEAM_NAME}/messages/@${BOT_USERNAME}`);
        await channelsPage.toBeVisible();

        await clickWizardButton(channelsPage, 'Are you using gitlab.com?', 'No');
        await submitDialog(page, {'Enter GitLab URL': credentials.gitlab_url});

        await clickWizardButton(channelsPage, 'Register an OAuth Application in GitLab', 'Continue');

        await clickWizardButton(channelsPage, 'Application ID', 'Continue');
        await submitDialog(page, {
            'Enter GitLab Instance name': INSTANCE_NAME,
            'Enter GitLab OAuth Application ID': credentials.client_id,
            'Enter GitLab OAuth Secret': credentials.client_secret,
        });

        await clickWizardButton(channelsPage, 'default GitLab instance', 'Yes');
        await expectWizardStep(channelsPage, 'Connect your GitLab account');
    } finally {
        await browser.close();
    }

    const installed = await defaultInstanceName(client);
    if (installed !== INSTANCE_NAME) {
        throw new Error(`Expected default GitLab instance ${INSTANCE_NAME}, got ${String(installed)}`);
    }
}
