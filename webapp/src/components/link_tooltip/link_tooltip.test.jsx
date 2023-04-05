import {getInfoAboutLink} from './link_tooltip';

describe('getInfoAboutLink should work as expected', () => {
    it('Should return an object of correct owner, repo, type, and number when given a valid GitLab hostname and href with all required information', () => {
        const href = 'https://gitlab.com/mattermost/test/-/merge_requests/123';
        const hostname = 'gitlab.com';
        const expected = {
            owner: 'mattermost',
            repo: 'test',
            type: 'merge_requests',
            number: '123',
        };

        const result = getInfoAboutLink(href, hostname);
        expect(result).toEqual(expected);
    });

    it('Should return an empty object when given a valid GitLab hostname and href missing the number', () => {
        const href = 'https://gitlab.com/mattermost/test/-/merge_requests';
        const hostname = 'gitlab.com';

        const result = getInfoAboutLink(href, hostname);
        expect(result).toEqual({});
    });

    it('Should return an empty object when given a valid GitLab hostname and href with missing the type parameter', () => {
        const href = 'https://gitlab.com/mattermost/test/123';
        const hostname = 'gitlab.com';

        const result = getInfoAboutLink(href, hostname);
        expect(result).toEqual({});
    });

    it('Should return an empty object when given an empty hostname and valid href', () => {
        const href = 'https://gitlab.com/mattermost/test/-/merge_requests/123';
        const hostname = '';

        const result = getInfoAboutLink(href, hostname);
        expect(result).toEqual({});
    });

    it('Should return an empty object when given a valid hostname and a invalid href', () => {
        const href = 'https://gitlab.com/mattermost/test/123Yes I think this is the right MR';
        const hostname = 'gitlab.com';

        const result = getInfoAboutLink(href, hostname);
        expect(result).toEqual({});
    });

    it('Should return an object valid owner, repo, type, and number with comment id when given a valid hostname and issue comment as href', () => {
        const href = 'https://gitlab.com/mattermost/test/-/issues/3#note_1340573704';
        const hostname = 'gitlab.com';
        const expected = {
            owner: 'mattermost',
            repo: 'test',
            type: 'issues',
            number: '3#note_1340573704',
        };

        const result = getInfoAboutLink(href, hostname);
        expect(result).toEqual(expected);
    });
});
