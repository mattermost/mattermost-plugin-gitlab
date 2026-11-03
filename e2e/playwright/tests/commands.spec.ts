// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test} from '@mattermost/playwright-lib';

import {GITLAB_URL, INSTANCE_NAME} from '../support/constants';
import {loadCredentials} from '../support/credentials';
import {ensureConnected, openTownSquare, runCommandExpecting} from '../support/ui';

const UNKNOWN_ACTION = 'Unknown action, please use /gitlab help to see all actions available.';

test.describe('slash commands', () => {
    /**
     * @objective Verify /gitlab help, about, and unknown actions reply with usage information
     */
    test('shows help, build information, and the unknown action reply', {tag: '@gitlab_plugin'}, async ({pw}) => {
        // # Log in as a connected member and open Town Square
        const channelsPage = await ensureConnected(pw);

        // # Run /gitlab help, /gitlab about, and an unknown action
        // * Verify each command replies with its help, version, or usage text
        await runCommandExpecting(channelsPage, '/gitlab help', 'Mattermost GitLab Plugin - Slash Command Help');
        await runCommandExpecting(channelsPage, '/gitlab about', /GitLab version/i);
        await runCommandExpecting(channelsPage, '/gitlab unknown-action', UNKNOWN_ACTION);
    });

    /**
     * @objective Verify /gitlab me and /gitlab connect report the linked GitLab account
     */
    test('shows the connected GitLab profile from /gitlab me and /gitlab connect', {tag: '@gitlab_plugin'}, async ({pw}) => {
        // # Log in as a connected member and open Town Square
        const credentials = loadCredentials();
        const channelsPage = await ensureConnected(pw);

        // # Run /gitlab me
        const profile = await runCommandExpecting(channelsPage, '/gitlab me', 'You are connected to GitLab as:');

        // * Verify the reply names the connected GitLab user
        await profile.toContainText(credentials.e2e_username);

        // # Run /gitlab connect while already connected
        const alreadyConnected = await runCommandExpecting(channelsPage, '/gitlab connect', 'You are already connected to GitLab as');

        // * Verify the reply names the connected GitLab user instead of offering a new connection
        await alreadyConnected.toContainText(credentials.e2e_username);
    });

    /**
     * @objective Verify /gitlab instance commands are limited to system admins
     *
     * @precondition
     * The gitlab-e2e instance was installed by global setup
     */
    test('restricts /gitlab instance list to system admins', {tag: '@gitlab_plugin'}, async ({pw}) => {
        // # Run /gitlab instance list as a regular member
        const memberPage = await ensureConnected(pw);

        // * Verify the member is refused
        await runCommandExpecting(memberPage, '/gitlab instance list', 'Only System Admins are allowed to manage instances.');

        // # Run /gitlab instance list as the sysadmin
        const adminPage = await openTownSquare(pw, {asAdmin: true});
        const instances = await runCommandExpecting(adminPage, '/gitlab instance list', 'Installed GitLab Instances');

        // * Verify the installed instance and its URL are listed
        await instances.toContainText(INSTANCE_NAME);
        await instances.toContainText(GITLAB_URL);
    });

    /**
     * @objective Verify /gitlab settings toggles reminders and notifications and rejects invalid input
     */
    test('updates personal settings and rejects invalid values', {tag: '@gitlab_plugin'}, async ({pw}) => {
        // # Log in as a connected member and open Town Square
        const channelsPage = await ensureConnected(pw);

        // # Toggle reminders and notifications off and on
        // * Verify each change is confirmed with the setting name and value
        await runCommandExpecting(channelsPage, '/gitlab settings reminders off', 'Settings updated: reminders set to off.');
        await runCommandExpecting(channelsPage, '/gitlab settings reminders on', 'Settings updated: reminders set to on.');
        await runCommandExpecting(channelsPage, '/gitlab settings notifications off', 'Settings updated: notifications set to off.');
        await runCommandExpecting(channelsPage, '/gitlab settings notifications on', 'Settings updated: notifications set to on.');

        // # Run /gitlab settings with an invalid value, an unknown setting, and no arguments
        // * Verify each is rejected with a usage error
        await runCommandExpecting(channelsPage, '/gitlab settings notifications maybe', 'Invalid value. Accepted values are: "on" or "off".');
        await runCommandExpecting(channelsPage, '/gitlab settings colors on', 'Unknown setting.');
        await runCommandExpecting(channelsPage, '/gitlab settings', 'Please specify both a setting and value.');
    });

    /**
     * @objective Verify /gitlab todo replies with the to-do summary sections
     */
    test('lists the to-do summary from /gitlab todo', {tag: '@gitlab_plugin'}, async ({pw}) => {
        // # Log in as a connected member and run /gitlab todo
        const channelsPage = await ensureConnected(pw);
        const summary = await runCommandExpecting(channelsPage, '/gitlab todo', 'To-Do list');

        // * Verify the summary includes the review and assigned merge request sections
        await summary.toContainText('Review Requests');
        await summary.toContainText('Merge Requests Assigned');
    });

    /**
     * @objective Verify incomplete pipeline, webhook, and issue commands reply with usage errors
     */
    test('reports usage errors for incomplete pipeline, webhook, and issue commands', {tag: '@gitlab_plugin'}, async ({pw}) => {
        // # Log in as a connected member and open Town Square
        const channelsPage = await ensureConnected(pw);

        // # Run pipeline, webhook, and issue commands with missing or unsupported arguments
        // * Verify each replies with its usage error
        await runCommandExpecting(channelsPage, '/gitlab pipelines', 'Invalid pipelines command. Available commands are run, list');
        await runCommandExpecting(channelsPage, '/gitlab pipelines run e2e-missing/repo', 'Please specify a repository and a branch.');
        await runCommandExpecting(channelsPage, '/gitlab webhook add', UNKNOWN_ACTION);
        await runCommandExpecting(channelsPage, '/gitlab issue', "Invalid issue command. Available command is 'create'.");
        await runCommandExpecting(channelsPage, '/gitlab issue close', 'This command is not implemented yet. Command: close');
    });
});
