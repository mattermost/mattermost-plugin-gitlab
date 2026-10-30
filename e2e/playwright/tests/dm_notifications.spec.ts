// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test, type ChannelsPage} from '@mattermost/playwright-lib';

import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {
    addWebhook,
    ensureConnected,
    expectNoPost,
    expectPost,
    openBotDirectMessage,
    runCommandExpecting,
    safeUnsubscribe,
    subscribe,
} from '../support/ui';

// Root acts in GitLab, and the connected user receives the bot DMs. DMs do not
// depend on channel subscriptions, only on the project webhook.
test.describe('direct message notifications', () => {
    let sandbox: Sandbox;
    let channelsPage: ChannelsPage;

    test.beforeEach(async ({pw}) => {
        sandbox = await createSandbox();
        channelsPage = await ensureConnected(pw);
        await addWebhook(channelsPage, sandbox.path);
        await openBotDirectMessage(channelsPage);
    });

    test.afterEach(async () => {
        await safeUnsubscribe(channelsPage, sandbox?.path);
        await destroySandbox(sandbox);
    });

    /**
     * @objective Verify the GitLab bot DMs the connected user about activity on issues they are assigned to or mentioned in
     *
     * @precondition
     * A new project has a plugin webhook, and the member is viewing their DM with the GitLab bot
     */
    test('DMs the connected user about issue assignment, comments, mentions, and state changes', {tag: '@gitlab_plugin'}, async () => {
        const {root, project, path, credentials} = sandbox;
        const page = channelsPage.page;
        const reference = (iid: number) => `${path}#${iid}`;

        // # Open an issue assigned to the connected user
        const issue = await root.createIssue(project.id, `E2E DM issue ${uniquePath('')}`, {assigneeIds: [credentials.e2e_user_id]});

        // * Verify the assignment DM arrives
        await expectPost(page, `assigned you to issue ${reference(issue.iid)}`);

        // # Comment on the assigned issue
        await root.commentOnIssue(project.id, issue.iid, 'Plain follow-up');

        // * Verify the comment DM arrives
        await expectPost(page, `commented on your issue ${reference(issue.iid)}`);

        // # Mention the user on an issue they are not assigned to
        // A user gets one DM per webhook event, so an assignee who is also mentioned only hears about the comment.
        const other = await root.createIssue(project.id, `E2E DM mention ${uniquePath('')}`);
        const mention = `ping ${uniquePath('')}`;
        await root.commentOnIssue(project.id, other.iid, `@${credentials.e2e_username} ${mention}`);

        // * Verify the mention DM arrives
        await expectPost(page, `mentioned you on ${reference(other.iid)}`, mention);

        // # Close, reopen, and unassign the assigned issue
        await root.closeIssue(project.id, issue.iid);

        // * Verify a DM arrives for each change
        await expectPost(page, `closed your issue ${reference(issue.iid)}`);
        await root.reopenIssue(project.id, issue.iid);
        await expectPost(page, `reopened your issue ${reference(issue.iid)}`);
        await root.updateIssue(project.id, issue.iid, {assignee_ids: []});
        await expectPost(page, `unassigned you from issue ${reference(issue.iid)}`);
    });

    /**
     * @objective Verify the GitLab bot DMs the connected user about review requests and activity on their merge requests
     *
     * @precondition
     * A new project has a plugin webhook, and the member is viewing their DM with the GitLab bot
     */
    test('DMs the connected user about merge request reviews, comments, approvals, and merges', {tag: '@gitlab_plugin'}, async () => {
        const {root, user, project, path, credentials} = sandbox;
        const page = channelsPage.page;

        // # Open a merge request assigned to the connected user
        const assigned = await root.createMergeRequest(project.id, `E2E DM review ${uniquePath('')}`, {
            targetBranch: project.default_branch,
            assigneeIds: [credentials.e2e_user_id],
        });

        // * Verify the assignment DM arrives
        await expectPost(page, `requested your review on #${assigned.iid} in ${path}`);

        // # Add the connected user as a reviewer
        await root.updateMergeRequest(project.id, assigned.iid, {reviewer_ids: [credentials.e2e_user_id]});

        // * Verify the review request DM arrives
        await expectPost(page, `requested your review on merge request #${assigned.iid} in ${path}`);

        // # Comment on and close the merge request
        await root.commentOnMergeRequest(project.id, assigned.iid, 'Looks good');

        // * Verify the comment and close DMs arrive
        await expectPost(page, `commented on your merge request ${path}#${assigned.iid}`);
        await root.closeMergeRequest(project.id, assigned.iid);
        await expectPost(page, `closed your merge request #${assigned.iid} in ${path}`);

        // # Approve and merge a merge request authored by the connected user
        const authored = await user.createMergeRequest(project.id, `E2E DM authored ${uniquePath('')}`, {targetBranch: project.default_branch});
        await root.approveMergeRequest(project.id, authored.iid);

        // * Verify the approval and merge DMs arrive
        await expectPost(page, `approved your merge request #${authored.iid} in ${path}`);
        await root.mergeMergeRequest(project.id, authored.iid);
        await expectPost(page, `merged your merge request #${authored.iid} in ${path}`);
    });

    /**
     * @objective Verify turning notifications off stops mention DMs until they are turned back on
     *
     * @precondition
     * A new project has a plugin webhook, and the member is viewing their DM with the GitLab bot
     */
    test('stops sending mention DMs while notifications are turned off', {tag: '@gitlab_plugin'}, async () => {
        const {root, project, path, credentials} = sandbox;
        const page = channelsPage.page;
        const issue = await root.createIssue(project.id, `E2E muted ${uniquePath('')}`);
        const muted = `muted ${uniquePath('')}`;
        const heard = `heard ${uniquePath('')}`;

        // # Subscribe the DM channel to issue comments, so a channel post shows each comment webhook was handled
        await subscribe(channelsPage, path, 'issue_comments');
        try {
            // # Turn notifications off and mention the user
            await runCommandExpecting(channelsPage, '/gitlab settings notifications off', 'Settings updated: notifications set to off.');
            await root.commentOnIssue(project.id, issue.iid, `@${credentials.e2e_username} ${muted}`);

            // * Verify the comment webhook was handled
            await expectPost(page, 'New comment by root', muted);
        } finally {
            // # Turn notifications back on
            await runCommandExpecting(channelsPage, '/gitlab settings notifications on', 'Settings updated: notifications set to on.');
        }

        // # Mention the user again
        await root.commentOnIssue(project.id, issue.iid, `@${credentials.e2e_username} ${heard}`);

        // * Verify only the mention made after notifications were turned on is DMed
        await expectPost(page, 'mentioned you on', heard);
        await expectNoPost(page, 'mentioned you on', muted);
    });
});
