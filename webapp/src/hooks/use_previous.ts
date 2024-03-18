import {useEffect, useRef} from 'react';
import {Post} from 'mattermost-redux/types/posts';

export const usePrevious = (value: string | Post | null | undefined) => {
    const ref: React.MutableRefObject<string | Post | null | undefined> = useRef();

    // Store current value in ref
    useEffect(() => {
        ref.current = value;
    }, [value]); // Only re-run if value changes
    // Return previous value (happens before update in useEffect above)
    return ref.current;
};
