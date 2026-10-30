// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {GITLAB_URL, INSTANCE_NAME} from '../support/constants';
import {loadCredentials} from '../support/credentials';
import {ensureConnected, openTownSquare, runCommandExpecting} from '../support/ui';

const UNKNOWN_ACTION = 'Unknown action, please use /gitlab help to see all actions available.';

test.describe('slash commands', () => {
    test('shows help and build information', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, '/gitlab help', 'Mattermost GitLab Plugin - Slash Command Help');
        await runCommandExpecting(channelsPage, '/gitlab about', /GitLab version/i);
        await runCommandExpecting(channelsPage, '/gitlab unknown-action', UNKNOWN_ACTION);
    });

    test('shows the connected GitLab profile', async ({pw}) => {
        const credentials = loadCredentials();
        const channelsPage = await ensureConnected(pw);

        const profile = await runCommandExpecting(channelsPage, '/gitlab me', 'You are connected to GitLab as:');
        await expect(profile).toContainText(credentials.e2e_username);

        const alreadyConnected = await runCommandExpecting(channelsPage, '/gitlab connect', 'You are already connected to GitLab as');
        await expect(alreadyConnected).toContainText(credentials.e2e_username);
    });

    test('restricts instance management to system admins', async ({pw}) => {
        const memberPage = await ensureConnected(pw);
        await runCommandExpecting(memberPage, '/gitlab instance list', 'Only System Admins are allowed to manage instances.');

        const adminPage = await openTownSquare(pw, {asAdmin: true});
        const instances = await runCommandExpecting(adminPage, '/gitlab instance list', 'Installed GitLab Instances');
        await expect(instances).toContainText(INSTANCE_NAME);
        await expect(instances).toContainText(GITLAB_URL);
    });

    test('updates personal settings and rejects invalid values', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, '/gitlab settings reminders off', 'Settings updated: reminders set to off.');
        await runCommandExpecting(channelsPage, '/gitlab settings reminders on', 'Settings updated: reminders set to on.');
        await runCommandExpecting(channelsPage, '/gitlab settings notifications off', 'Settings updated: notifications set to off.');
        await runCommandExpecting(channelsPage, '/gitlab settings notifications on', 'Settings updated: notifications set to on.');
        await runCommandExpecting(channelsPage, '/gitlab settings notifications maybe', 'Invalid value. Accepted values are: "on" or "off".');
        await runCommandExpecting(channelsPage, '/gitlab settings colors on', 'Unknown setting.');
        await runCommandExpecting(channelsPage, '/gitlab settings', 'Please specify both a setting and value.');
    });

    test('lists the to-do summary', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);

        const summary = await runCommandExpecting(channelsPage, '/gitlab todo', 'To-Do list');
        await expect(summary).toContainText('Review Requests');
        await expect(summary).toContainText('Merge Requests Assigned');
    });

    test('reports usage errors for pipeline and webhook commands', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, '/gitlab pipelines', 'Invalid pipelines command. Available commands are run, list');
        await runCommandExpecting(channelsPage, '/gitlab pipelines run e2e-missing/repo', 'Please specify a repository and a branch.');
        await runCommandExpecting(channelsPage, '/gitlab webhook add', UNKNOWN_ACTION);
        await runCommandExpecting(channelsPage, '/gitlab issue', "Invalid issue command. Available command is 'create'.");
        await runCommandExpecting(channelsPage, '/gitlab issue close', 'This command is not implemented yet. Command: close');
    });
});
