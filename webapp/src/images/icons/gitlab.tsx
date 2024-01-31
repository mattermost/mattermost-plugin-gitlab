// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import React from 'react';

interface PropTypes {
    type: string;
}

const GitLabIcon = ({type}: PropTypes) => {
    const iconStyle = type === 'menu' ? {flex: '0 0 auto', width: '20px', height: '20px', padding: '2px', marginRight: '4px'} : {};

    return (
        <span className='MenuItem__icon'>
            <svg
                aria-hidden='true'
                focusable='false'
                role='img'
                viewBox='0 0 25 25'
                width='14'
                height='14'
                style={iconStyle}
            >
                <path
                    fill='#E24329'
                    d='m24.507 9.5-.034-.09L21.082.562a.896.896 0 0 0-1.694.091l-2.29 7.01H7.825L5.535.653a.898.898 0 0 0-1.694-.09L.451 9.411.416 9.5a6.297 6.297 0 0 0 2.09 7.278l.012.01.03.022 5.16 3.867 2.56 1.935 1.554 1.176a1.051 1.051 0 0 0 1.268 0l1.555-1.176 2.56-1.935 5.197-3.89.014-.01A6.297 6.297 0 0 0 24.507 9.5Z'
                />
                <path
                    fill='#FC6D26'
                    d='m24.507 9.5-.034-.09a11.44 11.44 0 0 0-4.56 2.051l-7.447 5.632 4.742 3.584 5.197-3.89.014-.01A6.297 6.297 0 0 0 24.507 9.5Z'
                />
                <path
                    fill='#FCA326'
                    d='m7.707 20.677 2.56 1.935 1.555 1.176a1.051 1.051 0 0 0 1.268 0l1.555-1.176 2.56-1.935-4.743-3.584-4.755 3.584Z'
                />
                <path
                    fill='#FC6D26'
                    d='M5.01 11.461a11.43 11.43 0 0 0-4.56-2.05L.416 9.5a6.297 6.297 0 0 0 2.09 7.278l.012.01.03.022 5.16 3.867 4.745-3.584-7.444-5.632Z'
                />
            </svg>
        </span>
    );
};

export default GitLabIcon;
