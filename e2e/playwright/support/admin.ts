// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './env';

import {Client4} from '@mattermost/client';
import type {UserProfile} from '@mattermost/types/users';
import {testConfig} from '@mattermost/playwright-lib';

import {TEAM_NAME} from './constants';

const DISPOSABLE_SERVER_URL = 'http://mattermost:8065';

export function isDisposableServer() {
    return new URL(testConfig.baseURL).origin === DISPOSABLE_SERVER_URL;
}

// A fixed user, rather than a random one per test, keeps a single GitLab OAuth connection across specs.
const REGULAR_USER = {
    username: 'gitlab-e2e-user',
    email: 'gitlab-e2e-user@example.com',
};

// The default password is public, so other servers must supply their own.
function regularUserPassword() {
    const password = process.env.PW_REGULAR_USER_PASSWORD;
    if (password) {
        return password;
    }
    if (!isDisposableServer()) {
        throw new Error(`Set PW_REGULAR_USER_PASSWORD to run against ${testConfig.baseURL}.`);
    }
    return 'Gl-E2e-Member#2026';
}

async function login(username: string, password: string) {
    const client = new Client4();
    client.setUrl(testConfig.baseURL);
    const profile = await client.login(username, password);
    return {client, user: {...profile, password}};
}

// loginAdmin returns a Client4 authenticated as the sysadmin configured via
// PW_ADMIN_USERNAME/PW_ADMIN_PASSWORD. Usable outside test fixtures (e.g. in
// global setup), unlike pw.getAdminClient().
export function loginAdmin() {
    return login(testConfig.adminUsername, testConfig.adminPassword);
}

export function loginRegularUser() {
    return login(REGULAR_USER.username, regularUserPassword());
}

export async function ensureRegularUser(adminClient: Client4) {
    const password = regularUserPassword();
    let user: UserProfile;
    try {
        user = await adminClient.getUserByUsername(REGULAR_USER.username);
    } catch {
        user = await adminClient.createUser({...REGULAR_USER, password} as UserProfile, '', '');
    }
    if (user.roles.split(' ').includes('system_admin')) {
        throw new Error(`${REGULAR_USER.username} must not be a system admin`);
    }

    const team = await adminClient.getTeamByName(TEAM_NAME);
    await adminClient.addToTeam(team.id, user.id);
}
