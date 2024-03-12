import React from 'react';

import {describe, expect, test, jest} from '@jest/globals';
import {render} from '@testing-library/react';

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

jest.mock('../sidebar_buttons', () => ({
    __esModule: true,
    default: () => <div data-testid='mocked-sidebar-buttons'/>,
}));

describe('TeamSidebar', () => {
    test.each([true, false])('should render when show is %s', (show) => {
        const {container} = render(
            <TeamSidebar
                show={show}
                theme={mockTheme}
            />);
        expect(container).toMatchSnapshot();
    });
});
