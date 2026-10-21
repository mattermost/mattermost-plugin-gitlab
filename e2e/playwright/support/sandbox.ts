// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {loadCredentials, type GitlabCredentials} from './credentials';
import {GitlabApi, uniquePath, type AccessLevel, type GitlabGroup, type GitlabProject} from './gitlab_api';

// A private group and project owned by root. The connected GitLab user joins the
// group with the given access, or not at all when access is 'none'.
export type Sandbox = {
    credentials: GitlabCredentials;
    root: GitlabApi;
    user: GitlabApi;
    group: GitlabGroup;
    project: GitlabProject;
    path: string;
};

export async function createSandbox(access: AccessLevel | 'none' = 'owner'): Promise<Sandbox> {
    const credentials = loadCredentials();
    const root = new GitlabApi(credentials);
    const group = await root.createGroup(uniquePath('e2e'));
    const project = await root.createProject(group.id, 'repo');
    if (access !== 'none') {
        await root.addMember(group.id, credentials.e2e_user_id, access);
    }
    return {
        credentials,
        root,
        user: GitlabApi.asConnectedUser(credentials),
        group,
        project,
        path: project.path_with_namespace,
    };
}

export async function destroySandbox(sandbox: Sandbox | undefined) {
    if (!sandbox) {
        return;
    }
    await sandbox.root.closeOpenItems(sandbox.project.id);
    await sandbox.root.deleteGroup(sandbox.group.id);
}
