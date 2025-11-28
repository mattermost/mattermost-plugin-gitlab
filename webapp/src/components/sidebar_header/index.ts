// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {getConnected} from 'src/selectors';
import {GlobalState} from 'src/types/store';

import SidebarHeader from './sidebar_header';

function mapStateToProps(state: GlobalState) {
    const members = state.entities.teams.myMembers || {};
    return {
        show: Object.keys(members).length <= 1,
        connected: getConnected(state),
    };
}

export default connect(mapStateToProps)(SidebarHeader);
