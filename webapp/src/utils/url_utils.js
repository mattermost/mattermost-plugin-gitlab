export const isValidUrl = (urlString) => {
    try {
        return Boolean(new URL(urlString));
    } catch (e) {
        return false;
    }
};
