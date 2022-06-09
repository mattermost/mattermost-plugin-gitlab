import Client from '../client';
import ActionTypes from '../action_types';
import {id} from '../manifest';

export function getConnected(reminder = false) {
    return async (dispatch) => {
        let data;
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

function checkAndHandleNotConnected(data) {
    return async (dispatch) => {
        if (data && data.id === 'not_connected') {
            dispatch({
                type: ActionTypes.RECEIVED_CONNECTED,
                data: {
                    connected: false,
                    gitlab_username: '',
                    gitlab_client_id: '',
                    settings: {},
                },
            });
            return false;
        }
        return true;
    };
}

export function getReviews() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getReviews();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(
            dispatch,
            getState,
        );
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_REVIEWS,
            data,
        });

        return {data};
    };
}

export function getReviewDetails(prList) {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getPrsDetails(prList);
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
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

export function getYourPrs() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getYourPrs();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(
            dispatch,
            getState,
        );
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_YOUR_PRS,
            data,
        });

        return {data};
    };
}

export function getYourPrDetails(prList) {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getPrsDetails(prList);
        } catch (error) {
            return {error};
        }
        const connected = await checkAndHandleNotConnected(data)(dispatch, getState);
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

export function getYourAssignments() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getYourAssignments();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(
            dispatch,
            getState,
        );
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_YOUR_ASSIGNMENTS,
            data,
        });

        return {data};
    };
}

export function getMentions() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getMentions();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(
            dispatch,
            getState,
        );
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_MENTIONS,
            data,
        });

        return {data};
    };
}

export function getUnreads() {
    return async (dispatch, getState) => {
        let data;
        try {
            data = await Client.getUnreads();
        } catch (error) {
            return {error};
        }

        const connected = await checkAndHandleNotConnected(data)(
            dispatch,
            getState,
        );
        if (!connected) {
            return {error: data};
        }

        dispatch({
            type: ActionTypes.RECEIVED_UNREADS,
            data,
        });

        return {data};
    };
}

/**
 * Stores "showRHSPlugin" action returned by
 * "registerRightHandSidebarComponent" in plugin initialization.
 */
export function setShowRHSAction(showRHSPluginAction) {
    return {
        type: ActionTypes.RECEIVED_SHOW_RHS_ACTION,
        showRHSPluginAction,
    };
}

export function updateRhsState(rhsState) {
    return {
        type: ActionTypes.UPDATE_RHS_STATE,
        state: rhsState,
    };
}

const GITLAB_USER_GET_TIMEOUT_MILLISECONDS = 1000 * 60 * 60; // 1 hour

export function getGitlabUser(userID) {
    return async (dispatch, getState) => {
        if (!userID) {
            return {};
        }

        const user = getState()[`plugins-${id}`].gitlabUsers[userID];
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

        let data;
        try {
            data = await Client.getGitlabUser(userID);
        } catch (error) {
            if (error.status === 404) {
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
