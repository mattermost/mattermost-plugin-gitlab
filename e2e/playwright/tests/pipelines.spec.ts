// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test, type ChannelsPage} from '@mattermost/playwright-lib';
import type {Page} from '@playwright/test';

import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {
    addWebhook,
    ensureConnected,
    openBotDirectMessage,
    postWithText,
    runCommandExpecting,
    runSlashCommand,
    safeUnsubscribe,
    subscribe,
} from '../support/ui';

const FAILING_BRANCH = 'failing';

// Runs on the gitlab-runner service. The build job fails on FAILING_BRANCH, and
// the deploy job creates a deployment to the production environment.
const CI_CONFIG = `stages: [build, deploy]
build:
  stage: build
  script:
    - test "$CI_COMMIT_REF_NAME" != "${FAILING_BRANCH}"
deploy:
  stage: deploy
  script:
    - echo deploying
  environment: production
`;

const PIPELINE_TIMEOUT = 150000;

test.describe('pipelines', () => {
    let sandbox: Sandbox;
    let channelsPage: ChannelsPage;

    test.beforeEach(async ({pw}) => {
        test.setTimeout(6 * 60 * 1000);
        sandbox = await createSandbox();
        channelsPage = await ensureConnected(pw);
        await addWebhook(channelsPage, sandbox.path);
    });

    test.afterEach(async () => {
        await safeUnsubscribe(channelsPage, sandbox?.path);
        await destroySandbox(sandbox);
    });

    test('runs pipelines from a command and posts pipeline, job, and deployment events', async () => {
        const {project, path, credentials} = sandbox;
        const branch = project.default_branch;
        const username = credentials.e2e_username;
        const page = channelsPage.page;

        await runCommandExpecting(channelsPage, `/gitlab pipelines run ${path} ${branch}`, 'failed to run pipeline for Project');
        await addCiConfig(sandbox);

        const unsubscribedReply = await runPipeline(channelsPage, path, branch);
        await expect(unsubscribedReply).toContainText('This channel is currently not subscribed to pipeline event');

        await subscribe(channelsPage, path, 'pipeline,jobs,deployments');
        const subscribedReply = await runPipeline(channelsPage, path, branch);
        await expect(subscribedReply).not.toContainText('not subscribed to pipeline event');

        await expectPipelinePost(page, 'New pipeline from', `by ${username}`);
        await expectPipelinePost(page, 'Pipeline Job Stage: build', 'Status: success');
        await expectPipelinePost(page, 'Deployment Stage: success');
        await expectPipelinePost(page, `Pipeline by ${username} success for`, 'Add CI configuration');

        await runPipeline(channelsPage, path, FAILING_BRANCH);
        await expectPipelinePost(page, 'Pipeline Job Stage: build', 'Status: failed');
        await expectPipelinePost(page, `Pipeline by ${username} fail for`);
    });

    // The plugin sends the event sender a refresh before dispatching DMs and then
    // skips users who already got a refresh, so the sender never receives this DM.
    test.fixme('sends the pipeline author a DM when their pipeline fails', async () => {
        const {path} = sandbox;
        await addCiConfig(sandbox);
        await runPipeline(channelsPage, path, FAILING_BRANCH);

        await openBotDirectMessage(channelsPage.page);
        await expectPipelinePost(channelsPage.page, `${path} Your pipeline has failed for`);
    });
});

// Pipeline webhooks can land between the command and its reply, so the reply is
// found by counting "Pipeline info" posts rather than taking the newest post.
async function runPipeline(channelsPage: ChannelsPage, path: string, ref: string) {
    const replies = channelsPage.page.locator('.post-message__text').filter({hasText: 'Pipeline info'});
    const before = await replies.count();
    await runSlashCommand(channelsPage, `/gitlab pipelines run ${path} ${ref}`);
    await expect(replies).toHaveCount(before + 1);
    return replies.last();
}

async function addCiConfig({root, project}: Sandbox) {
    await root.commitFile(project.id, project.default_branch, '.gitlab-ci.yml', CI_CONFIG, 'Add CI configuration');
    await root.createBranch(project.id, FAILING_BRANCH, project.default_branch);
}

async function expectPipelinePost(page: Page, ...texts: string[]) {
    await expect(postWithText(page, ...texts)).toBeVisible({timeout: PIPELINE_TIMEOUT});
}
