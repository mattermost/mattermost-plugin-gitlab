// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

type PropTypes = {
    inputId?: string;
    label?: string;
    children?: JSX.Element | null;
    required?: boolean;
};

const Setting = ({children, inputId, label, required}: PropTypes) => {
    return (
        <div className='form-group less'>
            {label && (
                <label
                    className='control-label margin-bottom x2'
                    htmlFor={inputId}
                >
                    {label}
                </label>)
            }
            {required && (
                <span
                    className='error-text'
                    style={{marginLeft: '3px'}}
                >
                    {'*'}
                </span>
            )
            }
            <div>
                {children}
            </div>
        </div>
    );
}

export default Setting;
