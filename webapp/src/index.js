// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import SidebarHeader from './components/sidebar_header';
import TeamSidebar from './components/team_sidebar';
import UserAttribute from './components/user_attribute';
import Reducer from './reducers';
import {getConnected} from './actions';
import {
    handleConnect,
    handleDisconnect,
    handleReconnect,
    handleRefresh,
} from './websocket';
import {id} from './manifest';
import Client from './client';
import {getPluginServerRoute} from './selectors';

let activityFunc;
let lastActivityTime = Number.MAX_SAFE_INTEGER;
const activityTimeout = 60 * 60 * 1000; // 1 hour

class PluginClass {
    async initialize(registry, store) {
        registry.registerReducer(Reducer);

        // This needs to be called before any API calls below
        Client.setServerRoute(getPluginServerRoute(store.getState()));

        await getConnected(true)(store.dispatch, store.getState);

        registry.registerLeftSidebarHeaderComponent(SidebarHeader);
        registry.registerBottomTeamSidebarComponent(TeamSidebar);
        registry.registerPopoverUserAttributesComponent(UserAttribute);

        registry.registerWebSocketEventHandler(
            `custom_${id}_gitlab_connect`,
            handleConnect(store)
        );
        registry.registerWebSocketEventHandler(
            `custom_${id}_gitlab_disconnect`,
            handleDisconnect(store)
        );
        registry.registerWebSocketEventHandler(
            `custom_${id}_gitlab_refresh`,
            handleRefresh(store)
        );
        registry.registerReconnectHandler(handleReconnect(store));

        activityFunc = () => {
            const now = new Date().getTime();
            if (now - lastActivityTime > activityTimeout) {
                handleReconnect(store, true)();
            }
            lastActivityTime = now;
        };

        document.addEventListener('click', activityFunc);
    }

    deinitialize() {
        document.removeEventListener('click', activityFunc);
    }
}

global.window.registerPlugin(id, new PluginClass());
