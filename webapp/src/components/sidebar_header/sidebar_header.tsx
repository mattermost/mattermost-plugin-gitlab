import React, {PureComponent, ReactElement} from 'react';

import SidebarButtons from '../sidebar_buttons';

import {Theme} from 'mattermost-redux/types/preferences';

interface SidebarHeaderProps{
    show: boolean,
    theme: Theme
}

export default class SidebarHeader extends PureComponent<SidebarHeaderProps> {

    render(): ReactElement | null {
        if (!this.props.show) {
            return null;
        }

        return (
            <SidebarButtons
                theme={this.props.theme}
                isTeamSidebar={false}
            />
        );
    }
}
