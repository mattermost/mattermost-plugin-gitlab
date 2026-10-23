// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fs from 'node:fs';
import path from 'node:path';

export type GitlabCredentials = {
    gitlab_url: string;
    client_id: string;
    client_secret: string;
    admin_token: string;
    e2e_token: string;
    runner_token: string;
    e2e_user_id: number;
    e2e_username: string;
    e2e_password: string;
    root_user_id: number;
};

const CREDENTIALS_PATH = path.resolve(__dirname, '../.gitlab-credentials.json');

export function loadCredentials(): GitlabCredentials {
    if (!fs.existsSync(CREDENTIALS_PATH)) {
        throw new Error(`Missing ${CREDENTIALS_PATH}. Run \`make e2e-server-up\` and the global setup bootstrap first.`);
    }
    return JSON.parse(fs.readFileSync(CREDENTIALS_PATH, 'utf8')) as GitlabCredentials;
}
