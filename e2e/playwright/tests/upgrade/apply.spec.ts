// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';
import semver from 'semver';

import {loginAdmin} from '../../support/admin';
import {findPluginBundle, installPlugin, restartPlugin} from '../../support/plugin';
import {bundleManifest, compareVersions, pluginErrorsSince, readUpgradeState, runningPluginVersion} from '../../support/upgrade';

/**
 * @objective Upgrade the plugin from the base release to the build under test, keeping the base release's settings
 *
 * @precondition
 * The upgrade-seed project installed the base release and seeded data through it
 */
test('upgrades the base release to the local build', {tag: '@upgrade'}, async () => {
    const state = readUpgradeState();
    const {client: adminClient} = await loginAdmin();
    const bundle = findPluginBundle();
    const buildVersion = bundleManifest(bundle).version;

    // Leave a margin for clock skew between the host and the server container.
    const since = new Date(Date.now() - 5000).toISOString();

    // # Upload the local build over the base release and restart it, without touching the plugin settings
    await installPlugin(adminClient, bundle);
    await restartPlugin(adminClient);

    // * Verify the local build is the one running
    const running = await runningPluginVersion(adminClient);
    expect(running).toBe(buildVersion);

    // * Verify it is the same major as the base release and not older
    expect(semver.major(running)).toBe(semver.major(state.baseVersion));
    const order = compareVersions(running, state.baseVersion);
    expect(order).toBeGreaterThanOrEqual(0);

    // A build past a newer tag is a true version upgrade. A build with the same version
    // as the base (only its +hash differs) still swaps in new code over the base data.
    const regime = order > 0 ? 'strict' : 'relaxed';
    test.info().annotations.push({type: 'upgrade', description: `${regime}: ${state.baseVersion} → ${running}`});

    // * Verify the server logged no plugin errors during the upgrade
    expect(pluginErrorsSince(since)).toEqual([]);
});
