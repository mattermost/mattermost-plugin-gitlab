// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

import React from 'react';

interface PropTypes {
    type: String;
}
export default class GitLabIcon extends React.PureComponent<PropTypes> {
    render() {
        const iconStyle = this.props.type === 'menu' ? {flex: '0 0 auto', width: '20px', height: '20px', borderRadius: '50px', padding: '2px'} : {};

        return (
            <span className='MenuItem__icon'>
                <svg
                    aria-hidden='true'
                    focusable='false'
                    role='img'
                    viewBox='0 0 24 24'
                    width='14'
                    height='14'
                    style={iconStyle}
                >
                    <path d='M21 14l-9 7l-9 -7l3 -11l3 7h6l3 -7z'/>
                </svg>
            </span>
        );
    }
}
