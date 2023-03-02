import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getGitlabUser} from '../../actions';

import {id} from '../../manifest';

import UserAttribute from './user_attribute.jsx';

function mapStateToProps(state, ownProps) {
    const idUser = ownProps.user ? ownProps.user.id : '';
    const user = state[`plugins-${id}`].gitlabUsers[idUser] || {};

    return {
        id: idUser,
        username: user.username,
        gitlabURL: state[`plugins-${id}`].gitlabURL,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators(
            {
                getGitlabUser,
            },
            dispatch,
        ),
    };
}

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(UserAttribute);
