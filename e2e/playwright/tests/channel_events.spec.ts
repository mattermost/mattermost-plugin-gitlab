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

    /**
     * @objective Verify every issue lifecycle event posts in a channel subscribed to the project
     *
     * @precondition
     * Town Square is subscribed to a new project with every channel feature, and the project has a plugin webhook
     */
    test('posts issue creation, labels, assignment, comments, and state changes in the subscribed channel', {tag: '@gitlab_plugin'}, async () => {
        const {root, project, credentials} = sandbox;
        const page = channelsPage.page;

        // # Open an issue in GitLab
        const issue = await root.createIssue(project.id, `E2E lifecycle issue ${uniquePath('')}`);

        // * Verify the new issue is posted
        await expectPost(page, issue.title, 'new issue by');

        // # Label the issue
        await root.createLabel(project.id, 'bug');
        await root.updateIssue(project.id, issue.iid, {labels: 'bug'});

        // * Verify the label change is posted
        await expectPost(page, issue.title, 'issue labeled bug by');

        // # Assign and then unassign the connected user
        await root.updateIssue(project.id, issue.iid, {assignee_ids: [credentials.e2e_user_id]});

        // * Verify the assignment is posted
        await expectPost(page, issue.title, `was assigned to ${credentials.e2e_username}`);

        await root.updateIssue(project.id, issue.iid, {assignee_ids: []});

        // * Verify the unassignment is posted
        await expectPost(page, issue.title, `was unassigned from ${credentials.e2e_username}`);

        // # Comment on the issue
        const comment = `Issue comment ${uniquePath('')}`;
        await root.commentOnIssue(project.id, issue.iid, comment);

        // * Verify the comment is posted
        await expectPost(page, 'New comment by root', comment);

        // # Close and reopen the issue
        await root.closeIssue(project.id, issue.iid);

        // * Verify both state changes are posted
        await expectPost(page, issue.title, 'closed by');
        await root.reopenIssue(project.id, issue.iid);
        await expectPost(page, issue.title, 'reopened by');
    });

    /**
     * @objective Verify every merge request lifecycle event posts in a channel subscribed to the project
     *
     * @precondition
     * Town Square is subscribed to a new project with every channel feature, and the project has a plugin webhook
     */
    test('posts merge request creation, assignment, comments, approvals, and merges in the subscribed channel', {tag: '@gitlab_plugin'}, async () => {
        const {root, user, project, credentials} = sandbox;
        const page = channelsPage.page;

        // # Open a merge request in GitLab
        const mergeRequest = await root.createMergeRequest(project.id, `E2E lifecycle MR ${uniquePath('')}`, {targetBranch: project.default_branch});

        // * Verify the new merge request is posted
        await expectPost(page, mergeRequest.title, 'new merge-request by');

        // # Assign the connected user
        await root.updateMergeRequest(project.id, mergeRequest.iid, {assignee_ids: [credentials.e2e_user_id]});

        // * Verify the assignment is posted
        await expectPost(page, mergeRequest.title, `was assigned to ${credentials.e2e_username}`);

        // # Comment on the merge request
        const comment = `MR comment ${uniquePath('')}`;
        await root.commentOnMergeRequest(project.id, mergeRequest.iid, comment);

        // * Verify the comment is posted
        await expectPost(page, 'New comment by root', comment);

        // # Approve and then unapprove as the connected user
        await user.approveMergeRequest(project.id, mergeRequest.iid);

        // * Verify the approval and the requested changes are posted
        await expectPost(page, mergeRequest.title, `was approved by ${credentials.e2e_username}`);
        await user.unapproveMergeRequest(project.id, mergeRequest.iid);
        await expectPost(page, mergeRequest.title, `changes were requested by ${credentials.e2e_username}`);

        // # Close, reopen, and merge the merge request
        await root.closeMergeRequest(project.id, mergeRequest.iid);

        // * Verify each state change is posted
        await expectPost(page, mergeRequest.title, 'was closed by');
        await root.reopenMergeRequest(project.id, mergeRequest.iid);
        await expectPost(page, mergeRequest.title, 'was reopened by');
        await root.mergeMergeRequest(project.id, mergeRequest.iid);
        await expectPost(page, mergeRequest.title, 'was merged by');
    });

    /**
     * @objective Verify pushes, tag changes, and releases post in a channel subscribed to the project
     *
     * @precondition
     * Town Square is subscribed to a new project with every channel feature, and the project has a plugin webhook
     */
    test('posts pushes, tag creation and deletion, and releases in the subscribed channel', {tag: '@gitlab_plugin'}, async () => {
        const {root, project, path} = sandbox;
        const page = channelsPage.page;
        const branch = project.default_branch;

        // # Push a commit to the default branch
        const commitMessage = `E2E push ${uniquePath('')}`;
        await root.commitFile(project.id, branch, `${uniquePath('push')}.md`, 'e2e', commitMessage);

        // * Verify the push is posted with its commit message
        await expectPost(page, `has pushed 1 commit to ${path}`, commitMessage);

        // # Create and then delete a tag
        const tag = uniquePath('v1.0.0-e2e');
        await root.createTag(project.id, tag, branch);

        // * Verify the tag creation and deletion are posted
        await expectPost(page, `New tag ${tag}`);
        await root.deleteTag(project.id, tag);
        await expectPost(page, `${tag} Tag deleted by`);

        // # Publish a release
        const releaseName = `E2E release ${uniquePath('')}`;
        await root.createRelease(project.id, uniquePath('v2.0.0-e2e'), releaseName, branch);

        // * Verify the release is posted
        await expectPost(page, 'Release: create', releaseName);
    });
});
