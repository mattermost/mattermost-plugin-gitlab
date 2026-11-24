// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test} from '@mattermost/playwright-lib';

import {readUpgradeStateIfAny, removeSeededData} from '../../support/upgrade';

/**
 * @objective Remove what the seed created once every upgrade check has run
 *
 * @precondition
 * Runs as the teardown of upgrade-seed, after upgrade-apply and upgrade-verify finish or are skipped
 */
test('removes the seeded data', {tag: '@upgrade'}, async () => {
    // # Unsubscribe Town Square and delete the seeded group, if the seed got far enough to create them
    const state = readUpgradeStateIfAny();
    if (state) {
        await removeSeededData(state);
    }
});
