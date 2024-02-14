export const validateGitlabUrl = (url: string): boolean => {
    const gitlabRegexPattern = /https?:\/\/(www\.)?.*\/([\w.?-]+)\/([\w-]+)\/-\/([\w-]+)\/([\d-]+$)/g;
    return gitlabRegexPattern.test(url);
};

export const isValidUrl = (urlString: string): boolean => {
    try {
        return Boolean(new URL(urlString));
    } catch {
        return false;
    }
};

export const getTruncatedText = (text: string, length: number): string => {
    let truncatedText = text.substring(0, length).trim();
    if (text.length > length) {
        truncatedText += '...';
    }
    return truncatedText;
};
