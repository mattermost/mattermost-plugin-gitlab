// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from '../manifest';

const {id} = manifest;

// RHS view types - determines which content is shown in the single registered RHS component
export const RHSViewType = {
    SUBSCRIPTIONS: 'subscriptions', // Shows RHSSidebar content (channel subscriptions)
    SIDEBAR_RIGHT: 'sidebar_right', // Shows SidebarRight content (PRs, reviews, issues, todos)
} as const;

export type RHSViewTypeValue = typeof RHSViewType[keyof typeof RHSViewType];

export default {
    RECEIVED_YOUR_PR_DETAILS: `${id}_received_your_pr_details`,
    RECEIVED_REVIEW_DETAILS: `${id}_received_review_details`,
    RECEIVED_MENTIONS: `${id}_received_mentions`,
    RECEIVED_LHS_DATA: `${id}_received_lhs_data`,
    RECEIVED_CONNECTED: `${id}_received_connected`,
    RECEIVED_GITLAB_USER: `${id}_received_gitlab_user`,
    OPEN_CREATE_ISSUE_MODAL: `${id}_open_create_modal`,
    OPEN_CREATE_ISSUE_MODAL_WITHOUT_POST: `${id}_open_create_modal_without_post`,
    CLOSE_CREATE_ISSUE_MODAL: `${id}_close_create_modal`,
    CLOSE_ATTACH_COMMENT_TO_ISSUE_MODAL: `${id}_close_attach_modal`,
    OPEN_ATTACH_COMMENT_TO_ISSUE_MODAL: `${id}_open_attach_modal`,
    RECEIVED_PROJECTS: `${id}_received_projects`,
    RECEIVED_SHOW_RHS_ACTION: `${id}_received_rhs_action`,
    UPDATE_RHS_STATE: `${id}_update_rhs_state`,
    RECEIVED_CHANNEL_SUBSCRIPTIONS: `${id}_received_channel_subscriptions`,
    SET_POPOUT_CHANNEL_ID: `${id}_set_popout_channel_id`,
    SET_RHS_VIEW_TYPE: `${id}_set_rhs_view_type`,
};
