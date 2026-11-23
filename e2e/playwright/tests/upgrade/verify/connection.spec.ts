// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {loginAdmin} from '../../../support/admin';
import {GitlabRhs} from '../../../support/components';
import {GITLAB_URL, INSTANCE_NAME, PLUGIN_ID} from '../../../support/constants';
import {loadCredentials} from '../../../support/credentials';
import {openTownSquare, runCommandExpecting} from '../../../support/ui';

test.describe('connection and settings after the upgrade', () => {
    /**
     * @objective Verify a member connected on the base release stays connected and their stored token still works
     *
     * @precondition
     * The member connected through OAuth on the base release, and the plugin was then upgraded to the local build
     */
    test('keeps the base release connection and its token working', {tag: '@upgrade'}, async ({pw}) => {
        const credentials = loadCredentials();

        // # Open the GitLab sidebar as the member, without reconnecting
        const channelsPage = await openTownSquare(pw);
        const rhs = new GitlabRhs(channelsPage);
        await rhs.open();

        // * Verify the member is still connected as the same GitLab user
        await expect(rhs.username).toHaveText(credentials.e2e_username);

        // * Verify /gitlab me reports the same GitLab account
        const profile = await runCommandExpecting(channelsPage, '/gitlab me', 'You are connected to GitLab as:');
        await profile.toContainText(credentials.e2e_username);

        // * Verify a command that calls GitLab with the stored token succeeds, so the token still decrypts
        const todos = await runCommandExpecting(channelsPage, '/gitlab todo', 'To-Do list');
        await todos.toContainText('Review Requests');
    });

    /**
     * @objective Verify the plugin settings and the default instance set up on the base release carry over the upgrade
     *
     * @precondition
     * The base release was configured and its instance installed through the wizard, and the upgrade did not reconfigure the plugin
     */
    test('keeps the base release settings and default instance', {tag: '@upgrade'}, async ({pw}) => {
        const credentials = loadCredentials();
        const {client} = await loginAdmin();

        // # Read the plugin settings
        const config = await client.getConfig();
        const settings = config.PluginSettings.Plugins?.[PLUGIN_ID] as Record<string, unknown> | undefined;

        // * Verify the settings from the base release are unchanged
        expect(settings?.gitlaburl).toBe(credentials.gitlab_url);
        expect(settings?.gitlaboauthclientid).toBe(credentials.client_id);
        expect(settings?.enableprivaterepo).toBe(true);
        expect(settings?.defaultinstancename).toBe(INSTANCE_NAME);

        // * Verify the secrets are still set. The API masks their values, so the connection and
        //   webhook delivery checks are what prove the encryption key and webhook secret carried over.
        expect(settings?.encryptionkey).toBeTruthy();
        expect(settings?.webhooksecret).toBeTruthy();

        // # List the installed instances as the sysadmin
        const adminPage = await openTownSquare(pw, {asAdmin: true});
        const instances = await runCommandExpecting(adminPage, '/gitlab instance list', 'Installed GitLab Instances');

        // * Verify the instance installed on the base release is still listed
        await instances.toContainText(INSTANCE_NAME);
        await instances.toContainText(GITLAB_URL);
    });
});
