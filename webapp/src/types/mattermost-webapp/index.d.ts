// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

export type ContextArgs = {channel_id: string};

type FormatTextOptions = {
    atMentions?: boolean;
    markdown?: boolean;
}

type MessageHtmlToComponentOptions = {
    mentionHighlight: boolean;
}

export interface RHSPluginPopoutListeners {
    onMessageFromPopout: (callback: (channel: string) => void) => void;
    sendToPopout: (channel: string, data?: any) => void;
}

export interface PluginRegistry {
    registerReducer(reducer)
    registerPostTypeComponent(typeName: string, component: React.ElementType)
    registerRightHandSidebarComponent(component: React.ComponentType<any>, title: string | JSX.Element)
    registerSlashCommandWillBePostedHook(hook: (rawMessage: string, contextArgs: ContextArgs) => Promise<{}>)
    registerWebSocketEventHandler(event: string, handler: (msg: any) => void)
    registerAppBarComponent(iconUrl: string, action: () => void, tooltipText: string)
    registerLeftSidebarHeaderComponent(component: React.ComponentType<any>)
    registerBottomTeamSidebarComponent(component: React.ComponentType<any>)
    registerPopoverUserAttributesComponent(component: React.ComponentType<any>)
    registerLinkTooltipComponent(component: React.ComponentType<any>)
    registerReconnectHandler(handler: any)
    registerPostDropdownMenuComponent(component: React.ComponentType<any>)
    registerPostDropdownMenuAction(action: any)
    registerRootComponent(component: React.ComponentType<any>)
    registerRHSPluginPopoutListener?: (pluginId: string, callback: (teamName: string, channelName: string, listeners: RHSPluginPopoutListeners) => void) => void

    // Add more if needed from https://developers.mattermost.com/extend/plugins/webapp/reference
}
