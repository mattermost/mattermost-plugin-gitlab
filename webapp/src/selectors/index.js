import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {createSelector} from 'reselect';

import {id as PluginId} from '../manifest';

export const getPluginServerRoute = (state) => {
    const config = getConfig(state);

    let basePath = '';
    if (config && config.SiteURL) {
        basePath = new URL(config.SiteURL).pathname;

        if (basePath && basePath[basePath.length - 1] === '/') {
            basePath = basePath.substr(0, basePath.length - 1);
        }
    }

    return basePath + '/plugins/' + PluginId;
};

function mapPrsToDetails(prs, details) {
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

export const getPluginState = (state) => state[`plugins-${PluginId}`];

export const getSidebarData = createSelector(
    getPluginState,
    (pluginState) => {
        return {
            username: pluginState.username,
            reviewDetails: pluginState.reviewDetails,
            reviews: mapPrsToDetails(pluginState.lhsData?.reviews, pluginState.reviewDetails),
            yourPrs: mapPrsToDetails(pluginState.lhsData?.prs, pluginState.yourPrDetails),
            yourPrDetails: pluginState.yourPrDetails,
            yourAssignments: pluginState.lhsData?.assignments,
            unreads: pluginState.lhsData?.unreads,
            org: pluginState.organization,
            gitlabURL: pluginState.gitlabURL,
            rhsState: pluginState.rhsState,
        };
    },
);
