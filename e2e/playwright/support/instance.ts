// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './env';

import type {Client4} from '@mattermost/client';
import {ChannelsPage, TestBrowser, testConfig} from '@mattermost/playwright-lib';
import {chromium, expect, type Locator, type Page} from '@playwright/test';

import {loginAdmin, loginRegularUser} from './admin';
import {BOT_USERNAME, INSTANCE_NAME, PLUGIN_ID, TEAM_NAME, TOWN_SQUARE} from './constants';
import {loadCredentials} from './credentials';

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

function latestBotPost(page: Page): Locator {
    return page.locator('.post').filter({has: page.locator('.post__header', {hasText: BOT_USERNAME})}).last();
}

async function clickWizardButton(page: Page, name: string) {
    await latestBotPost(page).getByRole('button', {name, exact: true}).click();
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
        await page.goto(`${testConfig.baseURL}/${TEAM_NAME}/messages/@${BOT_USERNAME}`);
        await new ChannelsPage(page).toBeVisible();

        await expect(latestBotPost(page)).toContainText('Are you using gitlab.com?');
        await clickWizardButton(page, 'No');
        await submitDialog(page, {'Enter GitLab URL': credentials.gitlab_url});

        await expect(latestBotPost(page)).toContainText('Register an OAuth Application in GitLab');
        await clickWizardButton(page, 'Continue');

        await expect(latestBotPost(page)).toContainText('Application ID');
        await clickWizardButton(page, 'Continue');
        await submitDialog(page, {
            'Enter GitLab Instance name': INSTANCE_NAME,
            'Enter GitLab OAuth Application ID': credentials.client_id,
            'Enter GitLab OAuth Secret': credentials.client_secret,
        });

        await expect(latestBotPost(page)).toContainText('default GitLab instance');
        await clickWizardButton(page, 'Yes');
        await expect(latestBotPost(page)).toContainText('Connect your GitLab account');
    } finally {
        await browser.close();
    }

    const installed = await defaultInstanceName(client);
    if (installed !== INSTANCE_NAME) {
        throw new Error(`Expected default GitLab instance ${INSTANCE_NAME}, got ${String(installed)}`);
    }
}
