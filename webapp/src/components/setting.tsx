// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

interface PropTypes {
    inputId?: string;
    label?: string;
    children?: JSX.Element | null;
    required?: boolean;
};

export default class Setting extends PureComponent<PropTypes> {
    render() {
        const {children, inputId, label, required} = this.props;

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
}
