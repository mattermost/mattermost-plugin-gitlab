// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {AnyAction, Dispatch, bindActionCreators} from 'redux';

import {UserProfile} from 'mattermost-redux/types/users';

import {getGitlabUser} from '../../actions';

import {GlobalState, pluginStateKey} from 'src/types/store';

import UserAttribute from './user_attribute';

export type UserAttributeProps = UserAttributeStateProps & UserAttributeDispatchProps

type UserAttributeDispatchProps = ReturnType<typeof mapDispatchToProps>;
type UserAttributeStateProps = ReturnType<typeof mapStateToProps>;

function mapStateToProps(state: GlobalState, ownProps: {user: UserProfile}) {
    const idUser = ownProps.user ? ownProps.user.id : '';
    const user = state[pluginStateKey].gitlabUsers[idUser] || {};
    return {
        id: idUser,
        username: user.username,
        gitlabURL: state[pluginStateKey].gitlabURL,
    };
}

function mapDispatchToProps(dispatch: Dispatch<AnyAction>) {
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
