// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Theme} from 'mattermost-redux/selectors/entities/preferences';
import * as CSS from 'csstype';

import {notificationReasons} from 'src/components/sidebar_right/gitlab_items';

import {Project} from './gitlab_types';

export interface Label {
    id: number;
    name: string;
    color: CSS.Properties;
    text_color: CSS.Properties;
    description: string;
}

export interface User {
    username: string;
    web_url: string;
    name: string;
}

export interface References {
    full: string;
}

export interface Target {
    title: string;
}

export interface Item {
    url: string;
    iid: number;
    project_id: number;
    sha: string;
    has_conflicts: boolean;
    id: number;
    status: string;
    title: string;
    created_at: string;
    updated_at: string;
    action_name: keyof typeof notificationReasons;
    web_url: string;
    target_url: string;
    repository_url?: string;
    author: User;
    references: References;
    project: Project;
    merge_status: string;
    merge_error: string;
    owner?: User;
    milestone?: {
        title: string;
    };
    repository?: {
        full_name: string;
    };
    label_details?: Label[];
    target: Target;
    num_approvers: number;
    total_reviewers: number;
    reviewers: User[];
    body: string;
    state: string;
    type: string;
    repo: string;
    description: string;
    target_branch: string;
    source_branch: string;
    labels: string[];
}

export interface GitlabItemsProps {
    item: Item;
    theme: Theme;
}
