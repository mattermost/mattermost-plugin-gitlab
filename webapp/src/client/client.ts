import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';

import {Options} from 'mattermost-redux/types/client4';

import {Item} from 'src/types/gitlab_items';
import {APIError, ConnectedData, GitlabUsersData, LHSData, SubscriptionData} from 'src/types';
import {CommentBody, IssueBody} from 'src/types/gitlab_types';

export default class Client {
    private url = '';

    setServerRoute(url: string): void {
        this.url = `${url}/api/v1`;
    }

    getConnected = async (reminder = false) => {
        return this.doGet<ConnectedData>(`${this.url}/connected?reminder=` + reminder);
    };

    getPrsDetails = async (prList: Item[]) => {
        return this.doPost<Item | APIError>(`${this.url}/prdetails`, prList);
    };

    getLHSData = async () => {
        return this.doGet<LHSData | APIError>(`${this.url}/lhs-data`);
    };

    getGitlabUser = async (userID: string) => {
        return this.doPost<GitlabUsersData>(`${this.url}/user`, {user_id: userID});
    };

    getIssue = async (owner: string, repo: string, issueNumber: string) => {
        return this.doGet<Item>(`${this.url}/issue?owner=${owner}&repo=${repo}&number=${issueNumber}`);
    };

    getPullRequest = async (owner: string, repo: string, prNumber: string) => {
        return this.doGet<Item>(`${this.url}/mergerequest?owner=${owner}&repo=${repo}&number=${prNumber}`);
    };

    getChannelSubscriptions = async (channelID: string) => {
        return this.doGet<SubscriptionData>(`${this.url}/channel/${channelID}/subscriptions`);
    };

    createIssue = async (payload: IssueBody) => {
        return this.doPost(`${this.url}/issue`, payload);
    }

    attachCommentToIssue = async (payload: CommentBody) => {
        return this.doPost(`${this.url}/attachcommenttoissue`, payload);
    }

    searchIssues = async (searchTerm: string) => {
        return this.doGet(`${this.url}/searchissues?search=${searchTerm}`);
    }

    getProjects = async () => {
        return this.doGet(`${this.url}/projects`);
    }

    getLabels = async (projectID: number) => {
        return this.doGet(`${this.url}/labels?projectID=${projectID}`);
    }

    getMilestones = async (projectID: number) => {
        return this.doGet(`${this.url}/milestones?projectID=${projectID}`);
    }

    getAssignees = async (projectID: number) => {
        return this.doGet(`${this.url}/assignees?projectID=${projectID}`);
    }

    private async doGet<Response>(url: string, headers: { [x: string]: string; } = {}): Promise<Response > {
        headers['X-Timezone-Offset'] = String(new Date().getTimezoneOffset());

        const options: Options = {
            method: 'get',
            headers,
        };

        const response = await fetch(url, Client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(Client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    }

    private async doPost<Response>(url: string, body: Object, headers: { [x: string]: string; } = {}): Promise<Response > {
        headers['X-Timezone-Offset'] = String(new Date().getTimezoneOffset());

        const options: Options = {
            method: 'post',
            body: JSON.stringify(body),
            headers,
        };

        const response = await fetch(url, Client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(Client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    }

    private async doDelete<Response>(url: string, headers: { [x: string]: string; } = {}): Promise<Response > {
        headers['X-Timezone-Offset'] = String(new Date().getTimezoneOffset());

        const options: Options = {
            method: 'delete',
            headers,
        };

        const response = await fetch(url, Client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(Client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    }

    private async doPut<Response>(url: string, body: Object, headers: { [x: string]: string; } = {}): Promise<Response > {
        headers['X-Timezone-Offset'] = String(new Date().getTimezoneOffset());

        const options = {
            method: 'put',
            body: JSON.stringify(body),
            headers,
        };

        const response = await fetch(url, Client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(Client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    }
}
