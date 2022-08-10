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

export const getPluginState = (state) => state[`plugins-${PluginId}`];

const sidebarData = (state) => {
    const pluginState = getPluginState(state);
    return {
        username: pluginState.username,
        reviews: pluginState.reviews,
        reviewDetails: pluginState.reviewDetails,
        yourPrs: pluginState.yourPrs,
        yourPrDetails: pluginState.yourPrDetails,
        yourAssignments: pluginState.yourAssignments,
        unreads: pluginState.unreads,
        org: pluginState.organization,
        gitlabURL: pluginState.gitlabURL,
        rhsState: pluginState.rhsState,
    };
};

export const getSidebarData = createSelector(sidebarData, (data) => data);
