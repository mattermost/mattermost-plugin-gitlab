// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export const PLUGIN_ID = 'com.github.manland.mattermost-plugin-gitlab';

export const TEAM_NAME = 'gitlab-e2e';
export const TOWN_SQUARE = 'town-square';

export const GITLAB_URL = 'http://gitlab.test:8929';

// AES-128/192/256 keys must be 16, 24, or 32 bytes. This one is 32.
export const ENCRYPTION_KEY = 'e2e-gitlab-plugin-encryption-key';
export const WEBHOOK_SECRET = 'e2e-webhook-secret';

export const INSTANCE_NAME = 'gitlab-e2e';
export const BOT_USERNAME = 'gitlab';

export const WEBHOOK_TIMEOUT = 45000;

// The plugin id contains dots, so an #id selector would parse them as classes.
export const APP_BAR_ICON = `[id="app-bar-icon-${PLUGIN_ID}"]`;
