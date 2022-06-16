// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators, Dispatch} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';
import {GenericAction} from 'mattermost-redux/types/actions';
import { GlobalState } from 'mattermost-redux/types/store';

import {id as pluginId} from '../../manifest';
import {openCreateIssueModal} from '../../actions';

import CreateIssuePostMenuAction from './create_issue';

type plugin = "plugin"

interface pluginMethods {
    connected: boolean
}

interface CurrentState extends GlobalState {
    plugin: pluginMethods;
}

interface OwnProps {
    postId: string;
}

const mapStateToProps = (state: CurrentState, ownProps: OwnProps) => {
    const post = getPost(state, ownProps.postId);
    const systemMessage = Boolean(!post || isSystemMessage(post));

    return {
        show: state[`plugins-${pluginId}` as plugin].connected && !systemMessage,
    };
};

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>) => {
    return {
        actions: bindActionCreators({
            open: openCreateIssueModal,
        }, dispatch),
    };
};

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssuePostMenuAction);
