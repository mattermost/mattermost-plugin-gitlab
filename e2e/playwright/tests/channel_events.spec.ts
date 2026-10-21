// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test, type ChannelsPage} from '@mattermost/playwright-lib';

import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {addWebhook, ensureConnected, expectPost, safeUnsubscribe, subscribe} from '../support/ui';

const ALL_CHANNEL_FEATURES = [
    'issues',
    'merges',
    'pushes',
    'issue_comments',
    'merge_request_comments',
    'issue_assigns',
    'merge_request_assigns',
    'tag',
    'releases',
].join(',');

test.describe('channel notifications for subscribed events', () => {
    let sandbox: Sandbox;
    let channelsPage: ChannelsPage;

    test.beforeEach(async ({pw}) => {
        sandbox = await createSandbox();
        channelsPage = await ensureConnected(pw);
        await subscribe(channelsPage, sandbox.path, ALL_CHANNEL_FEATURES);
        await addWebhook(channelsPage, sandbox.path);
    });

    test.afterEach(async () => {
        await safeUnsubscribe(channelsPage, sandbox?.path);
        await destroySandbox(sandbox);
    });

    test('posts the issue lifecycle', async () => {
        const {root, project, credentials} = sandbox;
        const page = channelsPage.page;
        const issue = await root.createIssue(project.id, `E2E lifecycle issue ${uniquePath('')}`);
        await expectPost(page, issue.title, 'new issue by');

        await root.createLabel(project.id, 'bug');
        await root.updateIssue(project.id, issue.iid, {labels: 'bug'});
        await expectPost(page, issue.title, 'issue labeled bug by');

        await root.updateIssue(project.id, issue.iid, {assignee_ids: [credentials.e2e_user_id]});
        await expectPost(page, issue.title, `was assigned to ${credentials.e2e_username}`);

        await root.updateIssue(project.id, issue.iid, {assignee_ids: []});
        await expectPost(page, issue.title, `was unassigned from ${credentials.e2e_username}`);

        const comment = `Issue comment ${uniquePath('')}`;
        await root.commentOnIssue(project.id, issue.iid, comment);
        await expectPost(page, 'New comment by root', comment);

        await root.closeIssue(project.id, issue.iid);
        await expectPost(page, issue.title, 'closed by');

        await root.reopenIssue(project.id, issue.iid);
        await expectPost(page, issue.title, 'reopened by');
    });

    test('posts the merge request lifecycle', async () => {
        const {root, user, project, credentials} = sandbox;
        const page = channelsPage.page;
        const mergeRequest = await root.createMergeRequest(project.id, `E2E lifecycle MR ${uniquePath('')}`, {targetBranch: project.default_branch});
        await expectPost(page, mergeRequest.title, 'new merge-request by');

        await root.updateMergeRequest(project.id, mergeRequest.iid, {assignee_ids: [credentials.e2e_user_id]});
        await expectPost(page, mergeRequest.title, `was assigned to ${credentials.e2e_username}`);

        const comment = `MR comment ${uniquePath('')}`;
        await root.commentOnMergeRequest(project.id, mergeRequest.iid, comment);
        await expectPost(page, 'New comment by root', comment);

        await user.approveMergeRequest(project.id, mergeRequest.iid);
        await expectPost(page, mergeRequest.title, `was approved by ${credentials.e2e_username}`);

        await user.unapproveMergeRequest(project.id, mergeRequest.iid);
        await expectPost(page, mergeRequest.title, `changes were requested by ${credentials.e2e_username}`);

        await root.closeMergeRequest(project.id, mergeRequest.iid);
        await expectPost(page, mergeRequest.title, 'was closed by');

        await root.reopenMergeRequest(project.id, mergeRequest.iid);
        await expectPost(page, mergeRequest.title, 'was reopened by');

        await root.mergeMergeRequest(project.id, mergeRequest.iid);
        await expectPost(page, mergeRequest.title, 'was merged by');
    });

    test('posts pushes, tags, and releases', async () => {
        const {root, project, path} = sandbox;
        const page = channelsPage.page;
        const branch = project.default_branch;

        const commitMessage = `E2E push ${uniquePath('')}`;
        await root.commitFile(project.id, branch, `${uniquePath('push')}.md`, 'e2e', commitMessage);
        await expectPost(page, `has pushed 1 commit to ${path}`, commitMessage);

        const tag = uniquePath('v1.0.0-e2e');
        await root.createTag(project.id, tag, branch);
        await expectPost(page, `New tag ${tag}`);

        await root.deleteTag(project.id, tag);
        await expectPost(page, `${tag} Tag deleted by`);

        const releaseName = `E2E release ${uniquePath('')}`;
        await root.createRelease(project.id, uniquePath('v2.0.0-e2e'), releaseName, branch);
        await expectPost(page, 'Release: create', releaseName);
    });
});
