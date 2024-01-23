export const isValidUrl = (urlString: string): boolean => {
    try {
        return Boolean(new URL(urlString));
    } catch (e) {
        return false;
    }
};
