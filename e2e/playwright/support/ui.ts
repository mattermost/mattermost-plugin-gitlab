// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {ChannelsPage, ChannelsPost, expect, testConfig, type PlaywrightExtended} from '@mattermost/playwright-lib';
import type {Page} from '@playwright/test';

import {loginAdmin, loginRegularUser} from './admin';
import {GitlabRhs} from './components';
import {BOT_USERNAME, TEAM_NAME, TOWN_SQUARE, WEBHOOK_TIMEOUT} from './constants';
import {loadCredentials} from './credentials';

// Specs act as a regular member. Use {asAdmin: true} only for sysadmin-gated commands.
export async function openTownSquare(pw: PlaywrightExtended, {asAdmin = false} = {}) {
    const {user} = asAdmin ? await loginAdmin() : await loginRegularUser();
    const {channelsPage} = await pw.testBrowser.login(user);
    await channelsPage.goto(TEAM_NAME, TOWN_SQUARE);
    await channelsPage.toBeVisible();
    return channelsPage;
}

export async function runSlashCommand(channelsPage: ChannelsPage, command: string) {
    const postCreate = channelsPage.centerView.postCreate;
    await postCreate.input.fill(command);

    // Pressing Enter would pick a slash-command autocomplete suggestion.
    await postCreate.sendMessageButton.click();
    await expect(postCreate.input).toHaveValue('');
}

function postsWithText(page: Page, ...texts: Array<string | RegExp>) {
    let posts = new ChannelsPage(page).centerView.container.getByTestId('postView');
    for (const text of texts) {
        posts = posts.filter({hasText: text});
    }
    return posts;
}

// The newest center-channel post containing every given fragment.
export function postWithText(page: Page, ...texts: Array<string | RegExp>) {
    return new ChannelsPost(postsWithText(page, ...texts).last());
}

export async function expectPostWithin(page: Page, timeout: number, ...texts: Array<string | RegExp>) {
    await expect(postWithText(page, ...texts).container).toBeVisible({timeout});
}

// Webhook deliveries are asynchronous, so allow time for GitLab to send them.
export async function expectPost(page: Page, ...texts: Array<string | RegExp>) {
    await expectPostWithin(page, WEBHOOK_TIMEOUT, ...texts);
}

export async function expectNoPost(page: Page, ...texts: Array<string | RegExp>) {
    await expect(postsWithText(page, ...texts)).toHaveCount(0);
}

// Earlier commands, and webhook posts landing in between, can leave matching posts
// on screen, so wait for one more match than before the command ran.
export async function runCommandExpecting(channelsPage: ChannelsPage, command: string, reply: string | RegExp) {
    const replies = postsWithText(channelsPage.page, reply);
    const before = await replies.count();
    await runSlashCommand(channelsPage, command);
    await expect(replies).toHaveCount(before + 1);
    return new ChannelsPost(replies.last());
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

export async function openBotDirectMessage(channelsPage: ChannelsPage) {
    await channelsPage.page.goto(`${testConfig.baseURL}/${TEAM_NAME}/messages/@${BOT_USERNAME}`);
    await channelsPage.toBeVisible();
}

// Plugin post actions are listed under the post's "Message actions" button.
export async function openPostMenuAction(channelsPage: ChannelsPage, post: ChannelsPost, action: string) {
    await post.hover();
    await post.postMenu.actionsButton.click();
    await channelsPage.page.getByRole('menuitem', {name: action}).click();
}

export async function disconnectAccount(channelsPage: ChannelsPage) {
    await runCommandExpecting(channelsPage, '/gitlab disconnect', 'Disconnected your GitLab account.');
    await expect(new GitlabRhs(channelsPage).connectLink).toBeVisible();
}

export async function ensureConnected(pw: PlaywrightExtended) {
    const channelsPage = await openTownSquare(pw);
    const rhs = new GitlabRhs(channelsPage);
    await rhs.open();
    const credentials = loadCredentials();
    if (!await rhs.isConnected()) {
        await rhs.connect(credentials);
    }
    await expect(rhs.username).toHaveText(credentials.e2e_username);
    return channelsPage;
}
