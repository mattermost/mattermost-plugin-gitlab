// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';
import type {Page} from '@playwright/test';

import {uniquePath, type GitlabTodo} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {ensureConnected} from '../support/ui';

const EXPECTED_TODOS = 2;

test.describe('sidebar review and to-do lists', () => {
    let sandbox: Sandbox;

    test.beforeEach(async () => {
        sandbox = await createSandbox();
        await sandbox.user.markAllTodosDone();
    });

    test.afterEach(async () => {
        await sandbox?.user.markAllTodosDone();
        await destroySandbox(sandbox);
    });

    test('lists merge requests needing review and pending to-dos', async ({pw}) => {
        const {root, user, project, credentials} = sandbox;
        const review = await root.createMergeRequest(project.id, `E2E needs review ${uniquePath('')}`, {
            targetBranch: project.default_branch,
            reviewerIds: [credentials.e2e_user_id],
        });
        const issue = await root.createIssue(project.id, `E2E todo issue ${uniquePath('')}`, {assigneeIds: [credentials.e2e_user_id]});
        const todos = await waitForTodos(sandbox);

        const channelsPage = await ensureConnected(pw);
        const page = channelsPage.page;
        await refreshHeader(page);
        await expect(page.getByTestId('gitlab-header-reviews-count')).toHaveText('1', {timeout: 20000});
        await expect(page.getByTestId('gitlab-header-todos-count')).toHaveText(String(EXPECTED_TODOS), {timeout: 20000});

        await page.getByTestId('gitlab-header-reviews').click();
        await expect(page.getByTestId('gitlab-sidebar-heading')).toHaveText('Merge Requests Needing Review');
        await expect(page.getByTestId(`gitlab-sidebar-item-title-${review.id}`)).toHaveText(review.title);

        await page.getByTestId('gitlab-header-todos').click();
        await expect(page.getByTestId('gitlab-sidebar-heading')).toHaveText('To-Do List');
        await expect(page.getByTestId(`gitlab-sidebar-item-${todoFor(todos, issue.iid, 'assigned').id}`)).toContainText(issue.title);
        await expect(page.getByTestId(`gitlab-sidebar-item-${todoFor(todos, review.iid, 'review_requested').id}`)).toContainText(review.title);

        await user.markAllTodosDone();
        await refreshHeader(page);
        await expect(page.getByTestId('gitlab-header-todos-count')).toHaveText('0', {timeout: 20000});
        await expect(page.getByTestId('gitlab-sidebar-empty')).toBeVisible();
    });
});

async function refreshHeader(page: Page) {
    await page.getByTestId('gitlab-header-refresh').click();
}

async function waitForTodos(sandbox: Sandbox) {
    await expect.poll(async () => (await sandbox.user.listTodos()).length, {timeout: 30000}).toBe(EXPECTED_TODOS);
    return sandbox.user.listTodos();
}

function todoFor(todos: GitlabTodo[], iid: number, action: string) {
    const todo = todos.find((candidate) => candidate.target.iid === iid && candidate.action_name === action);
    if (!todo) {
        throw new Error(`No ${action} to-do for !${iid}: ${JSON.stringify(todos)}`);
    }
    return todo;
}
