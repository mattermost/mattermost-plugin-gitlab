import {connect} from 'react-redux';
import {AnyAction, Dispatch, bindActionCreators} from 'redux';

import {UserProfile} from 'mattermost-redux/types/users';

import {getGitlabUser} from '../../actions';

import {GlobalState, pluginStateKey} from 'src/types/store';

import UserAttribute from './user_attribute';

interface UserAttributeStateProps {
    id: string,
    username: string,
    gitlabURL: string,
}

interface UserAttributeDispatchProps {
    actions: {
        getGitlabUser:(userID: string) => (dispatch: Dispatch<AnyAction>, getState: () => GlobalState) => Promise<any>,
    },
}

export type UserAttributeProps = UserAttributeStateProps & UserAttributeDispatchProps

function mapStateToProps(state: GlobalState, ownProps: {user: UserProfile}): UserAttributeStateProps {
    const idUser = ownProps.user ? ownProps.user.id : '';
    const user = state[pluginStateKey].gitlabUsers[idUser] || {};
    return {
        id: idUser,
        username: user.username,
        gitlabURL: state[pluginStateKey].gitlabURL,
    };
}

function mapDispatchToProps(dispatch: Dispatch<AnyAction>): UserAttributeDispatchProps {
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
)(UserAttribute as React.ComponentType<UserAttributeStateProps & UserAttributeDispatchProps>);
