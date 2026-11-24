// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './support/env';

import {defineConfig, type Project} from '@playwright/test';
import {testConfig} from '@mattermost/playwright-lib';

const chromium = {browserName: 'chromium'} as const;

// E2E_UPGRADE_FROM switches to the upgrade run: install the base release and seed
// data through it, upgrade to dist/, then verify that data on the new build. The
// three projects run in order, and the normal run never picks up tests/upgrade/.
// upgrade-cleanup is the seed's teardown: it runs once the other three finish, even
// when one fails, so a failed check never removes data a later check needs.
// Retries are off so a retried seed cannot leave a second set of seeded data behind.
const projects: Project[] = process.env.E2E_UPGRADE_FROM ? [
    {name: 'upgrade-seed', testMatch: /upgrade\/seed\.spec\.ts/, retries: 0, use: chromium, teardown: 'upgrade-cleanup'},
    {name: 'upgrade-apply', testMatch: /upgrade\/apply\.spec\.ts/, retries: 0, use: chromium, dependencies: ['upgrade-seed']},
    {name: 'upgrade-verify', testMatch: /upgrade\/verify\/.*\.spec\.ts/, retries: 0, use: chromium, dependencies: ['upgrade-apply']},
    {name: 'upgrade-cleanup', testMatch: /upgrade\/cleanup\.teardown\.ts/, retries: 0, use: chromium},
] : [
    {name: 'chrome', testIgnore: /upgrade\//, use: chromium},
];

// One GitLab user and one set of subscriptions, so the suite stays serial.
export default defineConfig({
    testDir: './tests',
    globalSetup: require.resolve('./global_setup'),
    outputDir: './results/output',
    fullyParallel: false,
    workers: 1,
    retries: process.env.CI ? 1 : 0,
    timeout: 2 * 60 * 1000,
    expect: {timeout: 15 * 1000},
    reporter: [
        ['list'],
        ['html', {outputFolder: './results/reporter', open: 'never'}],
        ...(process.env.CI ? [['github'] as const] : []),
    ],
    use: {
        baseURL: testConfig.baseURL,
        headless: testConfig.headless,
        launchOptions: {slowMo: testConfig.slowMo},
        viewport: {width: 1280, height: 1024},
        trace: 'retain-on-failure',
        screenshot: 'only-on-failure',
        video: 'retain-on-failure',
    },
    projects,
});
