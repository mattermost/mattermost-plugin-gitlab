// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';
import type {Page} from '@playwright/test';

import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import type {GitlabIssue, GitlabMergeRequest} from '../support/gitlab_api';
import {addWebhook, ensureConnected, expectPost, safeUnsubscribe, subscribe} from '../support/ui';

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

    test('posts project events and lists items assigned to the connected user', async ({pw}) => {
        sandbox = await createSandbox();
        const {root, project, credentials, path} = sandbox;
        subscriptionPath = path;

        channelsPage = await ensureConnected(pw);
        await subscribe(channelsPage, path);
        await addWebhook(channelsPage, path);

        const mine = await root.createIssue(project.id, `E2E issue ${path}`, {assigneeIds: [credentials.e2e_user_id]});
        const others = await root.createIssue(project.id, `E2E root issue ${path}`, {assigneeIds: [credentials.root_user_id]});
        const mergeRequest = await root.createMergeRequest(project.id, `E2E merge request ${path}`, {
            targetBranch: project.default_branch,
            assigneeIds: [credentials.e2e_user_id],
        });

        await expectPost(channelsPage.page, mine.title);
        await expectPost(channelsPage.page, others.title);
        await expectPost(channelsPage.page, mergeRequest.title);

        await expectSidebarLists(channelsPage.page, mine, others, mergeRequest);
    });

    test('posts events from a project covered by a group subscription', async ({pw}) => {
        sandbox = await createSandbox();
        const {root, project, credentials, group, path} = sandbox;
        subscriptionPath = group.path;

        channelsPage = await ensureConnected(pw);
        await subscribe(channelsPage, group.path);

        // Group webhooks are a GitLab Premium feature, so Community Edition needs a
        // project webhook. The group subscription still matches the project's namespace.
        await addWebhook(channelsPage, path);

        const mine = await root.createIssue(project.id, `E2E group issue ${path}`, {assigneeIds: [credentials.e2e_user_id]});
        const others = await root.createIssue(project.id, `E2E group root issue ${path}`, {assigneeIds: [credentials.root_user_id]});
        const mergeRequest = await root.createMergeRequest(project.id, `E2E group merge request ${path}`, {
            targetBranch: project.default_branch,
            assigneeIds: [credentials.e2e_user_id],
        });

        await expectPost(channelsPage.page, mine.title);
        await expectPost(channelsPage.page, others.title);
        await expectPost(channelsPage.page, mergeRequest.title);

        await expectSidebarLists(channelsPage.page, mine, others, mergeRequest);
    });
});

async function expectSidebarLists(page: Page, mine: GitlabIssue, others: GitlabIssue, mergeRequest: GitlabMergeRequest) {
    await page.getByTestId('gitlab-header-refresh').click();
    await expect(page.getByTestId('gitlab-header-issues-count')).toHaveText('1', {timeout: 20000});
    await expect(page.getByTestId('gitlab-header-merge-requests-count')).toHaveText('1', {timeout: 20000});

    await page.getByTestId('gitlab-header-issues').click();
    await expect(page.getByTestId('gitlab-sidebar-heading')).toHaveText('Issues');
    await expect(page.getByTestId(`gitlab-sidebar-item-title-${mine.id}`)).toHaveText(mine.title);
    await expect(page.getByTestId(`gitlab-sidebar-item-${others.id}`)).toHaveCount(0);

    await page.getByTestId('gitlab-header-merge-requests').click();
    await expect(page.getByTestId('gitlab-sidebar-heading')).toHaveText('Merge Requests Assigned');
    await expect(page.getByTestId(`gitlab-sidebar-item-title-${mergeRequest.id}`)).toHaveText(mergeRequest.title);
}
