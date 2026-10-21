// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';
import type {Locator} from '@playwright/test';

import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {ensureConnected, runSlashCommand} from '../support/ui';

test.describe('post menu actions and the issue modal', () => {
    let sandbox: Sandbox;

    test.beforeEach(async () => {
        sandbox = await createSandbox();
    });

    test.afterEach(async () => {
        await destroySandbox(sandbox);
    });

    test('creates a GitLab issue from a post', async ({pw}) => {
        const {root, project, path} = sandbox;
        const channelsPage = await ensureConnected(pw);
        const message = `Create an issue from this post ${uniquePath('')}`;
        const title = `E2E issue from post ${uniquePath('')}`;

        // With more than 100 projects the selector filters whatever list it has when typing
        // starts, so the project list must finish loading first.
        const projectsLoaded = channelsPage.page.waitForResponse((response) => response.url().endsWith('/api/v1/projects') && response.ok());
        await openPostMenuAction(channelsPage, message, 'gitlab-post-menu-create-issue');
        const form = channelsPage.page.getByTestId('gitlab-create-issue-form');
        await expect(form.getByTestId('gitlab-create-issue-description')).toHaveValue(message);
        await projectsLoaded;

        await selectOption(channelsPage, form.getByTestId('gitlab-create-issue-project'), path);
        await form.getByTestId('gitlab-create-issue-title').fill(title);
        await form.getByTestId('gitlab-create-issue-submit').click();
        await expect(form).toBeHidden();

        await expect.poll(async () => {
            const issue = (await root.listIssues(project.id)).find((candidate) => candidate.title === title);
            return issue?.description ?? '';
        }, {timeout: 20000}).toContain(message);
    });

    test('attaches a post to an existing GitLab issue', async ({pw}) => {
        const {root, project} = sandbox;
        const issue = await root.createIssue(project.id, `E2E attach target ${uniquePath('')}`);
        const channelsPage = await ensureConnected(pw);
        const message = `Attach this post ${uniquePath('')}`;

        await openPostMenuAction(channelsPage, message, 'gitlab-post-menu-attach-comment');
        const form = channelsPage.page.getByTestId('gitlab-attach-comment-form');
        await expect(form.getByTestId('gitlab-attach-comment-message')).toHaveValue(message);

        await selectOption(channelsPage, form.getByTestId('gitlab-attach-comment-issue'), issue.title);
        await form.getByTestId('gitlab-attach-comment-submit').click();
        await expect(form).toBeHidden();

        await expect.poll(async () => {
            const comments = await root.listIssueComments(project.id, issue.iid);
            return comments.filter((comment) => !comment.system).map((comment) => comment.body).join('\n');
        }, {timeout: 20000}).toContain(message);
    });

    test('opens the issue modal with the title from /gitlab issue create', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);
        const title = `E2E modal title ${uniquePath('')}`;

        await runSlashCommand(channelsPage, `/gitlab issue create ${title}`);
        const form = channelsPage.page.getByTestId('gitlab-create-issue-form');
        await expect(form.getByTestId('gitlab-create-issue-title')).toHaveValue(title);

        // Submitting without a project keeps the modal open.
        await form.getByTestId('gitlab-create-issue-submit').click();
        await expect(form).toBeVisible();
        await channelsPage.page.keyboard.press('Escape');
        await expect(form).toBeHidden();
    });
});

async function openPostMenuAction(channelsPage: ChannelsPage, message: string, actionTestId: string) {
    await channelsPage.centerView.postMessage(message);
    const post = await channelsPage.centerView.getLastPost();
    await post.toContainText(message);
    await post.hover();

    // Mattermost lists plugin post actions under the post's "Message actions" button.
    await post.postMenu.actionsButton.click();
    await channelsPage.page.getByTestId(actionTestId).click();
}

// The selectors are react-select inputs; options render outside this plugin's markup.
async function selectOption(channelsPage: ChannelsPage, container: Locator, text: string) {
    await container.getByRole('combobox').fill(text);
    await channelsPage.page.getByRole('option', {name: new RegExp(escapeRegExp(text))}).click();
}

function escapeRegExp(text: string) {
    return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
