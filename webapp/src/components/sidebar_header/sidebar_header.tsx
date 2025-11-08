// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent, ReactElement} from 'react';

import {Theme} from 'mattermost-redux/selectors/entities/preferences';

import SidebarButtons from '../sidebar_buttons';

interface SidebarHeaderProps {
    show: boolean;
    connected: boolean;
    theme: Theme;
}

export default class SidebarHeader extends PureComponent<SidebarHeaderProps> {
    render(): ReactElement | null {
        if (!this.props.show || !this.props.connected) {
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
