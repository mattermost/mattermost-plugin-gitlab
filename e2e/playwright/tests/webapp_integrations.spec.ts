// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';

import {GitlabLinkTooltip, GitlabProfileAttribute} from '../support/components';
import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {ensureConnected, runCommandExpecting} from '../support/ui';

test.describe('link tooltips and profile attributes', () => {
    let sandbox: Sandbox;

    test.beforeEach(async () => {
        sandbox = await createSandbox();
    });

    test.afterEach(async () => {
        await destroySandbox(sandbox);
    });

    /**
     * @objective Verify hovering GitLab issue and merge request links in a post shows the plugin's preview tooltip
     */
    test('previews GitLab issue and merge request links on hover', {tag: '@gitlab_plugin'}, async ({pw}) => {
        const {root, project} = sandbox;
        const issue = await root.createIssue(project.id, `E2E tooltip issue ${uniquePath('')}`);
        const mergeRequest = await root.createMergeRequest(project.id, `E2E tooltip MR ${uniquePath('')}`, {targetBranch: project.default_branch});
        const channelsPage = await ensureConnected(pw);

        // # Post and hover an issue link, then a merge request link
        // * Verify each tooltip shows the item's title and number
        // GitLab 18 reports issue URLs as /-/work_items/<iid>, which the tooltip does not
        // recognize, so the test posts the /-/issues/<iid> form that GitLab still serves.
        await expectTooltip(channelsPage, `${project.web_url}/-/issues/${issue.iid}`, issue.title, issue.iid);
        await expectTooltip(channelsPage, mergeRequest.web_url, mergeRequest.title, mergeRequest.iid);
    });

    /**
     * @objective Verify the user profile popover links to the connected GitLab account
     */
    test('shows the linked GitLab account in the user profile popover', {tag: '@gitlab_plugin'}, async ({pw}) => {
        const {credentials} = sandbox;
        const channelsPage = await ensureConnected(pw);

        // # Post a message after a bot reply, since consecutive posts by one user hide the avatar
        await runCommandExpecting(channelsPage, '/gitlab me', 'You are connected to GitLab as:');
        await channelsPage.centerView.postMessage(`Profile popover ${uniquePath('')}`);

        // # Open the profile popover from the post's avatar
        const post = await channelsPage.centerView.getLastPost();
        await post.profileIcon.click();
        await channelsPage.userProfilePopover.toBeVisible();

        // * Verify the popover links to the GitLab profile
        const link = new GitlabProfileAttribute(channelsPage).link(credentials.e2e_username);
        await expect(link).toHaveAttribute('href', `${credentials.gitlab_url}/${credentials.e2e_username}`);
    });
});

async function expectTooltip(channelsPage: ChannelsPage, url: string, title: string, iid: number) {
    const tooltip = new GitlabLinkTooltip(channelsPage);

    // # Post the link and hover it
    await channelsPage.centerView.postMessage(url);
    const post = await channelsPage.centerView.getLastPost();
    await tooltip.hoverLink(post, url);

    // * Verify the tooltip shows the title and number
    await expect(tooltip.title(title)).toBeVisible();
    await expect(tooltip.number(iid)).toBeVisible();

    // # Move the pointer away
    await channelsPage.page.mouse.move(0, 0);

    // * Verify the tooltip closes
    await expect(tooltip.container).toBeHidden();
}
