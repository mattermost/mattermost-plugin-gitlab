export const isValidUrl = (urlString) => {
    try {
        return new URL(urlString);
    } catch (e) {
        return false;
    }
};
