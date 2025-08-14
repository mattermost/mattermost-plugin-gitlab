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

type PostDropdownMenuAction = {
    text: React.ReactNode;
    action: (postId: string) => void;
    filter: (postId: string) => boolean;
};

export interface PluginRegistry {
    registerReducer(reducer)
    registerPostTypeComponent(typeName: string, component: React.ElementType)
    registerRightHandSidebarComponent(component: React.ReactNode, title: string | JSX.Element)
    registerSlashCommandWillBePostedHook(hook: (rawMessage: string, contextArgs: ContextArgs) => Promise<{}>)
    registerWebSocketEventHandler(event: string, handler: (msg: any) => void)
    registerAppBarComponent(iconUrl: string, action: () => void, tooltipText: string)
    registerLeftSidebarHeaderComponent(component: React.ReactNode)
    registerBottomTeamSidebarComponent(component: React.ReactNode)
    registerPopoverUserAttributesComponent(component: React.ReactNode)
    registerLinkTooltipComponent(component: React.ReactNode)
    registerReconnectHandler(handler: any)
    registerPostDropdownMenuComponent(component: React.ReactNode)
    registerPostDropdownMenuAction(action: PostDropdownMenuAction)
    registerRootComponent(component: React.ElementType)

    // Add more if needed from https://developers.mattermost.com/extend/plugins/webapp/reference
}
