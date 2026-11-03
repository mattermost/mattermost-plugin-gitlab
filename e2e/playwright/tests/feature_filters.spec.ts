// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test, type ChannelsPage} from '@mattermost/playwright-lib';

import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {addWebhook, ensureConnected, expectNoPost, expectPost, safeUnsubscribe, subscribe} from '../support/ui';

// GitLab does not guarantee webhook order, so each negative check is made only
// after a later event that does reach the channel has been posted.
test.describe('subscription feature filters', () => {
    let sandbox: Sandbox;
    let channelsPage: ChannelsPage;

    test.beforeEach(async ({pw}) => {
        sandbox = await createSandbox();
        channelsPage = await ensureConnected(pw);
        await addWebhook(channelsPage, sandbox.path);
    });

    test.afterEach(async () => {
        await safeUnsubscribe(channelsPage, sandbox?.path);
        await destroySandbox(sandbox);
    });

    /**
     * @objective Verify a label filter limits issue posts to issues carrying that label
     *
     * @precondition
     * A new project has a plugin webhook
     */
    test('posts only issues carrying the subscribed label', {tag: '@gitlab_plugin'}, async () => {
        const {root, project, path} = sandbox;

        // # Subscribe to issues labeled "bug"
        await subscribe(channelsPage, path, 'issues,label:"bug"');
        await root.createLabel(project.id, 'bug');

        // # Open one unlabeled and one labeled issue
        const unlabeled = await root.createIssue(project.id, `E2E unlabeled ${uniquePath('')}`);
        const labeled = await root.createIssue(project.id, `E2E labeled ${uniquePath('')}`, {labels: ['bug']});

        // * Verify only the labeled issue is posted
        await expectPost(channelsPage.page, labeled.title, 'new issue by');
        await expectNoPost(channelsPage.page, unlabeled.title);
    });

    /**
     * @objective Verify an issue_assigns subscription posts assignments without new issue events
     *
     * @precondition
     * A new project has a plugin webhook
     */
    test('posts issue assignments without new issue events when only issue_assigns is subscribed', {tag: '@gitlab_plugin'}, async () => {
        const {root, project, path, credentials} = sandbox;

        // # Subscribe to issue assignments only
        await subscribe(channelsPage, path, 'issue_assigns');

        // # Open an issue and assign the connected user
        const issue = await root.createIssue(project.id, `E2E assign only ${uniquePath('')}`);
        await root.updateIssue(project.id, issue.iid, {assignee_ids: [credentials.e2e_user_id]});

        // * Verify the assignment is posted but the new issue is not
        await expectPost(channelsPage.page, issue.title, 'was assigned to');
        await expectNoPost(channelsPage.page, issue.title, 'new issue by');
    });

    /**
     * @objective Verify an issues subscription skips merge request and push events
     *
     * @precondition
     * A new project has a plugin webhook
     */
    test('skips merge requests and pushes when only issues are subscribed', {tag: '@gitlab_plugin'}, async () => {
        const {root, project, path} = sandbox;

        // # Subscribe to issues only
        await subscribe(channelsPage, path, 'issues');

        // # Open a merge request and then an issue
        const mergeRequest = await root.createMergeRequest(project.id, `E2E unsubscribed MR ${uniquePath('')}`, {targetBranch: project.default_branch});
        const issue = await root.createIssue(project.id, `E2E subscribed issue ${uniquePath('')}`);

        // * Verify only the issue is posted
        await expectPost(channelsPage.page, issue.title, 'new issue by');
        await expectNoPost(channelsPage.page, mergeRequest.title);
        await expectNoPost(channelsPage.page, `has pushed 1 commit to ${path}`);
    });

    /**
     * @objective Verify confidential issues post only when the subscription includes confidential_issues
     *
     * @precondition
     * A new project has a plugin webhook
     */
    test('posts confidential issues only when confidential_issues is subscribed', {tag: '@gitlab_plugin'}, async () => {
        const {root, project, path} = sandbox;

        // # Subscribe to issues without confidential issues
        await subscribe(channelsPage, path, 'issues');

        // # Open a confidential issue and then a public one
        const hidden = await root.createIssue(project.id, `E2E hidden confidential ${uniquePath('')}`, {confidential: true});
        const visible = await root.createIssue(project.id, `E2E public ${uniquePath('')}`);

        // * Verify only the public issue is posted
        await expectPost(channelsPage.page, visible.title, 'new issue by');
        await expectNoPost(channelsPage.page, hidden.title);

        // # Add confidential issues to the subscription and open another confidential issue
        await subscribe(channelsPage, path, 'issues,confidential_issues');
        const shown = await root.createIssue(project.id, `E2E shown confidential ${uniquePath('')}`, {confidential: true});

        // * Verify the confidential issue is posted
        await expectPost(channelsPage.page, shown.title, 'new issue by');
    });
});
