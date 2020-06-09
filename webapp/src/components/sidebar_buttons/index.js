import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {
    getReviews,
    getUnreads,
    getYourPrs,
    getYourAssignments,
} from '../../actions';

import {id} from '../../manifest';

import {getPluginServerRoute} from '../../selectors';

import SidebarButtons from './sidebar_buttons.jsx';

function mapStateToProps(state) {
    return {
        connected: state[`plugins-${id}`].connected,
        username: state[`plugins-${id}`].username,
        clientId: state[`plugins-${id}`].clientId,
        reviews: state[`plugins-${id}`].reviews,
        yourPrs: state[`plugins-${id}`].yourPrs,
        yourAssignments: state[`plugins-${id}`].yourAssignments,
        unreads: state[`plugins-${id}`].unreads,
        gitlabURL: state[`plugins-${id}`].gitlabURL,
        org: state[`plugins-${id}`].organization,
        pluginServerRoute: getPluginServerRoute(state),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators(
            {
                getReviews,
                getUnreads,
                getYourPrs,
                getYourAssignments,
            },
            dispatch
        ),
    };
}

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(SidebarButtons);
