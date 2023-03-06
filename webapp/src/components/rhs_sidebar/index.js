import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';
import {getCurrentUserId} from 'mattermost-redux/selectors/entities/users';

import {id} from '../../manifest';
import {
    getChannelSubscriptions,
} from '../../actions';
import {getPluginServerRoute} from '../../selectors';

import RHSSidebar from './rhs_sidebar.jsx';

const noSubscriptions = [];

function mapStateToProps(state) {
    const currentUserId = getCurrentUserId(state);
    const currentChannelId = getCurrentChannelId(state);

    return {
        currentUserId,
        connected: state[`plugins-${id}`].connected,
        username: state[`plugins-${id}`].username,
        gitlabURL: state[`plugins-${id}`].gitlabURL,
        currentChannelId,
        currentChannelSubscriptions: state[`plugins-${id}`].subscriptions[currentChannelId] || noSubscriptions,
        pluginServerRoute: getPluginServerRoute(state),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators(
            {
                getChannelSubscriptions,
            },
            dispatch,
        ),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(RHSSidebar);
