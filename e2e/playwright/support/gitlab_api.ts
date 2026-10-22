// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {GitlabCredentials} from './credentials';

const ACCESS_LEVELS = {guest: 10, owner: 50};
const POLL_INTERVAL_MS = 1000;

export type AccessLevel = keyof typeof ACCESS_LEVELS;

export type GitlabGroup = {
    id: number;
    path: string;
};

export type GitlabProject = {
    id: number;
    path: string;
    path_with_namespace: string;
    default_branch: string;
    web_url: string;
};

export type GitlabIssue = {
    id: number;
    iid: number;
    title: string;
    web_url: string;
};

export type GitlabMergeRequest = {
    id: number;
    iid: number;
    title: string;
    web_url: string;
    source_branch: string;
};

export type GitlabPipeline = {
    id: number;
    status: string;
};

export type GitlabTodo = {
    id: number;
    action_name: string;
    target: {iid: number; title: string};
};

type IssueOptions = {
    assigneeIds?: number[];
    labels?: string[];
    confidential?: boolean;
    description?: string;
};

type MergeRequestOptions = {
    targetBranch: string;
    assigneeIds?: number[];
    reviewerIds?: number[];
    description?: string;
};

export class GitlabApi {
    private readonly baseURL: string;

    private readonly token: string;

    constructor(credentials: GitlabCredentials, token = credentials.admin_token) {
        this.baseURL = credentials.gitlab_url.replace(/\/$/, '');
        this.token = token;
    }

    // Acts as the GitLab user connected in Mattermost, so events it triggers are sent by that user.
    static asConnectedUser(credentials: GitlabCredentials) {
        return new GitlabApi(credentials, credentials.e2e_token);
    }

    async createGroup(path: string): Promise<GitlabGroup> {
        return this.request('POST', '/groups', {
            name: path,
            path,
            visibility: 'private',
        });
    }

    async createProject(groupId: number, path: string): Promise<GitlabProject> {
        const project = await this.request<GitlabProject>('POST', '/projects', {
            name: path,
            path,
            namespace_id: groupId,
            visibility: 'private',
            initialize_with_readme: true,
        });
        project.default_branch = await this.waitForDefaultBranch(project.id);
        return project;
    }

    async addMember(groupId: number, userId: number, accessLevel: AccessLevel) {
        await this.request('POST', `/groups/${groupId}/members`, {
            user_id: userId,
            access_level: ACCESS_LEVELS[accessLevel],
        });
    }

    async createLabel(projectId: number, name: string) {
        await this.request('POST', `/projects/${projectId}/labels`, {name, color: '#428BCA'});
    }

    async createIssue(projectId: number, title: string, options: IssueOptions = {}): Promise<GitlabIssue> {
        return this.request('POST', `/projects/${projectId}/issues`, {
            title,
            description: options.description,
            assignee_ids: options.assigneeIds,
            labels: options.labels?.join(','),
            confidential: options.confidential,
        });
    }

    async updateIssue(projectId: number, iid: number, changes: Record<string, unknown>) {
        await this.request('PUT', `/projects/${projectId}/issues/${iid}`, changes);
    }

    async closeIssue(projectId: number, iid: number) {
        await this.updateIssue(projectId, iid, {state_event: 'close'});
    }

    async reopenIssue(projectId: number, iid: number) {
        await this.updateIssue(projectId, iid, {state_event: 'reopen'});
    }

    async commentOnIssue(projectId: number, iid: number, body: string) {
        await this.request('POST', `/projects/${projectId}/issues/${iid}/notes`, {body});
    }

    async listIssues(projectId: number): Promise<Array<GitlabIssue & {description: string}>> {
        return this.request('GET', `/projects/${projectId}/issues?per_page=100`);
    }

    async listIssueComments(projectId: number, iid: number): Promise<Array<{body: string; system: boolean}>> {
        return this.request('GET', `/projects/${projectId}/issues/${iid}/notes?per_page=100`);
    }

