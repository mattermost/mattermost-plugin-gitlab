// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';
import {getCurrentUserId} from 'mattermost-redux/selectors/entities/users';

import manifest from '../../manifest';
import {
    getChannelSubscriptions,
    sendEphemeralPost,
} from '../../actions';
import {getPluginServerRoute} from '../../selectors';

import RHSSidebar from './rhs_sidebar';

const noSubscriptions = [];

function mapStateToProps(state) {
    const currentUserId = getCurrentUserId(state);
    const {id} = manifest;
    const pluginState = state[`plugins-${id}`];

    // In popout windows, use the channel ID synced from the parent window
    // Fall back to getCurrentChannelId for normal usage
    const currentChannelId = pluginState.popoutChannelId || getCurrentChannelId(state);

    return {
        currentUserId,
        connected: pluginState.connected,
        username: pluginState.username,
        gitlabURL: pluginState.gitlabURL,
        currentChannelId,
        currentChannelSubscriptions: pluginState.subscriptions[currentChannelId] || noSubscriptions,
        pluginServerRoute: getPluginServerRoute(state),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators(
            {
                getChannelSubscriptions,
                sendEphemeralPost,
            },
            dispatch,
        ),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(RHSSidebar);
