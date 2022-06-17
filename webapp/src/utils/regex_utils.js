export function validateGitlabURL(url) {
    const gitlabRegexPattern = /https?:\/\/(www\.)?gitlab\.com\/([\w.?-]+)\/([\w-]+)\/-\/([\w-]+)\/([\d-]+)/g;
    return gitlabRegexPattern.test(url);
}
