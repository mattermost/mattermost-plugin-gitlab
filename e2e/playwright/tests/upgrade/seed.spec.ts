// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {loginAdmin} from '../../support/admin';
import {GitlabRhs} from '../../support/components';
import {loadCredentials} from '../../support/credentials';
import {uniquePath} from '../../support/gitlab_api';
import {findPluginBundle, installAndConfigurePlugin} from '../../support/plugin';
import {createSandbox} from '../../support/sandbox';
import {addWebhook, expectPost, openTownSquare, runCommandExpecting, subscribe} from '../../support/ui';
import {
    UPGRADE_FEATURES,
    UPGRADE_LABEL,
    bundleManifest,
    fetchBaseRelease,
    readUpgradeStateIfAny,
    removeSeededData,
    runningPluginVersion,
    upgradeFrom,
    writeUpgradeState,
} from '../../support/upgrade';

/**
 * @objective Create data through the base release's own flows, for the upgrade run to verify on the new build
 *
 * @precondition
 * E2E_UPGRADE_FROM is set, and dist/ holds the build under test
 */
test('installs the base release and seeds data through it', {tag: '@upgrade'}, async ({pw}) => {
    // The install wizard and the bundle download take longer than a normal test.
    test.setTimeout(10 * 60 * 1000);
    const {client: adminClient} = await loginAdmin();

    // # Resolve the release to upgrade from and download its bundle
    const buildVersion = bundleManifest(findPluginBundle()).version;
    const base = await fetchBaseRelease(adminClient, upgradeFrom() ?? 'latest', buildVersion);
    test.info().annotations.push({type: 'upgrade', description: `${base.tag} → ${buildVersion}`});

    // # Install the base release and configure it the way an admin would
    await installAndConfigurePlugin(base.bundlePath);

    // * Verify the base release is the one running
    expect(await runningPluginVersion(adminClient)).toBe(base.manifest.version);

    // # Connect the regular member through the GitLab OAuth popup. Installing the instance disconnected them.
    const credentials = loadCredentials();
    const channelsPage = await openTownSquare(pw);
    const rhs = new GitlabRhs(channelsPage);
    await rhs.open();
    await rhs.signInThroughPopup(credentials);

    // * Verify the member is connected. The base release's sidebar predates the test ids, so ask the plugin.
    await expect(async () => {
        const profile = await runCommandExpecting(channelsPage, '/gitlab me', 'You are connected to GitLab as:');
        await profile.toContainText(credentials.e2e_username);
    }).toPass({timeout: 60000});

    // # Remove what an earlier upgrade run seeded
    const previous = readUpgradeStateIfAny();
    if (previous) {
        await removeSeededData(previous);
    }

    // # Create a project and subscribe Town Square to it with a label filter
    const sandbox = await createSandbox();
    writeUpgradeState({
        baseTag: base.tag,
        baseVersion: base.manifest.version,
        groupId: sandbox.group.id,
        projectId: sandbox.project.id,
        projectPath: sandbox.path,
    });
    await sandbox.root.createLabel(sandbox.project.id, UPGRADE_LABEL);
    await subscribe(channelsPage, sandbox.path, UPGRADE_FEATURES);
    await addWebhook(channelsPage, sandbox.path);

    // # Open a labeled issue
    const issue = await sandbox.root.createIssue(sandbox.project.id, `E2E seeded issue ${uniquePath('')}`, {labels: [UPGRADE_LABEL]});

    // * Verify the base release delivers it to the subscribed channel
    await expectPost(channelsPage.page, issue.title, 'new issue by');

    writeUpgradeState({
        baseTag: base.tag,
        baseVersion: base.manifest.version,
        groupId: sandbox.group.id,
        projectId: sandbox.project.id,
        projectPath: sandbox.path,
        seededIssueTitle: issue.title,
    });
});
