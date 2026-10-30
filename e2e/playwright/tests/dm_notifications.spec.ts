// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';

import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {
    addWebhook,
    ensureConnected,
    expectPost,
    openBotDirectMessage,
    postWithText,
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
        await openBotDirectMessage(channelsPage.page);
    });

    test.afterEach(async () => {
        await safeUnsubscribe(channelsPage, sandbox?.path);
        await destroySandbox(sandbox);
    });

    test('notifies about issue assignment, comments, mentions, and state changes', async () => {
        const {root, project, path, credentials} = sandbox;
        const page = channelsPage.page;
        const reference = (iid: number) => `${path}#${iid}`;

        const issue = await root.createIssue(project.id, `E2E DM issue ${uniquePath('')}`, {assigneeIds: [credentials.e2e_user_id]});
        await expectPost(page, `assigned you to issue ${reference(issue.iid)}`);

        await root.commentOnIssue(project.id, issue.iid, 'Plain follow-up');
        await expectPost(page, `commented on your issue ${reference(issue.iid)}`);

        // A user gets one DM per webhook event, so an assignee who is also mentioned only
        // hears about the comment. The mention is made on an issue they are not assigned to.
        const other = await root.createIssue(project.id, `E2E DM mention ${uniquePath('')}`);
        const mention = `ping ${uniquePath('')}`;
        await root.commentOnIssue(project.id, other.iid, `@${credentials.e2e_username} ${mention}`);
        await expectPost(page, `mentioned you on ${reference(other.iid)}`, mention);

        await root.closeIssue(project.id, issue.iid);
        await expectPost(page, `closed your issue ${reference(issue.iid)}`);

        await root.reopenIssue(project.id, issue.iid);
        await expectPost(page, `reopened your issue ${reference(issue.iid)}`);

        await root.updateIssue(project.id, issue.iid, {assignee_ids: []});
        await expectPost(page, `unassigned you from issue ${reference(issue.iid)}`);
    });

    test('notifies about merge request reviews, approvals, and merges', async () => {
        const {root, user, project, path, credentials} = sandbox;
        const page = channelsPage.page;

        const assigned = await root.createMergeRequest(project.id, `E2E DM review ${uniquePath('')}`, {
            targetBranch: project.default_branch,
            assigneeIds: [credentials.e2e_user_id],
        });
        await expectPost(page, `requested your review on #${assigned.iid} in ${path}`);

        await root.updateMergeRequest(project.id, assigned.iid, {reviewer_ids: [credentials.e2e_user_id]});
        await expectPost(page, `requested your review on merge request #${assigned.iid} in ${path}`);

        await root.commentOnMergeRequest(project.id, assigned.iid, 'Looks good');
        await expectPost(page, `commented on your merge request ${path}#${assigned.iid}`);

        await root.closeMergeRequest(project.id, assigned.iid);
        await expectPost(page, `closed your merge request #${assigned.iid} in ${path}`);

        const authored = await user.createMergeRequest(project.id, `E2E DM authored ${uniquePath('')}`, {targetBranch: project.default_branch});
        await root.approveMergeRequest(project.id, authored.iid);
        await expectPost(page, `approved your merge request #${authored.iid} in ${path}`);

        await root.mergeMergeRequest(project.id, authored.iid);
        await expectPost(page, `merged your merge request #${authored.iid} in ${path}`);
    });

    // The comment subscription posts in this DM channel too, which shows the webhook
    // was handled before notifications are turned back on.
    test('stops sending DMs while notifications are turned off', async () => {
        const {root, project, path, credentials} = sandbox;
        const page = channelsPage.page;
        const issue = await root.createIssue(project.id, `E2E muted ${uniquePath('')}`);
        const muted = `muted ${uniquePath('')}`;
        const heard = `heard ${uniquePath('')}`;

        await subscribe(channelsPage, path, 'issue_comments');
        try {
            await runCommandExpecting(channelsPage, '/gitlab settings notifications off', 'Settings updated: notifications set to off.');
            await root.commentOnIssue(project.id, issue.iid, `@${credentials.e2e_username} ${muted}`);
            await expectPost(page, 'New comment by root', muted);
        } finally {
            await runCommandExpecting(channelsPage, '/gitlab settings notifications on', 'Settings updated: notifications set to on.');
        }

        await root.commentOnIssue(project.id, issue.iid, `@${credentials.e2e_username} ${heard}`);
        await expectPost(page, 'mentioned you on', heard);
        await expect(postWithText(page, 'mentioned you on', muted)).toHaveCount(0);
    });
});
