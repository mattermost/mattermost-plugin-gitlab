// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getConfig} from 'mattermost-redux/selectors/entities/general';
import {Store, Action} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import {useDispatch} from 'react-redux';

import {getConnected as getConnectedState, getPluginServerRoute} from './selectors';
import SidebarHeader from './components/sidebar_header';
import TeamSidebar from './components/team_sidebar';
import RHSSidebar from './components/rhs_sidebar';
import UserAttribute from './components/user_attribute';
import CreateIssuePostMenuAction from './components/post_options/create_issue';
import AttachCommentToIssuePostMenuAction from './components/post_options/attach_comment_to_issue';
import CreateIssueModal from './components/modals/create_issue/create_issue_modal';
import AttachCommentToIssueModal from './components/modals/attach_comment_to_issue/attach_comment_to_issue_modal';
import SidebarRight from './components/sidebar_right';
import LinkTooltip from './components/link_tooltip';

import {GlobalState} from './types/store';

import Reducer from './reducers';
import {getConnected, openAttachCommentToIssueModal, openCreateIssueModal, setShowRHSAction} from './actions';
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
                const dispatch = useDispatch();
                dispatch(openCreateIssueModal(postId));
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
                const dispatch = useDispatch();
                dispatch(openAttachCommentToIssueModal(postId));
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
    }
}

window.registerPlugin(manifest.id, new PluginClass());
