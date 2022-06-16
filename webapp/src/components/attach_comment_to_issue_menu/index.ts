// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {ActionCreatorsMapObject, bindActionCreators, Dispatch} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';
import {ActionFunc, GenericAction} from 'mattermost-redux/types/actions';
import {GlobalState} from 'mattermost-redux/types/store';

import {id as pluginId} from '../../manifest';
import {openAttachCommentToIssueModal} from '../../actions';

import AttachCommentToIssuePostMenuAction from './attach_comment_to_issue';

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

type Actions = {
    open: (postId: any) => {
        type: string;
        data: {
            postId: any;
        };
    };
};

const mapStateToProps = (state: CurrentState, ownProps: OwnProps) => {
    const post = getPost(state, ownProps.postId);
    const systemMessage = post ? isSystemMessage(post) : true;

    return {
        show: state[`plugins-${pluginId}` as plugin].connected && !systemMessage,
    };
};

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>) => {
    return {
        actions: bindActionCreators<ActionCreatorsMapObject<ActionFunc | GenericAction>, Actions>({
            open: openAttachCommentToIssueModal,
        }, dispatch),
    };
};

export default connect(mapStateToProps, mapDispatchToProps)(AttachCommentToIssuePostMenuAction);
