import {getUsernameAndProjectName} from './gitlab_issue_selector'

describe('getUsernameAndProjectName should work as expected', () => {
    test('Should return an empty string when the URL is invalid', () => {
      const web_url = 'https://invalid-url';
      const result = getUsernameAndProjectName(web_url);
      expect(result).toBe('');
    });
  
    test('Should return the correct project name when the URL is valid', () => {
      const web_url = 'https://gitlab.com/username/projectName/-/issues/1';
      const result = getUsernameAndProjectName(web_url);
      expect(result).toBe('username/projectName');
    });
  
    test('Should return an empty string when the URL does not contain the project name', () => {
      const web_url = 'https://gitlab.com/username';
      const result = getUsernameAndProjectName(web_url);
      expect(result).toBe('');
    });
});
