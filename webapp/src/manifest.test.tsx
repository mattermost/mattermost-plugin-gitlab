import {test, expect} from '@jest/globals';

import manifest from './manifest';

test('Plugin manifest, id and version are defined', () => {
    expect(manifest).toBeDefined();
    expect(manifest.id).toBeDefined();
    expect(manifest.version).toBeDefined();
});
