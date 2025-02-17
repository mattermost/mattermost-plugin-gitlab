// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, expect, it, jest} from '@jest/globals';

import {isDesktopApp} from './user_agent';

describe('user_agent', () => {
    it("should return true if 'Mattermost' and 'Electron' are in userAgent", () => {
        jest.spyOn(window.navigator, 'userAgent', 'get').mockReturnValue('Mattermost Electron');
        expect(isDesktopApp()).toBe(true);
    });

    it("should return false if only 'Mattermost' present in userAgent", () => {
        jest.spyOn(window.navigator, 'userAgent', 'get').mockReturnValue('Mattermost');
        expect(isDesktopApp()).toBe(false);
    });

    it("should return false if only 'Electron' present in userAgent", () => {
        jest.spyOn(window.navigator, 'userAgent', 'get').mockReturnValue('Electron');
        expect(isDesktopApp()).toBe(false);
    });

    it("should return false if neither 'Mattermost' nor 'Electron' present in userAgent", () => {
        jest.spyOn(window.navigator, 'userAgent', 'get').mockReturnValue('Google chrome');
        expect(isDesktopApp()).toBe(false);
    });
});
