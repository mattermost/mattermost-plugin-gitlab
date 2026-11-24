// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import './support/env';

import {defineConfig} from '@playwright/test';
import {testConfig} from '@mattermost/playwright-lib';

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
    projects: [
        {
            name: 'chrome',
            use: {browserName: 'chromium'},
        },
    ],
});
