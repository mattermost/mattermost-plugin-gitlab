export function validateGitlabURL(url: string): boolean {
    const gitlabRegexPattern = /https?:\/\/(www\.)?.*\/([\w.?-]+)\/([\w-]+)\/-\/([\w-]+)\/([\d-]+$)/g;
    return gitlabRegexPattern.test(url);
}
