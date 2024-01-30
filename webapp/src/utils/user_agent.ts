export const isDesktopApp = () => {
    const userAgent = window.navigator.userAgent;
    return userAgent.indexOf('Mattermost') !== -1 && userAgent.indexOf('Electron') !== -1;
};
