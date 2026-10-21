// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';

import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {
    addWebhook,
    ensureConnected,
    openSubscriptionsSidebar,
    postWithText,
    runCommandExpecting,
    safeUnsubscribe,
    subscribe,
} from '../support/ui';

test.describe('subscriptions', () => {
    let sandbox: Sandbox | undefined;
    let channelsPage: ChannelsPage | undefined;

    test.afterEach(async () => {
        if (sandbox) {
            await safeUnsubscribe(channelsPage, sandbox.path);
        }
        await destroySandbox(sandbox);
        sandbox = undefined;
        channelsPage = undefined;
    });

    test('adds, lists, and deletes a project subscription', async ({pw}) => {
        sandbox = await createSandbox();
        const {path} = sandbox;
        channelsPage = await ensureConnected(pw);
        const page = channelsPage.page;
        await openSubscriptionsSidebar(page);
        await expect(page.getByTestId('gitlab-rhs-empty')).toBeVisible();

        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path}`, 'A Webhook is needed');
        await expect(page.getByTestId('gitlab-rhs-subscription').filter({hasText: path})).toBeVisible();

        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', 'Subscriptions in this channel');
        await expect(postWithText(page, `${path} - merges,issues,tag`)).toBeVisible();

        await runCommandExpecting(channelsPage, `/gitlab subscriptions delete ${path}`, 'Successfully deleted subscription');
        await expect(page.getByTestId('gitlab-rhs-empty')).toBeVisible();

        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', 'Currently there are no subscriptions in this channel');
    });

    test('stores custom features and shows them in the sidebar', async ({pw}) => {
        sandbox = await createSandbox();
        const {path} = sandbox;
        const features = 'issues,merges,pushes,pipeline,label:"bug"';
        channelsPage = await ensureConnected(pw);
        await openSubscriptionsSidebar(channelsPage.page);

        await subscribe(channelsPage, path, features);
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', `${path} - ${features}`);

        const row = channelsPage.page.getByTestId('gitlab-rhs-subscription').filter({hasText: path});
        for (const feature of ['issues', 'merges', 'pushes', 'pipeline']) {
            // eslint-disable-next-line no-await-in-loop
            await expect(row.getByText(feature, {exact: true})).toBeVisible();
        }

        // Subscribing again replaces the feature list rather than adding a second subscription.
        await subscribe(channelsPage, path, 'tag');
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', `${path} - tag`);
        await expect(channelsPage.page.getByTestId('gitlab-rhs-subscription').filter({hasText: path})).toHaveCount(1);
    });

    test('says when a webhook already exists for the subscription', async ({pw}) => {
        sandbox = await createSandbox();
        const {path} = sandbox;
        channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, `/gitlab webhook list ${path}`, `No webhooks found in ${path}`);
        await addWebhook(channelsPage, path);
        await runCommandExpecting(channelsPage, `/gitlab webhook list ${path}`, 'Merge Request Events');

        await subscribe(channelsPage, path);
        await expect(postWithText(channelsPage.page, `Successfully subscribed to ${path}`)).not.toContainText('A Webhook is needed');
    });

    test('rejects invalid features and label filters without issues or merges', async ({pw}) => {
        sandbox = await createSandbox();
        const {path} = sandbox;
        channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path} issues,bogus`, 'unknown features bogus');
        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path} pushes,label:"bug"`, "label filters require 'merges' or 'issues' feature");
        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path} issues,label:bug`, 'each label must be wrapped in quotes');
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', 'Currently there are no subscriptions in this channel');
    });

    test('rejects projects the connected user cannot see', async ({pw}) => {
        sandbox = await createSandbox('none');
        channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${sandbox.path}`, 'Resource with such path is not found.');
    });

    test('rejects subscriptions and webhooks for guests', async ({pw}) => {
        sandbox = await createSandbox('guest');
        const {path} = sandbox;
        channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path}`, "You don't have the permissions to create subscriptions for this project.");
        await runCommandExpecting(channelsPage, `/gitlab webhook list ${path}`, "You don't have permission to manage webhooks");
    });

    test('reports usage errors for incomplete subscription commands', async ({pw}) => {
        channelsPage = await ensureConnected(pw);

        await runCommandExpecting(channelsPage, '/gitlab subscriptions', 'Invalid subscribe command. Available commands are add, delete, and list');
        await runCommandExpecting(channelsPage, '/gitlab subscriptions add', 'Please provide the owner[/repo]');
        await runCommandExpecting(channelsPage, '/gitlab subscriptions delete', 'Please specify a repository.');
        await runCommandExpecting(channelsPage, '/gitlab subscriptions delete e2e-missing/repo', 'Subscription not found, please check repository name.');
    });
});
