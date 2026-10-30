// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {expect} from '@mattermost/playwright-lib';
import type {Locator, Page} from '@playwright/test';

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

    async selectProject(path: string) {
        await this.select(this.project, path);
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
