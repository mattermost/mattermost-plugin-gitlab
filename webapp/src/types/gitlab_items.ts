import {Theme} from 'mattermost-redux/types/preferences';
import * as CSS from 'csstype';

import {notificationReasons} from 'src/components/sidebar_right/gitlab_items';

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

export interface Project {
    path_with_namespace: string;
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
    labels_with_details?: Label[];
    target: Target;
    num_approvers: number;
    total_reviewers: number;
    reviewers: User[];
    body: string;
    state: string;
    type: string;
    repo: string;
    description: string;
}

export interface GitlabItemsProps {
    item: Item;
    theme: Theme;
}

export interface TooltipData {
    state: string;
    type: string;
    repo: string;
    description: string;
    created_at: string;
    iid: number;
    title: string;
    target_branch: string;
    source_branch: string;
    labels: string[];
    labels_with_details: Label[];
}

export interface ConnectedResponse {
    Connected: boolean;
    GitlabURL: string;
    Organization: string;
}

export interface PrDetails {
    iid: number;
    status: string | null;
    sha: string;
    num_approvers: number;
    project_id: number;
}

export interface IssueResponse {
    id: number;
    iid: number;
    external_id: string;
    state: string;
    description: string;
    health_status: string;
    author: any | null;
    milestone: any | null;
    project_id: number;
    assignees: any[];
    assignee: any | null;
    updated_at: string | null;
    closed_at: string | null;
    closed_by: any | null;
    title: string;
    created_at: string | null;
    moved_to_id: number;
    labels: string[];
    label_details: any[];
    upvotes: number;
    downvotes: number;
    due_date: string | null;
    web_url: string;
    references: any | null;
    time_stats: any | null;
    confidential: boolean;
    weight: number;
    discussion_locked: boolean;
    issue_type: string | null;
    subscribed: boolean;
    user_notes_count: number;
    links: any | null;
    issue_link_id: number;
    merge_requests_count: number;
    epic_issue_id: number;
    epic: any | null;
    iteration: any | null;
    task_completion_status: any | null;
}

interface Todo {
    id: number;
    project: any | null;
    author: BasicUser | null;
    action_name: any;
    target_type: any;
    target: any | null;
    target_url: string;
    body: string;
    state: string;
    created_at: string | null;
}

export interface LHSContent {
    yourAssignedPrs: MergeRequest[];
    reviews: MergeRequest[];
    yourAssignedIssues: IssueResponse[];
    todos: Todo[];
}

export interface GitlabUserResponse {
    username: string;
}

export interface MergeRequestResponse extends MergeRequest {
    labels_with_details?: Label[];
}

export interface SubscriptionResponse {
    repository_name: string;
    repository_url: string;
    features: string[];
    creator_id: string;
}

interface BasicUser {
    id: number;
    username: string;
    name: string;
    state: string;
    created_at: string | null;
    avatar_url: string;
    web_url: string;
}

interface MergeRequest {
    id: number;
    iid: number;
    target_branch: string;
    source_branch: string;
    project_id: number;
    title: string;
    state: string;
    created_at: string | null;
    updated_at: string | null;
    upvotes: number;
    downvotes: number;
    author: BasicUser | null;
    assignee: BasicUser | null;
    assignees: BasicUser[];
    reviewers: BasicUser[];
    source_project_id: number;
    target_project_id: number;
    labels: string[];
    label_details: any[];
    description: string;
    draft: boolean;
    work_in_progress: boolean;
    milestone: any | null;
    merge_when_pipeline_succeeds: boolean;
    detailed_merge_status: string;
    merge_error: string;
    merged_by: BasicUser | null;
    merged_at: string | null;
    closed_by: BasicUser | null;
    closed_at: string | null;
    subscribed: boolean;
    sha: string;
    merge_commit_sha: string;
    squash_commit_sha: string;
    user_notes_count: number;
    changes_count: string;
    should_remove_source_branch: boolean;
    force_remove_source_branch: boolean;
    allow_collaboration: boolean;
    web_url: string;
    references: any | null;
    discussion_locked: boolean;
    changes: any[];
    user: {
        can_merge: boolean;
    };
    time_stats: any | null;
    squash: boolean;
    pipeline: any | null;
    head_pipeline: any | null;
    diff_refs: {
        base_sha: string;
        head_sha: string;
        start_sha: string;
    };
    diverged_commits_count: number;
    rebase_in_progress: boolean;
    approvals_before_merge: number;
    reference: string;
    first_contribution: boolean;
    task_completion_status: any | null;
    has_conflicts: boolean;
    blocking_discussions_resolved: boolean;
    overflow: boolean;
    merge_status: string;
}
