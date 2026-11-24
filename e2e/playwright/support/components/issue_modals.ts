// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect} from '@mattermost/playwright-lib';
import type {Locator, Page, Request} from '@playwright/test';

abstract class IssueModal {
    readonly container: Locator;
    readonly closeButton: Locator;

    protected constructor(protected readonly page: Page, title: string) {
        this.container = page.getByRole('dialog').filter({has: page.getByRole('heading', {name: title, exact: true})});
        this.closeButton = this.container.getByRole('button', {name: 'Close'});
    }

    async toBeVisible() {
        await expect(this.container).toBeVisible();
    }

    async toBeHidden() {
        await expect(this.container).toBeHidden();
    }

    async close() {
        await this.closeButton.click();
    }

    // The selectors are react-select inputs whose options render outside the modal.
    protected async select(combobox: Locator, text: string) {
        await combobox.fill(text);
        await this.page.getByRole('option', {name: new RegExp(escapeRegExp(text))}).click();
    }
}

export class CreateIssueModal extends IssueModal {
    readonly project: Locator;
    readonly title: Locator;
    readonly description: Locator;
    readonly submitButton: Locator;

    constructor(page: Page) {
        super(page, 'Create GitLab Issue');
        this.project = this.container.getByLabel('Project', {exact: true});
        this.title = this.container.getByLabel('Issue title', {exact: true});
        this.description = this.container.getByLabel('Issue description', {exact: true});
        this.submitButton = this.container.getByRole('button', {name: 'Submit', exact: true});
    }

    // Works around a webapp bug, to be tracked in a separate ticket: selecting a project
    // mounts the label, assignee, and milestone pickers, and each fetches its options twice
    // (both useEffects in issue_attribute_selector.tsx run). When a fetch resolves, the
    // picker re-applies the selection from the render that started it (initially [] for
    // labels and assignees), so a label or assignee picked between the two responses is
    // cleared. Wait for all option fetches to settle. Remove once the webapp is fixed.
    async selectProject(path: string) {
        const requested = new Set<string>();
        let pending = 0;
        const optionEndpoint = (request: Request) => (/\/api\/v1\/(labels|assignees|milestones)\?/).exec(request.url())?.[1];
        const onStart = (request: Request) => {
            const endpoint = optionEndpoint(request);
            if (endpoint) {
                requested.add(endpoint);
                pending++;
            }
        };
        const onEnd = (request: Request) => {
            if (optionEndpoint(request)) {
                pending--;
            }
        };
        this.page.on('request', onStart);
        this.page.on('requestfinished', onEnd);
        this.page.on('requestfailed', onEnd);
        try {
            await this.select(this.project, path);

            // Each endpoint is requested, then nothing is in flight. This holds whether a
            // picker fetches once or twice: both fetches start in the same render, before
            // either can finish, so pending cannot reach 0 between them.
            await expect.poll(() => requested.size === 3 && pending === 0).toBe(true);

            // requestfinished only means the response arrived; the picker applies it in a
            // later page task. Run one page task and a frame so arrived responses are
            // applied and rendered. The pickers' loading spinner is no signal: the two
            // loads share one flag, and it clears after the first.
            await this.page.evaluate(() => new Promise<void>((resolve) => {
                setTimeout(() => requestAnimationFrame(() => resolve()), 0);
            }));
        } finally {
            this.page.off('request', onStart);
            this.page.off('requestfinished', onEnd);
            this.page.off('requestfailed', onEnd);
        }
    }

    async selectLabel(name: string) {
        await this.chooseAttribute('Labels', name);
    }

    async selectAssignee(username: string) {
        await this.chooseAttribute('Assignees', username);
    }

    async selectMilestone(title: string) {
        await this.chooseAttribute('Milestone', title);
    }

    // The label, assignee, and milestone selectors wrap a react-select whose Setting
    // label is not tied to its input, so scope by the form group that holds the label.
    private attributeGroup(label: string): Locator {
        return this.container.locator('div.form-group.less').filter({has: this.page.getByText(label, {exact: true})});
    }

    // Picks an option with Enter. Clicking the portalled option does not commit the
    // multi-select pickers (labels, assignees); Enter works for both the multi and single
    // selectors. Escape is avoided because it clears the committed value.
    //
    // Enter commits the highlighted option, which after typing is the first filtered one
    // (a-bug when picking bug). Hovering highlights the exact option instead. react-select
    // ignores the first mouse move after a keypress, so move onto the option twice.
    // aria-activedescendant cannot confirm the highlight: react-select omits it on macOS.
    private async chooseAttribute(label: string, text: string) {
        const group = this.attributeGroup(label);
        const combobox = group.getByRole('combobox');
        await combobox.fill(text);
        const option = this.page.getByRole('option', {name: new RegExp(`^${escapeRegExp(text)}$`)});
        await option.hover({position: {x: 5, y: 5}});
        await option.hover({position: {x: 10, y: 8}});
        await combobox.press('Enter');

        // * Verify the pick is shown so a dropped selection fails here, not at submit
        await expect(group.getByText(text, {exact: true})).toBeVisible();
    }

    // Waits until the chosen label, assignee, and milestone are all shown as chips, so
    // a submit does not race the react-select state update for the earlier picks.
    async expectSelections(label: string, assignee: string, milestone: string) {
        await expect(this.attributeGroup('Labels').getByText(label, {exact: true})).toBeVisible();
        await expect(this.attributeGroup('Assignees').getByText(assignee, {exact: true})).toBeVisible();
        await expect(this.attributeGroup('Milestone').getByText(milestone, {exact: true})).toBeVisible();
    }

    async submit() {
        await this.submitButton.click();
    }
}

export class AttachCommentModal extends IssueModal {
    readonly issue: Locator;
    readonly message: Locator;
    readonly attachButton: Locator;

    constructor(page: Page) {
        super(page, 'Attach Message to GitLab Issue');
        this.issue = this.container.getByLabel('Gitlab Issue', {exact: true});
        this.message = this.container.getByLabel('Message Attached to GitLab Issue', {exact: true});
        this.attachButton = this.container.getByRole('button', {name: 'Attach', exact: true});
    }

    async selectIssue(title: string) {
        await this.select(this.issue, title);
    }

    async attach() {
        await this.attachButton.click();
    }
}

function escapeRegExp(text: string) {
    return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