    async commitFile(projectId: number, branch: string, filePath: string, content: string, message: string, startBranch?: string) {
        await this.request('POST', `/projects/${projectId}/repository/commits`, {
            branch,
            start_branch: startBranch,
            commit_message: message,
            actions: [{action: 'create', file_path: filePath, content}],
        });
    }

    async createBranch(projectId: number, branch: string, ref: string) {
        await this.request('POST', `/projects/${projectId}/repository/branches`, {branch, ref});
    }

    async createMergeRequest(projectId: number, title: string, options: MergeRequestOptions): Promise<GitlabMergeRequest> {
        const branch = `e2e-${Date.now().toString(36)}${Math.random().toString(16).slice(2, 6)}`;
        await this.commitFile(projectId, branch, `${branch}.md`, 'e2e', `Add ${branch}`, options.targetBranch);
        return this.request('POST', `/projects/${projectId}/merge_requests`, {
            source_branch: branch,
            target_branch: options.targetBranch,
            title,
            description: options.description,
            assignee_ids: options.assigneeIds,
            reviewer_ids: options.reviewerIds,
        });
    }

    async updateMergeRequest(projectId: number, iid: number, changes: Record<string, unknown>) {
        await this.request('PUT', `/projects/${projectId}/merge_requests/${iid}`, changes);
    }

    async closeMergeRequest(projectId: number, iid: number) {
        await this.updateMergeRequest(projectId, iid, {state_event: 'close'});
    }

    async reopenMergeRequest(projectId: number, iid: number) {
        await this.updateMergeRequest(projectId, iid, {state_event: 'reopen'});
    }

    async approveMergeRequest(projectId: number, iid: number) {
        await this.request('POST', `/projects/${projectId}/merge_requests/${iid}/approve`);
    }

    async unapproveMergeRequest(projectId: number, iid: number) {
        await this.request('POST', `/projects/${projectId}/merge_requests/${iid}/unapprove`);
    }

    async commentOnMergeRequest(projectId: number, iid: number, body: string) {
        await this.request('POST', `/projects/${projectId}/merge_requests/${iid}/notes`, {body});
    }

    // GitLab checks mergeability asynchronously, so a merge right after creation can be rejected.
    async mergeMergeRequest(projectId: number, iid: number) {
        await this.poll(`merge request !${iid} to merge`, async () => {
            const response = await this.send('PUT', `/projects/${projectId}/merge_requests/${iid}/merge`);
            if (response.ok) {
                return true;
            }
            if ([405, 406, 409, 422].includes(response.status)) {
                return false;
            }
            throw new Error(`Merging !${iid} failed (${response.status}): ${await response.text()}`);
        });
    }

    async createTag(projectId: number, name: string, ref: string) {
        await this.request('POST', `/projects/${projectId}/repository/tags`, {tag_name: name, ref});
    }

    async deleteTag(projectId: number, name: string) {
        await this.request('DELETE', `/projects/${projectId}/repository/tags/${encodeURIComponent(name)}`);
    }

    async createRelease(projectId: number, tagName: string, name: string, ref: string) {
        await this.request('POST', `/projects/${projectId}/releases`, {tag_name: tagName, name, ref});
    }

    async waitForPipeline(projectId: number, ref: string, status: string): Promise<GitlabPipeline> {
        let found: GitlabPipeline | undefined;
        await this.poll(`a ${status} pipeline on ${ref}`, async () => {
            const pipelines = await this.request<GitlabPipeline[]>('GET', `/projects/${projectId}/pipelines?ref=${encodeURIComponent(ref)}`);
            found = pipelines.find((pipeline) => pipeline.status === status);
            return Boolean(found);
        }, 180000);
        return found!;
    }

    async listTodos(): Promise<GitlabTodo[]> {
        return this.request('GET', '/todos?state=pending');
    }

    async markAllTodosDone() {
        await this.request('POST', '/todos/mark_as_done');
    }

