// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {ActionCreatorsMapObject, bindActionCreators, Dispatch} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {ActionFunc, GenericAction} from 'mattermost-redux/types/actions';
import {GlobalState} from 'mattermost-redux/types/store';

import {id as pluginId} from '../../../manifest';
import {closeAttachCommentToIssueModal, attachCommentToIssue} from '../../../actions';

import AttachCommentToIssueModal from './attach_comment_to_issue';

type plugin = "plugin"

interface pluginMethods {
    attachCommentToIssueModalForPostId: string;
    attachCommentToIssueModalVisible: boolean
}

interface CurrentState extends GlobalState {
    plugin: pluginMethods;
}

type Actions = {
    close: () => {
        type: string;
    };
    create: (payload: any) => Promise<{
        error: any;
        data?: undefined;
    } | {
        data: any;
        error?: undefined;
    }>;
};

const mapStateToProps = (state: CurrentState) => {
    const postId = state[`plugins-${pluginId}` as plugin].attachCommentToIssueModalForPostId;
    const post = getPost(state, postId);

    return {
        visible: state[`plugins-${pluginId}` as plugin].attachCommentToIssueModalVisible,
        post,
    };
};


const mapDispatchToProps = (dispatch: Dispatch<GenericAction>) => {
    return {
        actions: bindActionCreators<ActionCreatorsMapObject<ActionFunc | GenericAction>, Actions>({
            close: closeAttachCommentToIssueModal,
            create: attachCommentToIssue,
        }, dispatch),
    };
};

export default connect(mapStateToProps, mapDispatchToProps)(AttachCommentToIssueModal);
