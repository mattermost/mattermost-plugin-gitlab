export function validateGitlabURL(url) {
    const gitlabRegexPattern = /https?:\/\/(www\.)?.*\/([\w.?-]+)\/([\w-]+)\/-\/([\w-]+)\/([\d-]+$)/g;
    return gitlabRegexPattern.test(url);
}
