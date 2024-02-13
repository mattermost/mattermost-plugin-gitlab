import {describe, expect, it} from '@jest/globals';

import {validateGitlabURL, isValidUrl} from './tooltip_utils';

describe('validateGitlabURL should work as expected', () => {
    it('Should return true for valid GitLab repository URL', () => {
        const URL = 'https://gitlab.com/username/repo/-/merge_requests/1234';
        expect(validateGitlabURL(URL)).toBe(true);
    });

    it('Should return false for invalid GitLab repository URL', () => {
        const URL = 'https://github.com/username/repo';
        expect(validateGitlabURL(URL)).toBe(false);
    });

    it('Should return false for non-URL string input', () => {
        const URL = 'not a URL';
        expect(validateGitlabURL(URL)).toBe(false);
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
