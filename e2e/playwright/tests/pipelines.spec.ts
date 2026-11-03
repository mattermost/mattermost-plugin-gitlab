// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test, type ChannelsPage} from '@mattermost/playwright-lib';

import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {
    addWebhook,
    ensureConnected,
    expectPostWithin,
    openBotDirectMessage,
    runCommandExpecting,
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

    /**
     * @objective Verify /gitlab pipelines run starts pipelines, and a subscribed channel gets pipeline, job, and deployment posts
     *
     * @precondition
     * The gitlab-runner service is registered, and a new project has a plugin webhook
     */
    test('runs pipelines from /gitlab pipelines run and posts pipeline, job, and deployment events', {tag: '@gitlab_plugin'}, async () => {
        const {project, path, credentials} = sandbox;
        const branch = project.default_branch;
        const username = credentials.e2e_username;

        // # Run a pipeline before the project has CI configuration
        // * Verify the command reports the pipeline could not run
        await runCommandExpecting(channelsPage, `/gitlab pipelines run ${path} ${branch}`, 'failed to run pipeline for Project');

        // # Add CI configuration and run a pipeline without a pipeline subscription
        await addCiConfig(sandbox);
        const unsubscribedReply = await runPipeline(channelsPage, path, branch);

        // * Verify the reply warns the channel is not subscribed to pipeline events
        await unsubscribedReply.toContainText('This channel is currently not subscribed to pipeline event');

        // # Subscribe to pipeline, job, and deployment events and run a pipeline
        await subscribe(channelsPage, path, 'pipeline,jobs,deployments');
        const subscribedReply = await runPipeline(channelsPage, path, branch);

        // * Verify the reply no longer warns and the successful run is posted
        await subscribedReply.toNotContainText('not subscribed to pipeline event');
        await expectPipelinePost(channelsPage, 'New pipeline from', `by ${username}`);
        await expectPipelinePost(channelsPage, 'Pipeline Job Stage: build', 'Status: success');
        await expectPipelinePost(channelsPage, 'Deployment Stage: success');
        await expectPipelinePost(channelsPage, `Pipeline by ${username} success for`, 'Add CI configuration');

        // # Run a pipeline on the branch whose build fails
        await runPipeline(channelsPage, path, FAILING_BRANCH);

        // * Verify the failed job and pipeline are posted
        await expectPipelinePost(channelsPage, 'Pipeline Job Stage: build', 'Status: failed');
        await expectPipelinePost(channelsPage, `Pipeline by ${username} fail for`);
    });

    /**
     * @objective Verify the pipeline author gets a DM from the GitLab bot when their pipeline fails
     *
     * @precondition
     * The gitlab-runner service is registered, and a new project has a plugin webhook
     */
    // The plugin sends the event sender a refresh before dispatching DMs and then
    // skips users who already got a refresh, so the sender never receives this DM.
    // https://mattermost.atlassian.net/browse/MM-71329
    test.fixme('DMs the pipeline author when their pipeline fails', {tag: '@gitlab_plugin'}, async () => {
        const {path} = sandbox;

        // # Add CI configuration and run a pipeline on the branch whose build fails
        await addCiConfig(sandbox);
        await runPipeline(channelsPage, path, FAILING_BRANCH);

        // # Open the DM with the GitLab bot
        await openBotDirectMessage(channelsPage);

        // * Verify the failed-pipeline DM arrives
        await expectPipelinePost(channelsPage, `${path} Your pipeline has failed for`);
    });
});

async function runPipeline(channelsPage: ChannelsPage, path: string, ref: string) {
    return runCommandExpecting(channelsPage, `/gitlab pipelines run ${path} ${ref}`, 'Pipeline info');
}

async function addCiConfig({root, project}: Sandbox) {
    await root.commitFile(project.id, project.default_branch, '.gitlab-ci.yml', CI_CONFIG, 'Add CI configuration');
    await root.createBranch(project.id, FAILING_BRANCH, project.default_branch);
}

async function expectPipelinePost(channelsPage: ChannelsPage, ...texts: string[]) {
    await expectPostWithin(channelsPage.page, PIPELINE_TIMEOUT, ...texts);
}
