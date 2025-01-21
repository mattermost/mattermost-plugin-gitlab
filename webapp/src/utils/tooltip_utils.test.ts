// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, expect, it} from '@jest/globals';

import {validateGitlabUrl, isValidUrl} from './tooltip_utils';

describe('validateGitlabUrl should work as expected', () => {
    it('Should return true for valid GitLab repository URL', () => {
        const url = 'https://gitlab.com/username/repo/-/merge_requests/1234';
        expect(validateGitlabUrl(url)).toBe(true);
    });

    it('Should return false for invalid GitLab repository URL', () => {
        const url = 'https://github.com/username/repo';
        expect(validateGitlabUrl(url)).toBe(false);
    });

    it('Should return false for non-URL string input', () => {
        const url = 'not a URL';
        expect(validateGitlabUrl(url)).toBe(false);
    });
});

describe('isValidUrl should work as expected', () => {
    it('should return true for a valid URL', () => {
        expect(isValidUrl('https://mattermost.com')).toBe(true);
    });

    it.each(['abc@example.com', 'xyz', '://example.com', 'example.com'])('should return false for invalid URLs %s', (input) => {
        expect(isValidUrl(input)).toBe(false);
    });
});
