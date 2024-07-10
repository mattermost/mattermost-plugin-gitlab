import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';

import {Item, TooltipData} from 'src/types/gitlab_items';
import {APIError, ConnectedData, GitlabUsersData, LHSData, SubscriptionData} from 'src/types';

export default class Client {
    private url = '';

    setServerRoute(url: string): void {
        this.url = `${url}/api/v1`;
    }

    getConnected = async (reminder: boolean) => {
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
        return this.doGet<TooltipData | null>(`${this.url}/issue?owner=${owner}&repo=${repo}&number=${issueNumber}`);
    };

    getPullRequest = async (owner: string, repo: string, prNumber: string) => {
        return this.doGet<TooltipData | null>(`${this.url}/mergerequest?owner=${owner}&repo=${repo}&number=${prNumber}`);
    };

    getChannelSubscriptions = async (channelID: string) => {
        return this.doGet<SubscriptionData>(`${this.url}/channel/${channelID}/subscriptions`);
    };

    private async doGet<Response>(url: string, body?: any, headers: Record<string, any> = {}): Promise<Response> {
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const options = {
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

    private async doPost<Response>(url: string, body: any, headers: Record<string, any> = {}): Promise<Response> {
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const options = {
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

    private async doDelete(url: string, body?: any, headers: Record<string, any> = {}): Promise<any> {
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

        const options = {
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

    private async doPut(url: string, body: any, headers: Record<string, any> = {}): Promise<any> {
        headers['X-Timezone-Offset'] = new Date().getTimezoneOffset();

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
