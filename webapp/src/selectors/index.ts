import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {createSelector} from 'reselect';

import manifest from '../manifest';
import {Item} from 'src/types/gitlab_items';
import {GlobalState, PluginState, pluginStateKey} from 'src/types/store';
import {SideBarData} from 'src/types';

export const getPluginServerRoute = (state: GlobalState) => {
    const config = getConfig(state);

    let basePath = '';
    if (config && config.SiteURL) {
        basePath = new URL(config.SiteURL).pathname;

        if (basePath && basePath[basePath.length - 1] === '/') {
            basePath = basePath.substr(0, basePath.length - 1);
        }
    }

    return basePath + '/plugins/' + manifest.id;
};

function mapPrsToDetails(prs?: Item[], details?: Item[]): Item[] {
    if (!prs || !prs.length) {
        return [];
    }

    return prs.map((pr) => {
        const foundDetails = details && details.find((prDetails) => pr.project_id === prDetails.project_id && pr.sha === prDetails.sha);
        if (!foundDetails) {
            return pr;
        }

        return {
            ...pr,
            status: foundDetails.status,
            num_approvers: foundDetails.num_approvers,
            total_reviewers: pr.reviewers.length,
        };
    });
}

export const getPluginState = (state: GlobalState) => state[`plugins-${manifest.id}` as pluginStateKey];

export const getSidebarData = createSelector(
    getPluginState,
    (pluginState: PluginState) => {
        return {
            username: pluginState.username,
            reviewDetails: pluginState.reviewDetails,
            reviews: mapPrsToDetails(pluginState.lhsData?.reviews, pluginState.reviewDetails || []),
            yourAssignedPrs: mapPrsToDetails(pluginState.lhsData?.yourAssignedPrs, pluginState.yourPrDetails || []),
            yourPrDetails: pluginState.yourPrDetails,
            yourAssignedIssues: pluginState.lhsData?.yourAssignedIssues,
            todos: pluginState.lhsData?.todos,
            org: pluginState.organization,
            gitlabURL: pluginState.gitlabURL,
            rhsState: pluginState.rhsState,
        } as SideBarData;
    },
);

export const getConnected = (state) => state[`plugins-${manifest.id}`].connected;

export const getConnectedGitlabUrl = (state) => state[`plugins-${manifest.id}`].gitlabURL;
