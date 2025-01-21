// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export interface IssueBody {
    title: string;
    description: string;
    project_id?: number;
    labels?: string[];
    assignees?: number[];
    milestone?: number;
    post_id: string;
    channel_id: string;
}

export interface Issue {
    iid: number;
    web_url: string;
    project_id: number;
}

export interface IssueSelection {
    value: Issue;
    label: string;
}

export interface CommentBody {
    project_id?: number;
    iid?: number;
    comment: string;
    post_id: string;
    web_url?: string;
}

export interface Assignee {
    id: number;
    username: string;
}

export interface Milestone{
    id: number;
    title: string;
}

export interface ProjectSelection {
    name: string;
    project_id?: number;
}

export interface Project{
    path_with_namespace: string;
    id: number;
}
