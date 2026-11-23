// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect, test} from '@mattermost/playwright-lib';

import {uniquePath} from '../support/gitlab_api';
import {createSandbox, destroySandbox, type Sandbox} from '../support/sandbox';
import {ensureConnected, postWithText} from '../support/ui';

// The code-preview feature (MessageWillBePosted) expands GitLab blob permalinks in a
// post into an inline code block. It cannot run against this stack as the plugin ships:
//   - gitlabPermalinkRegex in server/plugin.go only matches the literal host
//     "gitlab.com", so a self-hosted gitlab.test permalink is never recognized.
//   - EnableCodePreview defaults to "public", which skips private projects, and the
//     sandbox projects are private.
// The first is a product bug, to be tracked in a separate ticket: code previews never
// work for self-hosted GitLab. The test is kept as fixme to document the intended
// behavior. Enabling it needs that fix plus a public project or EnableCodePreview
// relaxed for the run.
test.describe('code preview permalinks', () => {
    let sandbox: Sandbox;

    test.beforeEach(async () => {
        sandbox = await createSandbox();
    });

    test.afterEach(async () => {
        await destroySandbox(sandbox);
    });

    /**
     * @objective Verify a GitLab blob permalink in a post is expanded into an inline code preview
     *
     * @precondition
     * A new project the connected user owns has a file to preview
     */
    test.fixme('expands a GitLab blob permalink into a code preview', {tag: '@gitlab_plugin'}, async ({pw}) => {
        const {root, project} = sandbox;
        const branch = project.default_branch;
        const fileName = `${uniquePath('preview')}.md`;
        const previewLine = `preview line ${uniquePath('')}`;

        // # Commit a file whose contents the preview should show
        await root.commitFile(project.id, branch, fileName, `${previewLine}\nsecond line\n`, 'Add preview file');

        // # Post a blob permalink pointing at the first line of the file
        const channelsPage = await ensureConnected(pw);
        const permalink = `${project.web_url}/-/blob/${branch}/${fileName}#L1`;
        await channelsPage.centerView.postMessage(permalink);

        // * Verify the post is replaced with a code preview of the referenced line
        await expect(postWithText(channelsPage.page, previewLine).container).toBeVisible();
    });
});
