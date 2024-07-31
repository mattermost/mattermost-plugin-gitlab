import {useEffect, useRef} from 'react';
import {Post} from 'mattermost-redux/types/posts';

// Extension of https://stackoverflow.com/a/53446665
export const usePrevious = (value: string | Post | null | undefined) => {
    const ref: React.MutableRefObject<string | Post | null | undefined> = useRef();

    useEffect(() => {
        ref.current = value;
    }, [value]);
    return ref.current;
};
