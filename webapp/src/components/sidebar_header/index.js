import {connect} from 'react-redux';

import manifest from '../../manifest';

import SidebarHeader from './sidebar_header.jsx';

function mapStateToProps(state) {
    const members = state.entities.teams.myMembers || {};
    return {
        show: Object.keys(members).length <= 1,
        connected: state[`plugins-${manifest.id}`].connected,
    };
}

export default connect(mapStateToProps)(SidebarHeader);
