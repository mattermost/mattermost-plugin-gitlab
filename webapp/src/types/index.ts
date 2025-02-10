// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Item} from './gitlab_items';

export type ConnectedData = {
    gitlab_url: string;
    connected: boolean;
    organization: string;
    gitlab_username: string;
    settings: UserSettingsData;
    gitlab_client_id: string;
}

export type LHSData = {
    reviews: Item[];
    yourAssignedPrs: Item[];
    yourAssignedIssues: Item[];
    todos: Item[];
}

export type UserSettingsData = {
    sidebar_buttons: string;
    daily_reminder: boolean;
    notifications: boolean;
}

export type GitlabUsersData = {
    username: string;
    last_try: number;
}

export type SubscriptionData = {
    repository_url: string;
    repository_name: string;
    features: string[];
    creator_id: string;
}

export type ShowRhsPluginActionData = {
    type: string;
    state: string;
    pluggableId: string;
}

export type APIError = {
    id?: string;
    message: string;
    status: number;
}

export type SideBarData = {
    username: string;
    reviewDetails: Item[];
    reviews: Item[];
    yourAssignedPrs: Item[];
    yourPrDetails: Item[];
    yourAssignedIssues: Item[];
    todos: Item[];
    org: string;
    gitlabURL: string;
    rhsState: string;
}

export type CreateIssueModalData = {
    postId?: string;
    title?: string;
    channelId?: string;
}
