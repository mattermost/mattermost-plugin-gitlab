// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {GitlabSidebar} from '../support/components';
import {uniquePath, type GitlabTodo} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {ensureConnected} from '../support/ui';

const EXPECTED_TODOS = 2;
const COUNT_TIMEOUT = 20000;

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

    /**
     * @objective Verify the sidebar counts and lists merge requests needing review and pending to-dos, and clears them once done
     *
     * @precondition
     * The connected GitLab user starts with no pending to-dos
     */
    test('lists merge requests needing review and pending to-dos in the GitLab sidebar', {tag: '@gitlab_plugin'}, async ({pw}) => {
        const {root, user, project, credentials} = sandbox;

        // # Request the connected user's review on a merge request and assign them an issue
        const review = await root.createMergeRequest(project.id, `E2E needs review ${uniquePath('')}`, {
            targetBranch: project.default_branch,
            reviewerIds: [credentials.e2e_user_id],
        });
        const issue = await root.createIssue(project.id, `E2E todo issue ${uniquePath('')}`, {assigneeIds: [credentials.e2e_user_id]});
        const todos = await waitForTodos(sandbox);

        // # Log in and refresh the GitLab sidebar counts
        const channelsPage = await ensureConnected(pw);
        const sidebar = new GitlabSidebar(channelsPage);
        await sidebar.refresh();

        // * Verify one review and both to-dos are counted
        await expect(sidebar.count('reviews')).toHaveText('1', {timeout: COUNT_TIMEOUT});
        await expect(sidebar.count('todos')).toHaveText(String(EXPECTED_TODOS), {timeout: COUNT_TIMEOUT});

        // # Open the merge requests needing review list
        await sidebar.openList('reviews');

        // * Verify it lists the merge request
        await expect(sidebar.itemTitle(review.id)).toHaveText(review.title);

        // # Open the to-do list
        await sidebar.openList('todos');

        // * Verify it lists the issue assignment and the review request
        await expect(sidebar.item(todoFor(todos, issue.iid, 'assigned').id)).toContainText(issue.title);
        await expect(sidebar.item(todoFor(todos, review.iid, 'review_requested').id)).toContainText(review.title);

        // # Mark every to-do done in GitLab and refresh the counts
        await user.markAllTodosDone();
        await sidebar.refresh();

        // * Verify the to-do count and list are empty
        await expect(sidebar.count('todos')).toHaveText('0', {timeout: COUNT_TIMEOUT});
        await expect(sidebar.emptyList).toBeVisible();
    });
});

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
