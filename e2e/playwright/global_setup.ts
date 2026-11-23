// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './support/env';

import {baseGlobalSetup} from '@mattermost/playwright-lib';

import {prepareServer, prepareStack} from './support/plugin';
import {upgradeFrom} from './support/upgrade';

export default async function globalSetup() {
    await baseGlobalSetup();

    // In an upgrade run the seed spec installs the base release and the apply spec
    // installs dist/, so setup leaves the plugin alone.
    if (upgradeFrom()) {
        await prepareStack();
        return;
    }
    await prepareServer();
}
