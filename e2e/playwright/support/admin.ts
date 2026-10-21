// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './env';

import {Client4} from '@mattermost/client';
import {testConfig} from '@mattermost/playwright-lib';

// loginAdmin returns a Client4 authenticated as the sysadmin configured via
// PW_ADMIN_USERNAME/PW_ADMIN_PASSWORD. Usable outside test fixtures (e.g. in
// global setup), unlike pw.getAdminClient().
export async function loginAdmin() {
    const client = new Client4();
    client.setUrl(testConfig.baseURL);
    const profile = await client.login(testConfig.adminUsername, testConfig.adminPassword);
    return {client, user: {...profile, password: testConfig.adminPassword}};
}