    // Group deletion is delayed, so close open items first to keep them out of the sidebar lists.
    async closeOpenItems(projectId: number) {
        const issues = await this.request<GitlabIssue[]>('GET', `/projects/${projectId}/issues?state=opened&per_page=100`);
        for (const issue of issues) {
            // eslint-disable-next-line no-await-in-loop
            await this.closeIssue(projectId, issue.iid);
        }
        const mergeRequests = await this.request<GitlabMergeRequest[]>('GET', `/projects/${projectId}/merge_requests?state=opened&per_page=100`);
        for (const mergeRequest of mergeRequests) {
            // eslint-disable-next-line no-await-in-loop
            await this.closeMergeRequest(projectId, mergeRequest.iid);
        }
    }

    // Sidebar counts span every project, so items left open by an interrupted run would skew them.
    async closeItemsInvolving(userId: number) {
        const queries: Array<[string, string]> = [
            ['issues', `assignee_id=${userId}`],
            ['merge_requests', `assignee_id=${userId}`],
            ['merge_requests', `reviewer_id=${userId}`],
        ];
        for (const [kind, filter] of queries) {
            // eslint-disable-next-line no-await-in-loop
            const items = await this.listAllPages<{iid: number; project_id: number}>(`/${kind}?scope=all&state=opened&per_page=100&${filter}`);
            for (const item of items) {
                // eslint-disable-next-line no-await-in-loop
                await this.request('PUT', `/projects/${item.project_id}/${kind}/${item.iid}`, {state_event: 'close'});
            }
        }
    }

    async deleteGroup(groupId: number) {
        const response = await this.send('DELETE', `/groups/${groupId}`);
        if (response.status === 404) {
            return;
        }
        if (!response.ok) {
            const body = await response.text();
            throw new Error(`DELETE /groups/${groupId} failed (${response.status}): ${body}`);
        }
    }

    private async waitForDefaultBranch(projectId: number) {
        let branch = '';
        await this.poll(`project ${projectId} to report a default branch`, async () => {
            const project = await this.request<GitlabProject>('GET', `/projects/${projectId}`);
            branch = project.default_branch;
            return Boolean(branch);
        });
        return branch;
    }

    private async poll(description: string, check: () => Promise<boolean>, timeoutMs = 30000) {
        const deadline = Date.now() + timeoutMs;
        while (Date.now() < deadline) {
            // eslint-disable-next-line no-await-in-loop
            if (await check()) {
                return;
            }
            // eslint-disable-next-line no-await-in-loop
            await new Promise((resolve) => setTimeout(resolve, POLL_INTERVAL_MS));
        }
        throw new Error(`Timed out waiting for ${description}`);
    }

    // Collects every page before the caller changes anything, since edits would shift the offsets.
    private async listAllPages<T>(apiPath: string): Promise<T[]> {
        const items: T[] = [];
        let page = '1';
        while (page) {
            // eslint-disable-next-line no-await-in-loop
            const response = await this.send('GET', `${apiPath}&page=${page}`);
            // eslint-disable-next-line no-await-in-loop
            const text = await response.text();
            if (!response.ok) {
                throw new Error(`GET ${apiPath} failed (${response.status}): ${text}`);
            }
            items.push(...JSON.parse(text) as T[]);
            page = response.headers.get('x-next-page') ?? '';
        }
        return items;
    }

    private async request<T>(method: string, apiPath: string, body?: unknown): Promise<T> {
        const response = await this.send(method, apiPath, body);
        const text = await response.text();
        if (!response.ok) {
            throw new Error(`${method} ${apiPath} failed (${response.status}): ${text}`);
        }
        if (!text) {
            return undefined as T;
        }
        return JSON.parse(text) as T;
    }

    private send(method: string, apiPath: string, body?: unknown) {
        return fetch(`${this.baseURL}/api/v4${apiPath}`, {
            method,
            headers: {
                'PRIVATE-TOKEN': this.token,
                'Content-Type': 'application/json',
            },
            body: body === undefined ? undefined : JSON.stringify(body),
        });
    }
}

export function uniquePath(prefix: string) {
    return `${prefix}${Math.random().toString(16).slice(2, 8)}`;
}
