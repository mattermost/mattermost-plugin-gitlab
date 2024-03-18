import {Client4} from 'mattermost-redux/client';
import {ClientError} from 'mattermost-redux/client/client4';

export default class Client {
    setServerRoute = (url) => {
        this.url = `${url}/api/v1`;
    }

    getConnected = async (reminder = false) => {
        return this.doGet(`${this.url}/connected?reminder=` + reminder);
    };

    getPrsDetails = async (prList) => {
        return this.doPost(`${this.url}/prdetails`, prList);
    }

    getLHSData= async () => {
        return this.doGet(`${this.url}/lhs-data`);
    }

    getMentions = async () => {
        return this.doGet(`${this.url}/mentions`);
    };

    getGitlabUser = async (userID) => {
        return this.doPost(`${this.url}/user`, {user_id: userID});
    };

    createIssue = async (payload) => {
        return this.doPost(`${this.url}/issue`, payload);
    }

    attachCommentToIssue = async (payload) => {
        return this.doPost(`${this.url}/attachcommenttoissue`, payload);
    }

    searchIssues = async (searchTerm) => {
        return this.doGet(`${this.url}/searchissues?search=${searchTerm}`);
    }

    getProjects = async () => {
        return this.doGet(`${this.url}/projects`);
    }

    getLabels = async (projectID) => {
        return this.doGet(`${this.url}/labels?projectID=${projectID}`);
    }

    getMilestones = async (projectID) => {
        return this.doGet(`${this.url}/milestones?projectID=${projectID}`);
    }

    getAssignees = async (projectID) => {
        return this.doGet(`${this.url}/assignees?projectID=${projectID}`);
    }

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
