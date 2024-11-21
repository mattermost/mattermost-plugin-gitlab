import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import {Project} from './gitlab_types';

export type GlobalState = ReduxGlobalState & {
    'plugins-com.github.manland.mattermost-plugin-gitlab': {
        createIssueModal: {
            postId: string;
            title: string;
            channelId: string;
        };
        isCreateIssueModalVisible: boolean;
        yourProjects: Project[];
        connected: boolean;
        postIdForAttachCommentToIssueModal: string;
        isAttachCommentToIssueModalVisible: boolean;
    }
}
