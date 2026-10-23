// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, testConfig, type ChannelsPage, type PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Page} from '@playwright/test';

import {loginAdmin, loginRegularUser} from './admin';
import {APP_BAR_ICON, BOT_USERNAME, TEAM_NAME, TOWN_SQUARE, WEBHOOK_TIMEOUT} from './constants';
import {loadCredentials} from './credentials';

// Specs act as a regular member. Use {asAdmin: true} only for sysadmin-gated commands.
export async function openTownSquare(pw: PlaywrightExtended, {asAdmin = false} = {}) {
    const {user} = asAdmin ? await loginAdmin() : await loginRegularUser();
    const {channelsPage} = await pw.testBrowser.login(user);
    await channelsPage.goto(TEAM_NAME, TOWN_SQUARE);
    await channelsPage.toBeVisible();
    return channelsPage;
}

// Mattermost renders the app-bar icon, so it has no data-testid from this plugin.
// The icon toggles the right-hand sidebar. When the sidebar is already open on
// the issue or merge-request list, close it first so the next click opens the
// subscriptions view instead of hiding the sidebar.
export async function openSubscriptionsSidebar(page: Page) {
    const username = page.getByTestId('gitlab-rhs-username');
    const connect = page.getByTestId('gitlab-rhs-connect');
    const listHeading = page.getByTestId('gitlab-sidebar-heading');
    if (await username.isVisible().catch(() => false) || await connect.isVisible().catch(() => false)) {
        return;
    }
    if (await listHeading.isVisible().catch(() => false)) {
        await page.locator(APP_BAR_ICON).click();
        await expect(listHeading).toBeHidden();
    }
    await page.locator(APP_BAR_ICON).click();
    await expect(username.or(connect)).toBeVisible();
}

export async function runSlashCommand(channelsPage: ChannelsPage, command: string) {
    const postCreate = channelsPage.centerView.postCreate;
    await postCreate.input.fill(command);

    // Pressing Enter would pick a slash-command autocomplete suggestion.
    await postCreate.sendMessageButton.click();
    await expect(postCreate.input).toHaveValue('');
}

// Matches the newest post containing every given fragment.
export function postWithText(page: Page, ...texts: Array<string | RegExp>) {
    let posts = page.locator('.post-message__text');
    for (const text of texts) {
        posts = posts.filter({hasText: text});
    }
    return posts.last();
}

// Webhook deliveries are asynchronous, so allow time for GitLab to send them.
export async function expectPost(page: Page, ...texts: Array<string | RegExp>) {
    await expect(postWithText(page, ...texts)).toBeVisible({timeout: WEBHOOK_TIMEOUT});
}

// Earlier commands can leave an identical reply on screen, so wait for one more match.
export async function runCommandExpecting(channelsPage: ChannelsPage, command: string, reply: string | RegExp) {
    const replies = channelsPage.page.locator('.post-message__text').filter({hasText: reply});
    const before = await replies.count();
    await runSlashCommand(channelsPage, command);
    await expect(replies).toHaveCount(before + 1);
    return replies.last();
}

export async function subscribe(channelsPage: ChannelsPage, path: string, features?: string) {
    const command = features ? `/gitlab subscriptions add ${path} ${features}` : `/gitlab subscriptions add ${path}`;
    await runCommandExpecting(channelsPage, command, `Successfully subscribed to ${path}`);
}

export async function addWebhook(channelsPage: ChannelsPage, path: string) {
    await runCommandExpecting(channelsPage, `/gitlab webhook add ${path}`, 'Webhook Created');
}

export async function safeUnsubscribe(channelsPage: ChannelsPage | undefined, path: string | undefined) {
    if (!channelsPage || !path) {
        return;
    }
    try {
        await runSlashCommand(channelsPage, `/gitlab subscriptions delete ${path}`);
    } catch {
        // The subscription may already have been removed, or the composer may be unavailable.
    }
}

export async function openBotDirectMessage(page: Page) {
    await page.goto(`${testConfig.baseURL}/${TEAM_NAME}/messages/@${BOT_USERNAME}`);
    await expect(page.locator('#post_textbox')).toBeVisible();
}

export async function connectAccount(page: Page) {
    const credentials = loadCredentials();
    const popupPromise = page.waitForEvent('popup');
    await page.getByTestId('gitlab-rhs-connect').click();
    const popup = await popupPromise;

    await popup.locator('#user_login').fill(credentials.e2e_username);
    await popup.locator('#user_password').fill(credentials.e2e_password);
    await popup.locator('input[name="commit"], button[type="submit"]').first().click();

    // A trusted application skips this. Click it if the app was not marked trusted.
    const authorize = popup.getByRole('button', {name: /^Authorize/});
    const needsConsent = await authorize.waitFor({state: 'visible', timeout: 8000}).then(() => true).catch(() => false);
    if (needsConsent) {
        await authorize.click();
    }

    await expect(page.getByTestId('gitlab-rhs-username')).toHaveText(credentials.e2e_username, {timeout: 30000});
}

export async function disconnectAccount(channelsPage: ChannelsPage) {
    await runSlashCommand(channelsPage, '/gitlab disconnect');
    await expect(postWithText(channelsPage.page, 'Disconnected your GitLab account.')).toBeVisible();
    await expect(channelsPage.page.getByTestId('gitlab-rhs-connect')).toBeVisible();
}

export async function ensureConnected(pw: PlaywrightExtended) {
    const channelsPage = await openTownSquare(pw);
    await openSubscriptionsSidebar(channelsPage.page);
    if (await channelsPage.page.getByTestId('gitlab-rhs-connect').isVisible().catch(() => false)) {
        await connectAccount(channelsPage.page);
    }
    const credentials = loadCredentials();
    await expect(channelsPage.page.getByTestId('gitlab-rhs-username')).toHaveText(credentials.e2e_username);
    return channelsPage;
}
