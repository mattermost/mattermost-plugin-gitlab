// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, type ChannelsPage} from '@mattermost/playwright-lib';
import type {Locator} from '@playwright/test';

const LISTS = {
    mergeRequests: {button: 'Merge requests assigned', count: 'gitlab-header-merge-requests-count', title: 'Merge Requests Assigned'},
    reviews: {button: 'Merge requests needing review', count: 'gitlab-header-reviews-count', title: 'Merge Requests Needing Review'},
    issues: {button: 'Issues', count: 'gitlab-header-issues-count', title: 'Issues'},
    todos: {button: 'To-Do list', count: 'gitlab-header-todos-count', title: 'To-Do List'},
};

export type GitlabList = keyof typeof LISTS;

// The GitLab buttons in the left sidebar header and the item list each one opens
// in the right-hand sidebar. The header only renders while the user is in a single team.
export class GitlabSidebar {
    readonly header: Locator;
    readonly list: Locator;
    readonly refreshButton: Locator;
    readonly emptyList: Locator;

    constructor(channelsPage: ChannelsPage) {
        this.header = channelsPage.sidebarLeft.container;
        this.list = channelsPage.sidebarRight.container;
        this.refreshButton = this.header.getByRole('link', {name: 'Refresh', exact: true});
        this.emptyList = this.list.getByText('You have no active items');
    }

    button(list: GitlabList) {
        return this.header.getByLabel(LISTS[list].button, {exact: true});
    }

    // The counts are bare numbers without an accessible name, so they keep test ids.
    count(list: GitlabList) {
        return this.header.getByTestId(LISTS[list].count);
    }

    heading(list: GitlabList) {
        return this.list.getByRole('link', {name: LISTS[list].title, exact: true});
    }

    item(id: number) {
        return this.list.getByTestId(`gitlab-sidebar-item-${id}`);
    }

    itemTitle(id: number) {
        return this.list.getByTestId(`gitlab-sidebar-item-title-${id}`);
    }

    async refresh() {
        await this.refreshButton.click();
    }

    async openList(list: GitlabList) {
        await this.button(list).click();
        await expect(this.heading(list)).toBeVisible();
    }

    async isListOpen() {
        for (const list of Object.keys(LISTS) as GitlabList[]) {
            // eslint-disable-next-line no-await-in-loop
            if (await this.heading(list).isVisible()) {
                return true;
            }
        }
        return false;
    }
}
