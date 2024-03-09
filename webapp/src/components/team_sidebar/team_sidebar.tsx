import React, { FC } from 'react';

import SidebarButtons from '../sidebar_buttons';
import { Theme } from 'mattermost-redux/types/preferences';

interface TeamSidebarProps {
    show: boolean;
    theme: Theme;
}

const TeamSidebar: FC<TeamSidebarProps> = ({ show, theme }) => {
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