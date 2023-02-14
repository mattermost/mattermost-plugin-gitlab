export const isValidUrl = (urlString) => {
    let url;
    try {
        url = new URL(urlString);
    } catch (e) {
        return false;
    }
    return url;
};
