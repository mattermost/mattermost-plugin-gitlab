// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export function formatTimeSince(date: string): string {
    const pastDateInSeconds = new Date(date).getTime();
    const secondsSince = Math.trunc((Date.now() - pastDateInSeconds) / 1000);
    if (secondsSince < 60) {
        return `${secondsSince} ${secondsSince === 1 ? 'second' : 'seconds'}`;
    }
    const minutesSince = Math.trunc(secondsSince / 60);
    if (minutesSince < 60) {
        return `${minutesSince} ${minutesSince === 1 ? 'minute' : 'minutes'}`;
    }
    const hoursSince = Math.trunc(minutesSince / 60);
    if (hoursSince < 24) {
        return `${hoursSince} ${hoursSince === 1 ? 'hour' : 'hours'}`;
    }
    const daysSince = Math.trunc(hoursSince / 24);
    return `${daysSince} ${daysSince === 1 ? 'day' : 'days'}`;
}
