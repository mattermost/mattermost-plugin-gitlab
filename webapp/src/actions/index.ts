// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getCurrentChannelId, getCurrentUserId} from 'mattermost-redux/selectors/entities/common';

import {PostTypes} from 'mattermost-redux/action_types';

import {AnyAction, Dispatch} from 'redux';

import Client from '../client';
import ActionTypes, {RHSViewTypeValue} from '../action_types';
import {APIError, ConnectedData, GitlabUsersData, LHSData, ShowRhsPluginActionData, SubscriptionData} from 'src/types';
import {Item} from 'src/types/gitlab_items';
import {GlobalState} from 'src/types/store';
import {getPluginState} from 'src/selectors';
import {CommentBody, IssueBody} from 'src/types/gitlab_types';

export function getConnected(reminder = false) {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data: ConnectedData;
        try {
            data = await Client.getConnected(reminder);
        } catch (error) {
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data,
        });

        return {data};
    };
}

function checkAndHandleNotConnected(data: APIError) {
    return async (dispatch: Dispatch<AnyAction>) => {
        if (data && data.id === 'not_connected') {
            dispatch({
                type: ActionTypes.RECEIVED_CONNECTED,
                data: {
                    connected: false,
                    gitlab_username: '',
                    gitlab_client_id: '',
                    settings: {},
                } as ConnectedData,
            });
            return false;
        }
        return true;
    };
}

export function getReviewDetails(prList: Item[]) {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data: Item | APIError;
        try {
            data = await Client.getPrsDetails(prList);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_REVIEW_DETAILS,
            data,
        });

        return {data};
    };
}

export function getYourPrDetails(prList: Item[]) {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data: Item | APIError;
        try {
            data = await Client.getPrsDetails(prList);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_YOUR_PR_DETAILS,
            data,
        });

        return {data};
    };
}

export function getLHSData() {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data: LHSData | APIError;
        try {
            data = await Client.getLHSData();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_LHS_DATA,
            data,
        });

        return {data};
    };
}

/**
 * Stores "showRHSPlugin" action returned by
 * "registerRightHandSidebarComponent" in plugin initialization.
 */
export function setShowRHSAction(showRHSPluginAction: () => ShowRhsPluginActionData) {
    return {
        type: ActionTypes.RECEIVED_SHOW_RHS_ACTION,
        showRHSPluginAction,
    };
}

export function updateRHSState(rhsState: string) {
    return {
        type: ActionTypes.UPDATE_RHS_STATE,
        state: rhsState,
    };
}

/**
 * Sets which view is displayed in the unified RHS component.
 * Used to switch between SidebarRight (PRs/reviews/etc) and RHSSidebar (subscriptions).
 */
export function setRHSViewType(viewType: RHSViewTypeValue) {
    return {
        type: ActionTypes.SET_RHS_VIEW_TYPE,
        viewType,
    };
}

const GITLAB_USER_GET_TIMEOUT_MILLISECONDS = 1000 * 60 * 60; // 1 hour

export function getGitlabUser(userID: string) {
    return async (dispatch: Dispatch<AnyAction>, getState: () => GlobalState) => {
        if (!userID) {
            return {};
        }

        const user = getPluginState(getState()).gitlabUsers[userID];
        if (
            user &&
            user.last_try &&
            Date.now() - user.last_try < GITLAB_USER_GET_TIMEOUT_MILLISECONDS
        ) {
            return {};
        }

        if (user && user.username) {
            return {data: user};
        }

        let data: GitlabUsersData;
        try {
            data = await Client.getGitlabUser(userID);
        } catch (error: unknown) {
            if ((error as APIError).status === 404) {
                dispatch({
                    type: ActionTypes.RECEIVED_GITLAB_USER,
                    userID,
                    data: {last_try: Date.now()},
                });
            }
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_GITLAB_USER,
            userID,
            data,
        });

        return {data};
    };
}

export function openCreateIssueModal(postId: string) {
    return {
        type: ActionTypes.OPEN_CREATE_ISSUE_MODAL,
        data: {
            postId,
        },
    };
}

export function openCreateIssueModalWithoutPost(title: string, channelId: string) {
    return {
        type: ActionTypes.OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST,
        data: {
            title,
            channelId,
        },
    };
}

export function closeCreateIssueModal() {
    return {
        type: ActionTypes.CLOSE_CREATE_ISSUE_MODAL,
    };
}

export function createIssue(payload: IssueBody) {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data;
        try {
            data = await Client.createIssue(payload);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }
        return {data};
    };
}

export function openAttachCommentToIssueModal(postId: string) {
    return {
        type: ActionTypes.OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL,
        data: {
            postId,
        },
    };
}

export function closeAttachCommentToIssueModal() {
    return {
        type: ActionTypes.CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL,
    };
}

export function attachCommentToIssue(payload: CommentBody) {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data;
        try {
            data = await Client.attachCommentToIssue(payload);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }
        return {data};
    };
}

export function getProjects() {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data;
        try {
            data = await Client.getProjects();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_PROJECTS,
            data,
        });

        return {data};
    };
}

export function getLabelOptions(projectID: number) {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data;
        try {
            data = await Client.getLabels(projectID);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }

        return {data};
    };
}

export function getMilestoneOptions(projectID: number) {
    return async (dispatch: Dispatch<AnyAction>) => {
        let data;
        try {
            data = await Client.getMilestones(projectID);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }

        return {data};
    };
}

export function getAssigneeOptions(projectID: number) {
    return async (dispatch: Dispatch<AnyAction>, getState: () => GlobalState) => {
        let data;
        try {
            data = await Client.getAssignees(projectID);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data as APIError)(dispatch);
        if (!connected) {
            return {error: data};
        }

        return {data};
    };
}

export function getChannelSubscriptions(channelId: string) {
    return async (dispatch: Dispatch<AnyAction>) => {
        if (!channelId) {
            return {};
        }

        let subscriptions: SubscriptionData;
        try {
            subscriptions = await Client.getChannelSubscriptions(channelId);
        } catch (error) {
            return {error};
        }

        dispatch({
            type: ActionTypes.RECEIVED_CHANNEL_SUBSCRIPTIONS,
            data: {
                channelId,
                subscriptions,
            },
        });

        return {subscriptions};
    };
}

export function sendEphemeralPost(message: string) {
    return (dispatch: Dispatch<AnyAction>, getState: () => GlobalState) => {
        const timestamp = Date.now();
        const state = getState();

        const post = {
            id: 'gitlabPlugin' + Date.now(),
            user_id: getCurrentUserId(state),
            channel_id: getCurrentChannelId(state),
            message,
            type: 'system_ephemeral',
            create_at: timestamp,
            update_at: timestamp,
            root_id: '',
            parent_id: '',
            props: {},
        };

        dispatch({
            type: PostTypes.RECEIVED_NEW_POST,
            data: post,
        });
    };
}
