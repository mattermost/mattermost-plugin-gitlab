// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

import {getConfig} from 'mattermost-redux/selectors/entities/general';

import SidebarHeader from './components/sidebar_header';
import TeamSidebar from './components/team_sidebar';
import RHSSidebar from './components/rhs_sidebar';
import UserAttribute from './components/user_attribute';
import Reducer from './reducers';
import SidebarRight from './components/sidebar_right';
import {getConnected, setShowRHSAction} from './actions';
import {
    handleConnect,
    handleDisconnect,
    handleReconnect,
    handleRefresh,
    handleChannelSubscriptionsUpdated,
} from './websocket';
import {id} from './manifest';
import Client from './client';
import {getPluginServerRoute} from './selectors';
import LinkTooltip from './components/link_tooltip';

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
        registry.registerLinkTooltipComponent(LinkTooltip);

        const {showRHSPlugin} = registry.registerRightHandSidebarComponent(SidebarRight, 'GitLab Plugin');
        store.dispatch(setShowRHSAction(() => store.dispatch(showRHSPlugin)));

        registry.registerWebSocketEventHandler(
            `custom_${id}_gitlab_connect`,
            handleConnect(store),
        );
        registry.registerWebSocketEventHandler(
            `custom_${id}_gitlab_disconnect`,
            handleDisconnect(store),
        );
        registry.registerWebSocketEventHandler(
            `custom_${id}_gitlab_refresh`,
            handleRefresh(store),
        );
        registry.registerWebSocketEventHandler(
            `custom_${id}_gitlab_channel_subscriptions_updated`,
            handleChannelSubscriptionsUpdated(store),
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

        // RHS Registration
        const {toggleRHSPlugin} = registry.registerRightHandSidebarComponent(RHSSidebar, 'GitLab');
        const boundToggleRHSAction = () => store.dispatch(toggleRHSPlugin);

        // App Bar icon
        if (registry.registerAppBarComponent) {
            const config = getConfig(store.getState());
            const siteUrl = (config && config.SiteURL) || '';
            const iconURL = `${siteUrl}/plugins/${id}/public/app-bar-icon.png`;
            registry.registerAppBarComponent(iconURL, boundToggleRHSAction, 'GitLab');
        }
    }

    deinitialize() {
        document.removeEventListener('click', activityFunc);
    }
}

global.window.registerPlugin(id, new PluginClass());
