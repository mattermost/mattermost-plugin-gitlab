// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import {RHSViewType, RHSViewTypeValue} from 'src/action_types';

import SidebarRight from '../sidebar_right';
import RHSSidebar from '../rhs_sidebar';

interface GitLabRHSProps {
    rhsViewType: RHSViewTypeValue;
    theme: Theme;
}

/**
 * Unified RHS component that switches between SidebarRight and RHSSidebar
 * based on the rhsViewType state. This solves the issue of Mattermost not
 * properly handling multiple RHS components from the same plugin in popouts.
 */
const GitLabRHS: React.FC<GitLabRHSProps> = ({rhsViewType, theme}: GitLabRHSProps) => {
    if (rhsViewType === RHSViewType.SIDEBAR_RIGHT) {
        return <SidebarRight theme={theme}/>;
    }

    // Default to subscriptions view (RHSSidebar)
    return <RHSSidebar/>;
};

export default GitLabRHS;
