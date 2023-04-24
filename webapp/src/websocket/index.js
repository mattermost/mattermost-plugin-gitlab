import ActionTypes from '../action_types';
import Constants from '../constants';
import {
    getConnected,
    getReviews,
    getUnreads,
    getYourPrs,
    getYourAssignments,
} from '../actions';
import {id} from '../manifest';

export function handleConnect(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }

        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: {
                ...msg.data,
                settings: {
                    sidebar_buttons: Constants.SETTING_BUTTONS_TEAM,
                    daily_reminder: true,
                },
            },
        });
    };
}

export function handleDisconnect(store) {
    return () => {
        store.dispatch({
            type: ActionTypes.RECEIVED_CONNECTED,
            data: {
                connected: false,
                gitlab_username: '',
                gitlab_client_id: '',
                settings: {},
            },
        });
    };
}

const wait = async () => {
    return new Promise(r => setTimeout(r, 100));
};

const fetchLHSCounts = () => async (dispatch, getState) => {
    await wait();
    getReviews()(store.dispatch, store.getState);
    await wait();
    getUnreads()(store.dispatch, store.getState);
    await wait();
    getYourPrs()(store.dispatch, store.getState);
    await wait();
    getYourAssignments()(store.dispatch, store.getState);
}

export function handleReconnect(store, reminder = false) {
    return async () => {
        const {data} = await getConnected(reminder)(
            store.dispatch,
            store.getState,
        );
        if (data && data.connected) {
            fetchLHSCounts(store.dispatch, store.getState);
        }
    };
}

export function handleRefresh(store) {
    return () => {
        if (store.getState()[`plugins-${id}`].connected) {
            fetchLHSCounts(store.dispatch, store.getState);
        }
    };
}

export function handleChannelSubscriptionsUpdated(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }

        const data = JSON.parse(msg.data.payload);
        store.dispatch({
            type: ActionTypes.RECEIVED_CHANNEL_SUBSCRIPTIONS,
            data: {
                channelId: data.channel_id,
                subscriptions: data.subscriptions,
            },
        });
    };
}
