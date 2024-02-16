import {combineReducers} from 'redux';

import ActionTypes from '../action_types';
import Constants from '../constants';
import {Item} from 'src/types/gitlab_items';
import {ConnectedData, GitlabUsersData, LHSData, ShowRhsPluginActionData, SubscriptionData} from 'src/types';

function connected(state = false, action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.connected;
    default:
        return state;
    }
}

function gitlabURL(state = '', action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data && action.data.gitlab_url) {
            return action.data.gitlab_url;
        }
        return '';
    default:
        return state;
    }
}

function organization(state = '', action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        if (action.data && action.data.organization) {
            return action.data.organization;
        }
        return '';
    default:
        return state;
    }
}

function username(state = '', action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.gitlab_username;
    default:
        return state;
    }
}

function settings(
    state = {
        sidebar_buttons: Constants.SETTING_BUTTONS_TEAM,
        daily_reminder: true,
        notifications: true,
    },
    action: {type: string, data: ConnectedData},
) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.settings;
    default:
        return state;
    }
}

function clientId(state = '', action: {type: string, data: ConnectedData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CONNECTED:
        return action.data.gitlab_client_id;
    default:
        return state;
    }
}

function reviewDetails(state = [], action: {type: string, data: Item}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_REVIEW_DETAILS:
        return action.data;
    default:
        return state;
    }
}

function yourPrDetails(state = [], action: {type: string, data: Item}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_YOUR_PR_DETAILS:
        return action.data;
    default:
        return state;
    }
}

function lhsData(state = [], action: {type: string, data: LHSData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_LHS_DATA:
        return action.data;
    default:
        return state;
    }
}

function rhsPluginAction(state = null, action: {type: string, showRHSPluginAction: ShowRhsPluginActionData}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_SHOW_RHS_ACTION:
        return action.showRHSPluginAction;
    default:
        return state;
    }
}

function rhsState(state = null, action: {type: string, state: string}) {
    switch (action.type) {
    case ActionTypes.UPDATE_RHS_STATE:
        return action.state;
    default:
        return state;
    }
}

function gitlabUsers(state: Record<string, GitlabUsersData> = {}, action: {type: string, data: GitlabUsersData, userID: string}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_GITLAB_USER: {
        const nextState = {...state};
        nextState[action.userID] = action.data;
        return nextState;
    }
    default:
        return state;
    }
}

function subscriptions(state: Record<string, SubscriptionData> = {}, action: {type: string, data: {channelId: string, subscriptions: SubscriptionData}}) {
    switch (action.type) {
    case ActionTypes.RECEIVED_CHANNEL_SUBSCRIPTIONS: {
        const nextState = {...state};
        nextState[action.data.channelId] = action.data.subscriptions;

        return nextState;
    }
    default:
        return state;
    }
}

export default combineReducers({
    connected,
    gitlabURL,
    organization,
    username,
    settings,
    clientId,
    gitlabUsers,
    rhsPluginAction,
    rhsState,
    yourPrDetails,
    reviewDetails,
    subscriptions,
    lhsData,
});
