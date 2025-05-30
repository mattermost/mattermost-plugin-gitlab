// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {
    updateRHSState,
    sendEphemeralPost,
    getLHSData,
} from '../../actions';

import manifest from '../../manifest';

import {getPluginServerRoute} from '../../selectors';

import SidebarButtons from './sidebar_buttons';

function mapStateToProps(state) {
    const {id} = manifest;
    return {
        connected: state[`plugins-${id}`].connected,
        username: state[`plugins-${id}`].username,
        clientId: state[`plugins-${id}`].clientId,
        reviews: state[`plugins-${id}`].lhsData?.reviews,
        yourAssignedPrs: state[`plugins-${id}`].lhsData?.yourAssignedPrs,
        yourAssignedIssues: state[`plugins-${id}`].lhsData?.yourAssignedIssues,
        todos: state[`plugins-${id}`].lhsData?.todos,
        gitlabURL: state[`plugins-${id}`].gitlabURL,
        org: state[`plugins-${id}`].organization,
        pluginServerRoute: getPluginServerRoute(state),
        showRHSPlugin: state[`plugins-${id}`].rhsPluginAction,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators(
            {
                updateRHSState,
                sendEphemeralPost,
                getLHSData,
            },
            dispatch,
        ),
    };
}

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(SidebarButtons);
