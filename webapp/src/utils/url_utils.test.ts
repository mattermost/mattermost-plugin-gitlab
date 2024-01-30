import {describe, expect, it} from '@jest/globals';

import {isValidUrl} from './url_utils';

describe('url_utils', () => {
    it('should return true for a valid URL', () => {
        expect(isValidUrl('https://mattermost.com')).toBe(true);
    });

    it.each(['abc@example.com', 'xyz', '://example.com', 'example.com'])('should return false for invalid URLs %s', (input) => {
        expect(isValidUrl(input)).toBe(false);
    });
});
