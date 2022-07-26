import {getConfig} from 'mattermost-redux/selectors/entities/general';

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
