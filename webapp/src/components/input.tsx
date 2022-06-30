// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react';

import Setting from './setting';

type PropTypes = {
    id: string;
    label: string;
    placeholder?: string;
    value: string;
    maxLength?: number;
    onChange: (s: string) => void;
    disabled?: boolean;
    required: boolean;
    readOnly?: boolean;
    type: string;
};

const Input = (props: PropTypes) => {   
    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>) => {
        props.onChange(e.target.value);
    }, [props.onChange]);

    const value = props.value ?? null;

    let input = null;
    if (props.type !== 'textarea') {
        input = (
            <input
                id={props.id}
                className='form-control'
                type={props.type === 'input' ? 'text' : 'number'}
                placeholder={props.placeholder}
                value={value}
                maxLength={props.maxLength}
                onChange={handleChange}
                disabled={props.disabled}
                readOnly={props.readOnly}
            />
        );
    } else {
        input = (
            <textarea
                style={{resize: 'none'}}
                id={props.id}
                className='form-control'
                rows= {5}
                placeholder={props.placeholder}
                value={value}
                maxLength={props.maxLength}
                onChange={handleChange}
                disabled={props.disabled}
                readOnly={props.readOnly}
            />
        );
    }

    return (
        <Setting
            label={props.label}
            inputId={props.id}
            required={props.required}
        >
            {input}
        </Setting>
    );
}

export default Input;
