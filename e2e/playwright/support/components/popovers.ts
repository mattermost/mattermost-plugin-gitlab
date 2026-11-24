// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {ChannelsPage, ChannelsPost} from '@mattermost/playwright-lib';
import type {Locator} from '@playwright/test';

// The preview the plugin shows while hovering a GitLab issue or merge request link.
export class GitlabLinkTooltip {
    // The tooltip is a plain box without a role, so it keeps a test id.
    readonly container: Locator;

    constructor(channelsPage: ChannelsPage) {
        this.container = channelsPage.page.getByTestId('gitlab-link-tooltip');
    }

    title(title: string) {
        return this.container.getByRole('heading', {name: title, exact: true});
    }

    number(iid: number) {
        return this.container.getByText(`#${iid}`, {exact: true});
    }

    async hoverLink(post: ChannelsPost, url: string) {
        await post.container.getByRole('link', {name: url, exact: true}).hover();
    }
}

// The GitLab account link the plugin adds to Mattermost's user profile popover.
export class GitlabProfileAttribute {
    readonly popover: Locator;

    constructor(channelsPage: ChannelsPage) {
        this.popover = channelsPage.userProfilePopover.container;
    }

    // The accessible name is prefixed with the GitLab icon glyph.
    link(gitlabUsername: string) {
        const escaped = gitlabUsername.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        return this.popover.getByRole('link', {name: new RegExp(`(^|\\s)${escaped}$`)});
    }
}
