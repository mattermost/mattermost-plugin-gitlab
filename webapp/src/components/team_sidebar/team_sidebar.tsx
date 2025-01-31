// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {FC} from 'react';

import {Theme} from 'mattermost-redux/types/preferences';

import SidebarButtons from '../sidebar_buttons';

interface TeamSidebarProps {
    show: boolean;
    theme: Theme;
}

const TeamSidebar: FC<TeamSidebarProps> = ({show, theme}: TeamSidebarProps) => {
    if (!show) {
        return null;
    }

    return (
        <SidebarButtons
            theme={theme}
            isTeamSidebar={true}
        />
    );
};

export default TeamSidebar;
