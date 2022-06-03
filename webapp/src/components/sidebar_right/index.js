// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {id as pluginId} from '../../manifest';

import SidebarRight from './sidebar_right.tsx';

function mapStateToProps(state) {
    return {
        username: state[`plugins-${pluginId}`].username,
        reviews: state[`plugins-${pluginId}`].reviews,
        yourPrs: state[`plugins-${pluginId}`].yourPrs,
        yourAssignments: state[`plugins-${pluginId}`].yourAssignments,
        unreads: state[`plugins-${pluginId}`].unreads,
        org: state[`plugins-${pluginId}`].organization,
        gitlabURL: state[`plugins-${pluginId}`].gitlabURL,
        rhsState: state[`plugins-${pluginId}`].rhsState,
    };
}

export default connect(mapStateToProps)(SidebarRight);
