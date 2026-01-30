// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getConfig} from 'mattermost-redux/selectors/entities/general';
import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/common';
import {Store, Action} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import {getConnected as getConnectedState, getPluginServerRoute} from './selectors';
import SidebarHeader from './components/sidebar_header';
import TeamSidebar from './components/team_sidebar';
import UserAttribute from './components/user_attribute';
import CreateIssuePostMenuAction from './components/post_options/create_issue';
import AttachCommentToIssuePostMenuAction from './components/post_options/attach_comment_to_issue';
import CreateIssueModal from './components/modals/create_issue/create_issue_modal';
import AttachCommentToIssueModal from './components/modals/attach_comment_to_issue/attach_comment_to_issue_modal';
import GitLabRHS from './components/gitlab_rhs';
import LinkTooltip from './components/link_tooltip';

import {GlobalState} from './types/store';

import Reducer from './reducers';
import ActionTypes, {RHSViewType} from './action_types';
import {getConnected, openAttachCommentToIssueModal, openCreateIssueModal, setShowRHSAction, getLHSData, updateRHSState, setRHSViewType, getChannelSubscriptions} from './actions';
import {
    handleConnect,
    handleDisconnect,
    handleReconnect,
    handleRefresh,
    handleOpenCreateIssueModal,
    handleChannelSubscriptionsUpdated,
} from './websocket';
import manifest from './manifest';
import Client from './client';
import Hooks from './hooks';

import {FormatTextOptions, MessageHtmlToComponentOptions, PluginRegistry} from './types/mattermost-webapp';

let activityFunc: (() => void) | undefined;
let lastActivityTime = Number.MAX_SAFE_INTEGER;
const activityTimeout = 60 * 60 * 1000; // 1 hour
const {id} = manifest;

class PluginClass {
    async initialize(registry: PluginRegistry, store: Store<GlobalState, Action<any>>) {
        registry.registerReducer(Reducer);

        // This needs to be called before any API calls below
        Client.setServerRoute(getPluginServerRoute(store.getState()));

        await getConnected(true)(store.dispatch);

        registry.registerLeftSidebarHeaderComponent(SidebarHeader);
        registry.registerBottomTeamSidebarComponent(TeamSidebar);
        registry.registerPopoverUserAttributesComponent(UserAttribute);
        registry.registerRootComponent(CreateIssueModal);
        registry.registerPostDropdownMenuAction({
            text: CreateIssuePostMenuAction,
            action: (postId: string) => {
                store.dispatch(openCreateIssueModal(postId));
            },
            filter: (postId: string): boolean => {
                const state: GlobalState = store.getState();
                const post = getPost(state, postId);
                const isPostSystemMessage = Boolean(!post || isSystemMessage(post));

                return getConnectedState(state) && !isPostSystemMessage;
            },
        });
        registry.registerPostDropdownMenuAction({
            text: AttachCommentToIssuePostMenuAction,
            action: (postId: string) => {
                store.dispatch(openAttachCommentToIssueModal(postId));
            },
            filter: (postId: string): boolean => {
                const state: GlobalState = store.getState();
                const post = getPost(state, postId);
                const isPostSystemMessage = Boolean(!post || isSystemMessage(post));

                return getConnectedState(state) && !isPostSystemMessage;
            },
        });
        registry.registerRootComponent(AttachCommentToIssueModal);
        registry.registerLinkTooltipComponent(LinkTooltip);

        const hooks = new Hooks(store);
        registry.registerSlashCommandWillBePostedHook(hooks.slashCommandWillBePostedHook);

        // Register the unified RHS component that handles both views
        const {showRHSPlugin, toggleRHSPlugin} = registry.registerRightHandSidebarComponent(GitLabRHS, 'GitLab');

        // Store the showRHSPlugin action for use by sidebar buttons
        store.dispatch(setShowRHSAction(() => store.dispatch(showRHSPlugin)));

        // Helper to show RHS with subscriptions view (used by App Bar)
        const showSubscriptionsRHS = () => {
            store.dispatch(setRHSViewType(RHSViewType.SUBSCRIPTIONS));
            store.dispatch(toggleRHSPlugin);
        };

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
            `custom_${id}_create_issue`,
            handleOpenCreateIssueModal(store),
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

        // App Bar icon - opens subscriptions view
        if (registry.registerAppBarComponent) {
            const config = getConfig(store.getState());
            const siteUrl = (config && config.SiteURL) || '';
            const iconURL = `${siteUrl}/plugins/${id}/public/app-bar-icon.png`;
            registry.registerAppBarComponent(iconURL, showSubscriptionsRHS, 'GitLab');
        }

        // Popout support for the unified RHS component
        if (registry.registerRHSPluginPopoutListener) {
            registry.registerRHSPluginPopoutListener(id, (teamName, channelName, listeners) => {
                listeners.onMessageFromPopout((channel: string) => {
                    const pluginState = (store.getState() as any)[`plugins-${manifest.id}`];

                    if (channel === 'GET_POPOUT_STATE') {
                        // Send all state needed by the popout in a single message
                        listeners.sendToPopout('SEND_POPOUT_STATE', {
                            rhsViewType: pluginState.rhsViewType,
                            rhsState: pluginState.rhsState,
                            channelId: getCurrentChannelId(store.getState() as any),
                        });
                    }
                });
            });

            if (window.WebappUtils?.popouts?.isPopoutWindow()) {
                // Fetch fresh data via API
                store.dispatch(getLHSData() as any);

                // Set up listener for state from parent window
                window.WebappUtils.popouts.onMessageFromParent((channel: string, data: any) => {
                    if (channel === 'SEND_POPOUT_STATE') {
                        // Set which view to display
                        store.dispatch(setRHSViewType(data.rhsViewType));

                        // Set the tab state for SidebarRight view
                        if (data.rhsState) {
                            store.dispatch(updateRHSState(data.rhsState));
                        }

                        // Set channel ID and fetch subscriptions for subscriptions view
                        if (data.channelId) {
                            store.dispatch({
                                type: ActionTypes.SET_POPOUT_CHANNEL_ID,
                                channelId: data.channelId,
                            });
                            store.dispatch(getChannelSubscriptions(data.channelId) as any);
                        }
                    }
                });

                // Request state from parent window
                window.WebappUtils.popouts.sendToParent('GET_POPOUT_STATE');
            }
        }
    }

    deinitialize() {
        if (activityFunc) {
            document.removeEventListener('click', activityFunc);
        }
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: PluginClass): void
        PostUtils: {
            formatText(text: string, options?: FormatTextOptions): string,
            messageHtmlToComponent(html: string, isRHS: boolean, option?: MessageHtmlToComponentOptions): React.ReactNode,
        }
        WebappUtils?: {
            popouts?: {
                isPopoutWindow: () => boolean,
                onMessageFromParent: (callback: (channel: string, state: any) => void) => void,
                sendToParent: (channel: string, data?: any) => void,
            }
        }
    }
}

window.registerPlugin(manifest.id, new PluginClass());
