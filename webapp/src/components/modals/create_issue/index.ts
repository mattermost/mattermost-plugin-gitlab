import {connect} from 'react-redux';
import {ActionCreatorsMapObject, bindActionCreators, Dispatch} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {GlobalState} from 'mattermost-redux/types/store';
import {ActionFunc, GenericAction} from 'mattermost-redux/types/actions';

import {id as pluginId} from 'src/manifest';
import {closeCreateIssueModal, createIssue} from 'src/actions';

import CreateIssueModal, {Actions} from './create_issue';

interface pluginMethods {
    createIssueModal: {
        postId: string;
        title: string;
        channelId: string;
    };
    isCreateIssueModalVisible: boolean
}

interface CurrentState extends GlobalState {
    plugin: pluginMethods;
}

const mapStateToProps = (state: CurrentState) => {
    const {postId, title, channelId} = state[`plugins-${pluginId}` as plugin].createIssueModal;
    
    const post = postId ? getPost(state, postId) : null;
    return {
        visible: state[`plugins-${pluginId}` as plugin].isCreateIssueModalVisible,
        post,
        title,
        channelId,
    };
};

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>) => {
    return {
        actions: bindActionCreators<ActionCreatorsMapObject<ActionFunc | GenericAction>, Actions>({
            close: closeCreateIssueModal,
            create: createIssue,
        }, dispatch),
    };
};

export default connect(mapStateToProps, mapDispatchToProps)(CreateIssueModal);
