// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';

import {GitlabSidebar} from '../support/components';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import type {GitlabIssue, GitlabMergeRequest} from '../support/gitlab_api';
import {addWebhook, ensureConnected, expectPost, safeUnsubscribe, subscribe} from '../support/ui';

const COUNT_TIMEOUT = 20000;

test.describe('subscription notifications', () => {
    let sandbox: Sandbox | undefined;
    let channelsPage: ChannelsPage | undefined;
    let subscriptionPath = '';

    test.afterEach(async () => {
        await safeUnsubscribe(channelsPage, subscriptionPath);
        await destroySandbox(sandbox);
        sandbox = undefined;
        channelsPage = undefined;
    });

    /**
     * @objective Verify a project subscription posts project events and the sidebar lists only the items assigned to the connected user
     */
    test('posts project events and lists the connected user\'s issues and merge requests in the sidebar', {tag: '@gitlab_plugin'}, async ({pw}) => {
        sandbox = await createSandbox();
        const {root, project, credentials, path} = sandbox;
        subscriptionPath = path;

        // # Subscribe Town Square to a new project and add the plugin webhook
        channelsPage = await ensureConnected(pw);
        await subscribe(channelsPage, path);
        await addWebhook(channelsPage, path);

        // # Open an issue for the connected user, an issue for root, and a merge request for the connected user
        const mine = await root.createIssue(project.id, `E2E issue ${path}`, {assigneeIds: [credentials.e2e_user_id]});
        const others = await root.createIssue(project.id, `E2E root issue ${path}`, {assigneeIds: [credentials.root_user_id]});
        const mergeRequest = await root.createMergeRequest(project.id, `E2E merge request ${path}`, {
            targetBranch: project.default_branch,
            assigneeIds: [credentials.e2e_user_id],
        });

        // * Verify each item is posted in the channel
        await expectPost(channelsPage.page, mine.title);
        await expectPost(channelsPage.page, others.title);
        await expectPost(channelsPage.page, mergeRequest.title);

        // * Verify the sidebar lists only the connected user's items
        await expectSidebarLists(channelsPage, mine, others, mergeRequest);
    });

    /**
     * @objective Verify a group subscription posts events from a project in that group
     *
     * @precondition
     * GitLab Community Edition has no group webhooks, so the project in the group gets a project webhook
     */
    test('posts events from a project covered by a group subscription', {tag: '@gitlab_plugin'}, async ({pw}) => {
        sandbox = await createSandbox();
        const {root, project, credentials, group, path} = sandbox;
        subscriptionPath = group.path;

        // # Subscribe Town Square to the group and add a webhook to the project in it
        channelsPage = await ensureConnected(pw);
        await subscribe(channelsPage, group.path);
        await addWebhook(channelsPage, path);

        // # Open an issue for the connected user, an issue for root, and a merge request for the connected user
        const mine = await root.createIssue(project.id, `E2E group issue ${path}`, {assigneeIds: [credentials.e2e_user_id]});
        const others = await root.createIssue(project.id, `E2E group root issue ${path}`, {assigneeIds: [credentials.root_user_id]});
        const mergeRequest = await root.createMergeRequest(project.id, `E2E group merge request ${path}`, {
            targetBranch: project.default_branch,
            assigneeIds: [credentials.e2e_user_id],
        });

        // * Verify each item is posted in the channel
        await expectPost(channelsPage.page, mine.title);
        await expectPost(channelsPage.page, others.title);
        await expectPost(channelsPage.page, mergeRequest.title);

        // * Verify the sidebar lists only the connected user's items
        await expectSidebarLists(channelsPage, mine, others, mergeRequest);
    });
});

async function expectSidebarLists(channelsPage: ChannelsPage, mine: GitlabIssue, others: GitlabIssue, mergeRequest: GitlabMergeRequest) {
    const sidebar = new GitlabSidebar(channelsPage);

    // # Refresh the GitLab sidebar counts
    await sidebar.refresh();

    // * Verify one issue and one merge request are counted
    await expect(sidebar.count('issues')).toHaveText('1', {timeout: COUNT_TIMEOUT});
    await expect(sidebar.count('mergeRequests')).toHaveText('1', {timeout: COUNT_TIMEOUT});

    // # Open the issues list
    await sidebar.openList('issues');

    // * Verify it lists the connected user's issue and not root's
    await expect(sidebar.itemTitle(mine.id)).toHaveText(mine.title);
    await expect(sidebar.item(others.id)).toHaveCount(0);

    // # Open the assigned merge requests list
    await sidebar.openList('mergeRequests');

    // * Verify it lists the connected user's merge request
    await expect(sidebar.itemTitle(mergeRequest.id)).toHaveText(mergeRequest.title);
}
