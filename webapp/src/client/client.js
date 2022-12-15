import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';

export default class Client {
    setServerRoute = (url) => {
        this.url = `${url}/api/v1`;
    }

    getConnected = async (reminder = false) => {
        return this.doGet(`${this.url}/connected?reminder=` + reminder);
    };

    getReviews = async () => {
        return this.doGet(`${this.url}/reviews`);
    };

    getYourPrs = async () => {
        return this.doGet(`${this.url}/yourprs`);
    };

    getYourAssignments = async () => {
        return this.doGet(`${this.url}/yourassignments`);
    };

    getMentions = async () => {
        return this.doGet(`${this.url}/mentions`);
    };

    getUnreads = async () => {
        return this.doGet(`${this.url}/unreads`);
    };

    getGitlabUser = async (userID) => {
        return this.doPost(`${this.url}/user`, {user_id: userID});
    };

    getIssue = async (owner, repo, issueNumber) => {
        return this.doGet(`${this.url}/issue?owner=${owner}&repo=${repo}&number=${issueNumber}`);
    }

    getPullRequest = async (owner, repo, prNumber) => {
        return this.doGet(`${this.url}/mergerequest?owner=${owner}&repo=${repo}&number=${prNumber}`);
    }

    getChannelSubscriptions = async (channelID) => {
        return this.doGet(`${this.url}/channel/${channelID}/subscriptions`);
    };

    doGet = async (url, body, headers = {}) => {
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
    };

    doPost = async (url, body, headers = {}) => {
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
    };

    doDelete = async (url, body, headers = {}) => {
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
    };

    doPut = async (url, body, headers = {}) => {
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
    };
}
