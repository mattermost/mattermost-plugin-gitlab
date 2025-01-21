// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Regex to match if a URl is valid merge request of issue URL
const gitlabRegexPattern = /https?:\/\/(www\.)?.*\/([\w.?-]+)\/([\w-]+)\/-\/([\w-]+)\/([\d-]+$)/g;

export const validateGitlabUrl = (url: string): boolean => {
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

export const getInfoAboutLink = (href: string, hostname: string) => {
    const linkInfo = href.split(`${hostname}/`)[1].split('/');
    if (linkInfo.length >= 5) {
        return {
            owner: linkInfo[0],
            repo: linkInfo[1],
            type: linkInfo[3],
            number: linkInfo[4],
        };
    }
    return {};
};
