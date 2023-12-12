const userAgent = window.navigator.userAgent;

export const isDesktopApp = () => userAgent.indexOf('Mattermost') !== -1 && userAgent.indexOf('Electron') !== -1;
