// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';

import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {addWebhook, ensureConnected, expectPost, postWithText, safeUnsubscribe, subscribe} from '../support/ui';

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

    test('only posts issues that carry a subscribed label', async () => {
        const {root, project, path} = sandbox;
        await subscribe(channelsPage, path, 'issues,label:"bug"');
        await root.createLabel(project.id, 'bug');

        const unlabeled = await root.createIssue(project.id, `E2E unlabeled ${uniquePath('')}`);
        const labeled = await root.createIssue(project.id, `E2E labeled ${uniquePath('')}`, {labels: ['bug']});

        await expectPost(channelsPage.page, labeled.title, 'new issue by');
        await expect(postWithText(channelsPage.page, unlabeled.title)).toHaveCount(0);
    });

    test('posts assignments without issue events when only issue_assigns is subscribed', async () => {
        const {root, project, path, credentials} = sandbox;
        await subscribe(channelsPage, path, 'issue_assigns');

        const issue = await root.createIssue(project.id, `E2E assign only ${uniquePath('')}`);
        await root.updateIssue(project.id, issue.iid, {assignee_ids: [credentials.e2e_user_id]});

        await expectPost(channelsPage.page, issue.title, 'was assigned to');
        await expect(postWithText(channelsPage.page, issue.title, 'new issue by')).toHaveCount(0);
    });

    test('skips merge requests and pushes that are not subscribed', async () => {
        const {root, project, path} = sandbox;
        await subscribe(channelsPage, path, 'issues');

        const mergeRequest = await root.createMergeRequest(project.id, `E2E unsubscribed MR ${uniquePath('')}`, {targetBranch: project.default_branch});
        const issue = await root.createIssue(project.id, `E2E subscribed issue ${uniquePath('')}`);

        await expectPost(channelsPage.page, issue.title, 'new issue by');
        await expect(postWithText(channelsPage.page, mergeRequest.title)).toHaveCount(0);
        await expect(postWithText(channelsPage.page, `has pushed 1 commit to ${path}`)).toHaveCount(0);
    });

    test('posts confidential issues only with confidential_issues', async () => {
        const {root, project, path} = sandbox;
        await subscribe(channelsPage, path, 'issues');

        const hidden = await root.createIssue(project.id, `E2E hidden confidential ${uniquePath('')}`, {confidential: true});
        const visible = await root.createIssue(project.id, `E2E public ${uniquePath('')}`);
        await expectPost(channelsPage.page, visible.title, 'new issue by');
        await expect(postWithText(channelsPage.page, hidden.title)).toHaveCount(0);

        await subscribe(channelsPage, path, 'issues,confidential_issues');
        const shown = await root.createIssue(project.id, `E2E shown confidential ${uniquePath('')}`, {confidential: true});
        await expectPost(channelsPage.page, shown.title, 'new issue by');
    });
});
