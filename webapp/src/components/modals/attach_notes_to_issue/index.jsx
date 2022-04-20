import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import {id as pluginId} from 'manifest';
import {closeAttachNotesToIssueModal, attachNotesToIssue} from './../../../actions';

import AttachNotesToIssue from './attach_notes_to_issue';

const mapStateToProps = (state) => {
    const postId = state[`plugins-${pluginId}`].attachNotesToIssueModalForPostId;
    const post = getPost(state, postId);

    return {
        visible: state[`plugins-${pluginId}`].attachNotesToIssueModalVisible,
        post,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeAttachNotesToIssueModal,
    create: attachNotesToIssue,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachNotesToIssue);