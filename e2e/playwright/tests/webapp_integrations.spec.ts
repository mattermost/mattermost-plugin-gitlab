// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';

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

    test('previews GitLab issue and merge request links', async ({pw}) => {
        const {root, project} = sandbox;
        const issue = await root.createIssue(project.id, `E2E tooltip issue ${uniquePath('')}`);
        const mergeRequest = await root.createMergeRequest(project.id, `E2E tooltip MR ${uniquePath('')}`, {targetBranch: project.default_branch});
        const channelsPage = await ensureConnected(pw);

        // GitLab 18 reports issue URLs as /-/work_items/<iid>, which the tooltip does not
        // recognize, so the test posts the /-/issues/<iid> form that GitLab still serves.
        await expectTooltip(channelsPage, `${project.web_url}/-/issues/${issue.iid}`, issue.title, issue.iid);
        await expectTooltip(channelsPage, mergeRequest.web_url, mergeRequest.title, mergeRequest.iid);
    });

    test('shows the linked GitLab account in the profile popover', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);

        // Consecutive posts by one user hide the avatar, so a bot reply goes in between.
        await runCommandExpecting(channelsPage, '/gitlab me', 'You are connected to GitLab as:');
        await channelsPage.centerView.postMessage(`Profile popover ${uniquePath('')}`);
        const post = await channelsPage.centerView.getLastPost();
        await post.profileIcon.click();

        const attribute = channelsPage.page.getByTestId('gitlab-user-attribute');
        await expect(attribute).toHaveText(sandbox.credentials.e2e_username);
        await expect(attribute).toHaveAttribute('href', `${sandbox.credentials.gitlab_url}/${sandbox.credentials.e2e_username}`);
    });
});

async function expectTooltip(channelsPage: ChannelsPage, url: string, title: string, iid: number) {
    await channelsPage.centerView.postMessage(url);
    const post = await channelsPage.centerView.getLastPost();
    await post.container.locator(`a[href="${url}"]`).hover();

    const tooltip = channelsPage.page.getByTestId('gitlab-link-tooltip');
    await expect(tooltip.getByTestId('gitlab-link-tooltip-title')).toHaveText(title);
    await expect(tooltip.getByTestId('gitlab-link-tooltip-number')).toHaveText(`#${iid}`);

    await channelsPage.page.mouse.move(0, 0);
    await expect(tooltip).toBeHidden();
}
