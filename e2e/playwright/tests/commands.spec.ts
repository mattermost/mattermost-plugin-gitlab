// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {GITLAB_URL, INSTANCE_NAME} from '../support/constants';
import {loadCredentials} from '../support/credentials';
import {ensureConnected, runCommandExpecting} from '../support/ui';

const UNKNOWN_ACTION = 'Unknown action, please use /gitlab help to see all actions available.';

test.describe('slash commands', () => {
    test('shows help and build information', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, '/gitlab help', 'Mattermost GitLab Plugin - Slash Command Help');
        await runCommandExpecting(channelsPage, '/gitlab about', /GitLab version/i);
        await runCommandExpecting(channelsPage, '/gitlab unknown-action', UNKNOWN_ACTION);
    });

    test('shows the connected GitLab profile and the installed instance', async ({pw}) => {
        const credentials = loadCredentials();
        const channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, '/gitlab me', 'You are connected to GitLab as:');
        await expect(channelsPage.page.locator('.post-message__text').last()).toContainText(credentials.e2e_username);

        await runCommandExpecting(channelsPage, '/gitlab instance list', 'Installed GitLab Instances');
        const instances = channelsPage.page.locator('.post-message__text').last();
        await expect(instances).toContainText(INSTANCE_NAME);
        await expect(instances).toContainText(GITLAB_URL);

        await runCommandExpecting(channelsPage, '/gitlab connect', 'Click here to link your GitLab account.');
    });

    test('updates personal settings and rejects invalid values', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, '/gitlab settings reminders off', 'Settings updated.');
        await runCommandExpecting(channelsPage, '/gitlab settings reminders on', 'Settings updated.');
        await runCommandExpecting(channelsPage, '/gitlab settings notifications off', 'Settings updated.');
        await runCommandExpecting(channelsPage, '/gitlab settings notifications on', 'Settings updated.');
        await runCommandExpecting(channelsPage, '/gitlab settings notifications maybe', 'Invalid value. Accepted values are: "on" or "off".');
        await runCommandExpecting(channelsPage, '/gitlab settings colors on', 'Unknown setting.');
        await runCommandExpecting(channelsPage, '/gitlab settings', 'Please specify both a setting and value.');
    });

    test('lists the to-do summary', async ({pw}) => {
        const channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, '/gitlab todo', 'To-Do list');
        const summary = channelsPage.page.locator('.post-message__text').last();
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
