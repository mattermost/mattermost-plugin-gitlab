// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {GitlabRhs} from '../../../support/components';
import {loadCredentials} from '../../../support/credentials';
import {GitlabApi, uniquePath} from '../../../support/gitlab_api';
import {expectNoPost, expectPost, openTownSquare, runCommandExpecting} from '../../../support/ui';
import {UPGRADE_FEATURES, UPGRADE_LABEL, readUpgradeState, removeSeededData} from '../../../support/upgrade';

test.describe('subscriptions after the upgrade', () => {
    // The last verify spec, so it removes what the seed created.
    test.afterAll(async () => {
        await removeSeededData(readUpgradeState());
    });

    /**
     * @objective Verify a subscription created on the base release is still stored with its features
     *
     * @precondition
     * Town Square was subscribed to the seeded project with a label filter on the base release
     */
    test('keeps the base release subscription and its features', {tag: '@upgrade'}, async ({pw}) => {
        const {projectPath} = readUpgradeState();

        // # Open Town Square and the GitLab sidebar
        const channelsPage = await openTownSquare(pw);
        const rhs = new GitlabRhs(channelsPage);
        await rhs.open();

        // * Verify the list reply shows the subscription with the same features
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', `${projectPath} - ${UPGRADE_FEATURES}`);

        // * Verify the sidebar shows the subscription and its features
        await expect(rhs.subscription(projectPath)).toHaveCount(1);
        for (const feature of ['issues', 'issue_comments']) {
            // eslint-disable-next-line no-await-in-loop
            await expect(rhs.subscriptionFeature(projectPath, feature)).toBeVisible();
        }
    });

    /**
     * @objective Verify the webhook and subscription created on the base release still deliver events, with the label filter applied
     *
     * @precondition
     * The seeded project has the webhook added on the base release, and Town Square is subscribed to it with a label filter
     */
    test('delivers events for the base release subscription', {tag: '@upgrade'}, async ({pw}) => {
        const {projectId} = readUpgradeState();
        const root = new GitlabApi(loadCredentials());
        const channelsPage = await openTownSquare(pw);

        // # Open one unlabeled and one labeled issue in the seeded project
        const unlabeled = await root.createIssue(projectId, `E2E upgrade unlabeled ${uniquePath('')}`);
        const labeled = await root.createIssue(projectId, `E2E upgrade labeled ${uniquePath('')}`, {labels: [UPGRADE_LABEL]});

        // * Verify only the labeled issue is posted
        await expectPost(channelsPage.page, labeled.title, 'new issue by');
        await expectNoPost(channelsPage.page, unlabeled.title);
    });
});
