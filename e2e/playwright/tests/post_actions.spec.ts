// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {AttachCommentModal, CreateIssueModal} from '../support/components';
import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {ensureConnected, openPostMenuAction, runSlashCommand} from '../support/ui';

const GITLAB_TIMEOUT = 20000;

test.describe('post menu actions and the issue modal', () => {
    let sandbox: Sandbox;

    test.beforeEach(async () => {
        sandbox = await createSandbox();
    });

    test.afterEach(async () => {
        await destroySandbox(sandbox);
    });

    /**
     * @objective Verify "Create GitLab Issue" in the post menu creates an issue whose description is the post message
     */
    test('creates a GitLab issue from the post menu with the post as its description', {tag: '@gitlab_plugin'}, async ({pw}) => {
        const {root, project, path} = sandbox;
        const message = `Create an issue from this post ${uniquePath('')}`;
        const title = `E2E issue from post ${uniquePath('')}`;

        // # Post a message as a connected member
        const channelsPage = await ensureConnected(pw);
        await channelsPage.centerView.postMessage(message);
        const post = await channelsPage.centerView.getLastPost();
        await post.toContainText(message);

        // # Open "Create GitLab Issue" from the post menu
        // With more than 100 projects the selector filters whatever list it has when typing
        // starts, so the project list must finish loading first.
        const projectsLoaded = channelsPage.page.waitForResponse((response) => response.url().endsWith('/api/v1/projects') && response.ok());
        await openPostMenuAction(channelsPage, post, 'Create GitLab Issue');
        const modal = new CreateIssueModal(channelsPage.page);

        // * Verify the description is prefilled with the post message
        await expect(modal.description).toHaveValue(message);
        await projectsLoaded;

        // # Pick the project, enter a title, and submit
        await modal.selectProject(path);
        await modal.title.fill(title);
        await modal.submit();

        // * Verify the modal closes and GitLab has the issue with the post in its description
        await modal.toBeHidden();
        await expect.poll(async () => {
            const issue = (await root.listIssues(project.id)).find((candidate) => candidate.title === title);
            return issue?.description ?? '';
        }, {timeout: GITLAB_TIMEOUT}).toContain(message);
    });

    /**
     * @objective Verify "Attach to GitLab Issue" in the post menu adds the post message as a comment on the chosen issue
     */
    test('attaches a post to an existing GitLab issue from the post menu', {tag: '@gitlab_plugin'}, async ({pw}) => {
        const {root, project} = sandbox;
        const issue = await root.createIssue(project.id, `E2E attach target ${uniquePath('')}`);
        const message = `Attach this post ${uniquePath('')}`;

        // # Post a message as a connected member
        const channelsPage = await ensureConnected(pw);
        await channelsPage.centerView.postMessage(message);
        const post = await channelsPage.centerView.getLastPost();
        await post.toContainText(message);

        // # Open "Attach to GitLab Issue" from the post menu
        await openPostMenuAction(channelsPage, post, 'Attach to GitLab Issue');
        const modal = new AttachCommentModal(channelsPage.page);

        // * Verify the message is prefilled with the post
        await expect(modal.message).toHaveValue(message);

        // # Pick the issue and attach the post
        await modal.selectIssue(issue.title);
        await modal.attach();

        // * Verify the modal closes and the issue has a comment with the post
        await modal.toBeHidden();
        await expect.poll(async () => {
            const comments = await root.listIssueComments(project.id, issue.iid);
            return comments.filter((comment) => !comment.system).map((comment) => comment.body).join('\n');
        }, {timeout: GITLAB_TIMEOUT}).toContain(message);
    });

    /**
     * @objective Verify /gitlab issue create opens the issue modal with the given title and requires a project
     */
    test('opens the issue modal from /gitlab issue create with the title prefilled', {tag: '@gitlab_plugin'}, async ({pw}) => {
        const title = `E2E modal title ${uniquePath('')}`;

        // # Run /gitlab issue create with a title
        const channelsPage = await ensureConnected(pw);
        await runSlashCommand(channelsPage, `/gitlab issue create ${title}`);
        const modal = new CreateIssueModal(channelsPage.page);

        // * Verify the modal opens with the title prefilled
        await expect(modal.title).toHaveValue(title);

        // # Submit without picking a project
        await modal.submit();

        // * Verify the modal stays open
        await modal.toBeVisible();

        // # Close the modal
        await modal.close();

        // * Verify the modal closes
        await modal.toBeHidden();
    });
});
