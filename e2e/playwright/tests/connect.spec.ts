// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {GitlabRhs} from '../support/components';
import {loadCredentials} from '../support/credentials';
import {disconnectAccount, openTownSquare} from '../support/ui';

/**
 * @objective Verify a regular member can link their GitLab account through the OAuth popup
 *
 * @precondition
 * The GitLab OAuth application is marked trusted, so GitLab skips the consent screen
 */
test('connects a regular member to GitLab through the OAuth popup in the subscriptions sidebar', {tag: '@gitlab_plugin'}, async ({pw}) => {
    // # Log in as a regular member and open the GitLab subscriptions sidebar
    const credentials = loadCredentials();
    const channelsPage = await openTownSquare(pw);
    const rhs = new GitlabRhs(channelsPage);
    await rhs.open();

    // # Disconnect first if an earlier run left the member connected
    if (await rhs.isConnected()) {
        await disconnectAccount(channelsPage);
    }

    // # Connect through the GitLab sign-in popup
    await rhs.connect(credentials);

    // * Verify the sidebar shows the connected GitLab username
    await expect(rhs.username).toHaveText(credentials.e2e_username);
});
