const userAgent = window.navigator.userAgent;

export function isDesktopApp() {
    return userAgent.indexOf('Mattermost') !== -1 && userAgent.indexOf('Electron') !== -1;
}
