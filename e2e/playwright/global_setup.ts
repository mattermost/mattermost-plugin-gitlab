// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './support/env';

import {baseGlobalSetup} from '@mattermost/playwright-lib';

import {prepareServer} from './support/plugin';

export default async function globalSetup() {
    await baseGlobalSetup();
    await prepareServer();
}
