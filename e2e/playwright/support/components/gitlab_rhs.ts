// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, type ChannelsPage} from '@mattermost/playwright-lib';
import type {Locator, Page} from '@playwright/test';

import {PLUGIN_ID} from '../constants';
import type {GitlabCredentials} from '../credentials';

import {GitlabSidebar} from './gitlab_sidebar';

// The subscriptions view the GitLab app-bar icon opens in the right-hand sidebar.
export class GitlabRhs {
    readonly container: Locator;
    readonly appBarIcon: Locator;
    readonly connectLink: Locator;
    readonly username: Locator;
    readonly noSubscriptions: Locator;

    constructor(private readonly channelsPage: ChannelsPage) {
        this.container = channelsPage.sidebarRight.container;

        // Mattermost renders the app-bar icon, so it only has the id Mattermost assigns.
        // The plugin id contains dots, so an #id selector would parse them as classes.
        this.appBarIcon = channelsPage.appBar.container.locator(`[id="app-bar-icon-${PLUGIN_ID}"]`);
        this.connectLink = this.container.getByRole('link', {name: 'Connect account', exact: true});
        this.username = this.container.getByTestId('gitlab-rhs-username');
        this.noSubscriptions = this.container.getByText('There are no GitLab subscriptions available in this channel.');
    }

    // Subscription rows have no role, so they keep a test id.
    subscription(path: string) {
        return this.container.getByTestId('gitlab-rhs-subscription').filter({hasText: path});
    }

    subscriptionFeature(path: string, feature: string) {
        return this.subscription(path).getByText(feature, {exact: true});
    }

    // The icon toggles the right-hand sidebar. When the sidebar already shows a
    // GitLab item list, close it first so the next click opens the subscriptions
    // view instead of hiding the sidebar.
    async open() {
        if (await this.isOpen()) {
            return;
        }
        const sidebar = new GitlabSidebar(this.channelsPage);
        if (await sidebar.isListOpen()) {
            await this.appBarIcon.click();
            await expect(this.container).toBeHidden();
        }
        await this.appBarIcon.click();
        await expect(this.username.or(this.connectLink)).toBeVisible();
    }

    async isOpen() {
        return await this.username.isVisible() || await this.connectLink.isVisible();
    }

    async isConnected() {
        return this.username.isVisible();
    }

    // Connects through the GitLab OAuth popup that "Connect account" opens.
    async connect(credentials: GitlabCredentials) {
        const popupPromise = this.channelsPage.page.waitForEvent('popup');
        await this.connectLink.click();
        await new GitlabSignInPopup(await popupPromise).signIn(credentials.e2e_username, credentials.e2e_password);
        await expect(this.username).toHaveText(credentials.e2e_username, {timeout: 30000});
    }
}

export class GitlabSignInPopup {
    readonly username: Locator;
    readonly password: Locator;
    readonly signInButton: Locator;
    readonly authorizeButton: Locator;

    constructor(readonly page: Page) {
        this.username = page.getByLabel('Username or primary email');
        this.password = page.getByLabel('Password', {exact: true});
        this.signInButton = page.getByRole('button', {name: 'Sign in', exact: true});
        this.authorizeButton = page.getByRole('button', {name: /^Authorize/});
    }

    async signIn(username: string, password: string) {
        await this.username.fill(username);
        await this.password.fill(password);
        await this.signInButton.click();

        // A trusted application skips the consent screen. Approve it if the app was not marked trusted.
        const needsConsent = await this.authorizeButton.waitFor({state: 'visible', timeout: 8000}).then(() => true).catch(() => false);
        if (needsConsent) {
            await this.authorizeButton.click();
        }
    }
}
