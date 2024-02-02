import ActionTypes from '../action_types';
import Constants from '../constants';
import {
    getConnected,
    openCreateIssueModalWithoutPost,
    getLHSData,
} from '../actions';
import manifest from '../manifest';

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

export function handleReconnect(store, reminder = false) {
    return async () => {
        const {data} = await getConnected(reminder)(
            store.dispatch,
            store.getState,
        );
        if (data && data.connected) {
            getLHSData()(store.dispatch, store.getState);
        }
    };
}

export function handleRefresh(store) {
    return () => {
        if (store.getState()[`plugins-${manifest.id}`].connected) {
            getLHSData()(store.dispatch, store.getState);
        }
    };
}

export function handleOpenCreateIssueModal(store) {
    return (msg) => {
        if (!msg.data) {
            return;
        }
        store.dispatch(openCreateIssueModalWithoutPost(msg.data.title, msg.data.channel_id));
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
