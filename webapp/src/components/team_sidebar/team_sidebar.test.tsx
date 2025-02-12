// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {describe, expect, test} from '@jest/globals';
import {render} from '@testing-library/react';
import {configureStore, createSlice} from '@reduxjs/toolkit';

import {Provider} from 'react-redux';

import TeamSidebar from './team_sidebar';

const mockTheme = {
    sidebarBg: '#ffffff',
    sidebarText: '#333333',
    sidebarUnreadText: '#ff0000',
    sidebarTextHoverBg: '#eeeeee',
    sidebarTextActiveBorder: '#007bff',
    sidebarTextActiveColor: '#007bff',
    sidebarHeaderBg: '#f8f9fa',
    sidebarHeaderTextColor: '#495057',
    onlineIndicator: '#28a745',
    awayIndicator: '#ffc107',
    dndIndicator: '#dc3545',
    mentionBg: '#ffeb3b',
    mentionBj: '#ffeb3b',
    mentionColor: '#333333',
    centerChannelBg: '#f8f9fa',
    centerChannelColor: '#333333',
    newMessageSeparator: '#007bff',
    linkColor: '#007bff',
    buttonBg: '#007bff',
    buttonColor: '#ffffff',
    errorTextColor: '#dc3545',
    mentionHighlightBg: '#ffeb3b',
    mentionHighlightLink: '#007bff',
    codeTheme: 'solarized-dark',
};

const mockSlice = createSlice({
    name: 'mock-reducer',
    initialState: {
        'plugins-com.github.manland.mattermost-plugin-gitlab': {
            connected: true,
            username: 'mattermost',
            clientId: '',
            lhsData: {
                reviews: [],
                yourAssignedPrs: [],
                yourAssignedIssues: [],
                todos: [],
            },
            gitlabURL: 'https://gitlab.com/gitlab-org',
            organization: '',
            rhsPluginAction: () => true,
        },
        entities: {general: {config: {}}},
    },
    reducers: {},
});

const mockStore = configureStore({
    reducer: mockSlice.reducer,
});

describe('TeamSidebar', () => {
    test.each([true, false])('should render when show is %s', (show) => {
        const {container} = render(
            <Provider store={mockStore}>
                <TeamSidebar
                    show={show}
                    theme={mockTheme}
                />
            </Provider>,
        );

        expect(container).toMatchSnapshot();
    });
});
