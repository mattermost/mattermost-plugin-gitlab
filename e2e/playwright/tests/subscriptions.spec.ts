// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';

import {GitlabRhs} from '../support/components';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {
    addWebhook,
    ensureConnected,
    expectPost,
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

    /**
     * @objective Verify a project subscription can be added, listed in the command reply and the sidebar, and deleted
     */
    test('adds, lists, and deletes a project subscription from slash commands and the sidebar', {tag: '@gitlab_plugin'}, async ({pw}) => {
        sandbox = await createSandbox();
        const {path} = sandbox;

        // # Open the GitLab subscriptions sidebar
        channelsPage = await ensureConnected(pw);
        const rhs = new GitlabRhs(channelsPage);
        await rhs.open();

        // * Verify the channel has no subscriptions
        await expect(rhs.noSubscriptions).toBeVisible();

        // # Subscribe to the project
        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path}`, 'A Webhook is needed');

        // * Verify the sidebar shows the subscription
        await expect(rhs.subscription(path)).toBeVisible();

        // # List the subscriptions
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', 'Subscriptions in this channel');

        // * Verify the reply shows the project with the default features
        await expectPost(channelsPage.page, `${path} - merges,issues,tag`);

        // # Delete the subscription
        await runCommandExpecting(channelsPage, `/gitlab subscriptions delete ${path}`, 'Successfully deleted subscription');

        // * Verify the sidebar and the list reply show no subscriptions
        await expect(rhs.noSubscriptions).toBeVisible();
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', 'Currently there are no subscriptions in this channel');
    });

    /**
     * @objective Verify a subscription keeps custom features, shows them in the sidebar, and is replaced by a later subscription
     */
    test('stores custom subscription features and shows them in the sidebar', {tag: '@gitlab_plugin'}, async ({pw}) => {
        sandbox = await createSandbox();
        const {path} = sandbox;
        const features = 'issues,merges,pushes,pipeline,label:"bug"';

        // # Open the GitLab subscriptions sidebar and subscribe with custom features
        channelsPage = await ensureConnected(pw);
        const rhs = new GitlabRhs(channelsPage);
        await rhs.open();
        await subscribe(channelsPage, path, features);

        // * Verify the list reply and the sidebar show each feature
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', `${path} - ${features}`);
        for (const feature of ['issues', 'merges', 'pushes', 'pipeline']) {
            // eslint-disable-next-line no-await-in-loop
            await expect(rhs.subscriptionFeature(path, feature)).toBeVisible();
        }

        // # Subscribe again with a different feature
        await subscribe(channelsPage, path, 'tag');

        // * Verify the feature list is replaced rather than a second subscription added
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', `${path} - tag`);
        await expect(rhs.subscription(path)).toHaveCount(1);
    });

    /**
     * @objective Verify subscribing does not ask for a webhook when the project already has one
     */
    test('skips the webhook prompt when subscribing to a project that already has a webhook', {tag: '@gitlab_plugin'}, async ({pw}) => {
        sandbox = await createSandbox();
        const {path} = sandbox;
        channelsPage = await ensureConnected(pw);

        // # List the project's webhooks before and after adding one
        // * Verify the list is empty and then includes the plugin webhook
        await runCommandExpecting(channelsPage, `/gitlab webhook list ${path}`, `No webhooks found in ${path}`);
        await addWebhook(channelsPage, path);
        await runCommandExpecting(channelsPage, `/gitlab webhook list ${path}`, 'Merge Request Events');

        // # Subscribe to the project
        await subscribe(channelsPage, path);

        // * Verify the reply does not ask for a webhook
        await postWithText(channelsPage.page, `Successfully subscribed to ${path}`).toNotContainText('A Webhook is needed');
    });

    /**
     * @objective Verify subscriptions with unknown features or misused label filters are rejected
     */
    test('rejects unknown features and label filters without issues or merges', {tag: '@gitlab_plugin'}, async ({pw}) => {
        sandbox = await createSandbox();
        const {path} = sandbox;
        channelsPage = await ensureConnected(pw);

        // # Subscribe with an unknown feature, a label filter without issues or merges, and an unquoted label
        // * Verify each is rejected with its validation error
        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path} issues,bogus`, 'unknown features bogus');
        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path} pushes,label:"bug"`, "label filters require 'merges' or 'issues' feature");
        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path} issues,label:bug`, 'each label must be wrapped in quotes');

        // * Verify no subscription was created
        await runCommandExpecting(channelsPage, '/gitlab subscriptions list', 'Currently there are no subscriptions in this channel');
    });

    /**
     * @objective Verify a project the connected GitLab user cannot see cannot be subscribed to
     *
     * @precondition
     * The connected GitLab user is not a member of the new project
     */
    test('rejects subscriptions to projects the connected user cannot see', {tag: '@gitlab_plugin'}, async ({pw}) => {
        sandbox = await createSandbox('none');
        channelsPage = await ensureConnected(pw);

        // # Subscribe to the project
        // * Verify the project is reported as not found
        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${sandbox.path}`, 'Resource with such path is not found.');
    });

    /**
     * @objective Verify a GitLab guest cannot create subscriptions or manage webhooks for a project
     *
     * @precondition
     * The connected GitLab user is a guest of the new project's group
     */
    test('rejects subscriptions and webhook commands from project guests', {tag: '@gitlab_plugin'}, async ({pw}) => {
        sandbox = await createSandbox('guest');
        const {path} = sandbox;
        channelsPage = await ensureConnected(pw);

        // # Subscribe to the project and list its webhooks
        // * Verify both are refused for lack of permissions
        await runCommandExpecting(channelsPage, `/gitlab subscriptions add ${path}`, "You don't have the permissions to create subscriptions for this project.");
        await runCommandExpecting(channelsPage, `/gitlab webhook list ${path}`, "You don't have permission to manage webhooks");
    });

    /**
     * @objective Verify incomplete subscription commands reply with usage errors
     */
    test('reports usage errors for incomplete subscription commands', {tag: '@gitlab_plugin'}, async ({pw}) => {
        channelsPage = await ensureConnected(pw);

        // # Run subscription commands without a subcommand or repository, and delete an unknown subscription
        // * Verify each replies with its usage error
        await runCommandExpecting(channelsPage, '/gitlab subscriptions', 'Invalid subscribe command. Available commands are add, delete, and list');
        await runCommandExpecting(channelsPage, '/gitlab subscriptions add', 'Please provide the owner[/repo]');
        await runCommandExpecting(channelsPage, '/gitlab subscriptions delete', 'Please specify a repository.');
        await runCommandExpecting(channelsPage, '/gitlab subscriptions delete e2e-missing/repo', 'Subscription not found, please check repository name.');
    });
});
