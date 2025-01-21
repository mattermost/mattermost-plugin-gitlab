// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {GenericAction} from 'mattermost-redux/types/actions';

import {Dispatch} from 'react';

import {Issue, Assignee, Milestone} from './gitlab_types';
import {Label} from './gitlab_items';

export type OnChangeType = SelectionType | SelectionType[] | null;

export type SelectionType = {
    value: number | string | Issue;
    label: string;
}

export type LabelSelectionType = {
    value: string;
    label: string;
}

export type MilestoneSelectionType = {
    value: number;
    label: string;
}

export type AssigneeSelectionType = {
    value: number;
    label: string;
}

export type ErrorType = {
    message: string;
}

export type pluginReduxStoreKey = 'plugins-com.github.manland.mattermost-plugin-gitlab'

export type AttributeType = Assignee | Milestone | Label;

export type FetchIssueAttributeOptionsForProject<T> = (projectID?: number) => (dispatch: Dispatch<GenericAction>) => Promise<{
    error?: ErrorType;
    data?: T[];
}>

export type ReactSelectOption = {
    value: Issue;
    label: string;
}
