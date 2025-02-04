// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {combineReducers, Reducer} from 'redux';

import ActionTypes from '../action_types';
import Constants from '../constants';
import {Item} from 'src/types/gitlab_items';
import {ConnectedData, CreateIssueModalData, GitlabUsersData, LHSData, ShowRhsPluginActionData, SubscriptionData} from 'src/types';
import {Project} from 'src/types/gitlab_types';

const connected: Reducer<boolean, {type: string, data: ConnectedData}> = (state = false, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.connected;
    default:
        return state;
    }
};

const gitlabURL: Reducer<string, {type: string, data: ConnectedData}> = (state = '', action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data && action.data.gitlab_url) {
            return action.data.gitlab_url;
        }
        return '';
    default:
        return state;
    }
};

const organization: Reducer<string, {type: string, data: ConnectedData}> = (state = '', action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data && action.data.organization) {
            return action.data.organization;
        }
        return '';
    default:
        return state;
    }
};

const username: Reducer<string, {type: string, data: ConnectedData}> = (state = '', action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.gitlab_username;
    default:
        return state;
    }
};

const settings: Reducer<{
    sidebar_buttons: string,
    daily_reminder: boolean,
    notifications: boolean,
}, {type: string, data: ConnectedData}> = (state = {
    sidebar_buttons: Constants.SETTING_BUTTONS_TEAM,
    daily_reminder: true,
    notifications: true,
}, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.settings;
    default:
        return state;
    }
};

const clientId: Reducer<string, {type: string, data: ConnectedData}> = (state = '', action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.gitlab_client_id;
    default:
        return state;
    }
};

const reviewDetails: Reducer<Item[] | null, {type: string, data: Item[]}> = (state = [], action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_REVIEW_DETAILS:
        return action.data;
    default:
        return state;
    }
};

const yourPrDetails: Reducer<Item[] | null, {type: string, data: Item[]}> = (state = [], action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_YOUR_PR_DETAILS:
        return action.data;
    default:
        return state;
    }
};

const lhsData: Reducer<LHSData | null, {type: string, data: LHSData}> = (state = null, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_LHS_DATA:
        return action.data;
    default:
        return state;
    }
};

function rhsPluginAction(state = null, action: {type: string, showRHSPluginAction: ShowRhsPluginActionData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_SHOW_RHS_ACTION:
        return action.showRHSPluginAction;
    default:
        return state;
    }
}

const rhsState: Reducer<string | null, {type: string, state: string}> = (state = null, action) => {
    switch (action.type) {
    case ActionTypes.UPDATE_RHS_STATE:
        return action.state;
    default:
        return state;
    }
};

const gitlabUsers: Reducer<Record<string, GitlabUsersData>, {type: string, data: GitlabUsersData, userID: string}> = (state = {}, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_GITLAB_USER: {
        const nextState = {...state};
        nextState[action.userID] = action.data;
        return nextState;
    }
    default:
        return state;
    }
};

const isCreateIssueModalVisible = (state = false, action: {type: string}) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST:
        return true;
    case ActionTypes.CLOSE_CREATE_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const isAttachCommentToIssueModalVisible = (state = false, action: {type: string}) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return true;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return false;
    default:
        return state;
    }
};

const postIdForAttachCommentToIssueModal = (state = '', action: {type: string, data: {postId: string}}) => {
    switch (action.type) {
    case ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return action.data.postId;
    case ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL:
        return '';
    default:
        return state;
    }
};

const createIssueModal: Reducer<CreateIssueModalData, {type: string, data: CreateIssueModalData}> = (state = {}, action) => {
    switch (action.type) {
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL:
    case ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST:
        return {
            ...state,

            postId: action.data.postId,
            title: action.data.title,
            channelId: action.data.channelId,
        };
    case ActionTypes.CLOSE_CREATE_ISSUE_MODAL:
        return {};
    default:
        return state;
    }
};

function yourProjects(state = [] as Project[], action: {type: string, data: Project[]}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_PROJECTS:
        return action.data;
    default:
        return state;
    }
}

const subscriptions: Reducer<Record<string, SubscriptionData>, {type: string, data: {channelId: string, subscriptions: SubscriptionData}}> = (state = {}, action) => {
    switch (action.type) {
    case ActionTypes.RECEIVED_CHANNEL_SUBSCRIPTIONS: {
        const nextState = {...state};
        nextState[action.data.channelId] = action.data.subscriptions;

        return nextState;
    }
    default:
        return state;
    }
};

export default combineReducers({
    connected,
    gitlabURL,
    organization,
    username,
    settings,
    clientId,
    gitlabUsers,
    isCreateIssueModalVisible,
    yourProjects,
    createIssueModal,
    postIdForAttachCommentToIssueModal,
    isAttachCommentToIssueModalVisible,
    rhsPluginAction,
    rhsState,
    yourPrDetails,
    reviewDetails,
    subscriptions,
    lhsData,
});
