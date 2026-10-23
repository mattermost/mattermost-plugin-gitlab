// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {loadCredentials} from '../support/credentials';
import {connectAccount, disconnectAccount, openSubscriptionsSidebar, openTownSquare} from '../support/ui';

test('connects a regular member through the GitLab OAuth popup', async ({pw}) => {
    const credentials = loadCredentials();
    const channelsPage = await openTownSquare(pw);
    await openSubscriptionsSidebar(channelsPage.page);

    if (await channelsPage.page.getByTestId('gitlab-rhs-username').isVisible().catch(() => false)) {
        await disconnectAccount(channelsPage);
    }

    await connectAccount(channelsPage.page);
    await expect(channelsPage.page.getByTestId('gitlab-rhs-username')).toHaveText(credentials.e2e_username);
});
