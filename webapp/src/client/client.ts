import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';
import {ConnectedResponse, PrDetails, LHSContent, GitlabUserResponse, IssueResponse, MergeRequestResponse, SubscriptionResponse} from 'src/types/gitlab_items';

export default class Client {
    private url = '';

    setServerRoute(url: string): void {
        this.url = `${url}/api/v1`;
    }

    async getConnected(reminder: boolean): Promise<ConnectedResponse> {
        return this.doGet(`${this.url}/connected?reminder=` + reminder);
    }

    async getPrsDetails(prList: any): Promise<PrDetails> {
        return this.doPost(`${this.url}/prdetails`, prList);
    }

    async getLHSData(): Promise<LHSContent> {
        return this.doGet(`${this.url}/lhs-data`);
    }

    async getGitlabUser(userID: string): Promise<GitlabUserResponse> {
        return this.doPost(`${this.url}/user`, {user_id: userID});
    }

    async getIssue(owner: string, repo: string, issueNumber: string): Promise<IssueResponse> {
        return this.doGet(`${this.url}/issue?owner=${owner}&repo=${repo}&number=${issueNumber}`);
    }

    async getPullRequest(owner: string, repo: string, prNumber: string): Promise<MergeRequestResponse> {
        return this.doGet(`${this.url}/mergerequest?owner=${owner}&repo=${repo}&number=${prNumber}`);
    }

    async getChannelSubscriptions(channelID: string): Promise<SubscriptionResponse> {
        return this.doGet(`${this.url}/channel/${channelID}/subscriptions`);
    }

    private async doGet(url: string, body?: any, headers: Record<string, any> = {}): Promise<any> {
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

    private async doPost(url: string, body: any, headers: Record<string, any> = {}): Promise<any> {
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
