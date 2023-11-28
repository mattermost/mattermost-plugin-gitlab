import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {
    updateRHSState,
    getLHSData,
} from '../../actions';

import {id} from '../../manifest';

import {getPluginServerRoute} from '../../selectors';

import SidebarButtons from './sidebar_buttons.jsx';

function mapStateToProps(state) {
    return {
        connected: state[`plugins-${id}`].connected,
        username: state[`plugins-${id}`].username,
        clientId: state[`plugins-${id}`].clientId,
        reviews: state[`plugins-${id}`].lhsData?.reviews,
        yourPrs: state[`plugins-${id}`].lhsData?.prs,
        yourAssignments: state[`plugins-${id}`].lhsData?.assignments,
        unreads: state[`plugins-${id}`].lhsData?.unreads,
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
