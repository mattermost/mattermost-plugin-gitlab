// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import Setting from './setting';

interface PropTypes {
    id: string;
    label: string;
    placeholder?: string;
    value: string;
    maxLength?: number;
    onChange?: (s: string) => void;
    disabled?: boolean;
    required: boolean;
    readOnly?: boolean;
    type: string;
};

const Input = (props: PropTypes) => {   
    const handleChange = (e: React.ChangeEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>) => {
        props.onChange?.(e.target.value);
    };

    const value = props.value ?? '';

    let input = null;
    if (props.type === 'input') {
        input = (
            <input
                id={props.id}
                className='form-control'
                type='text'
                placeholder={props.placeholder}
                value={value}
                maxLength={props.maxLength}
                onChange={handleChange}
                disabled={props.disabled}
                readOnly={props.readOnly}
            />
        );
    } else if (props.type === 'number') {
        input = (
            <input
                id={props.id}
                className='form-control'
                type='number'
                placeholder={props.placeholder}
                value={value}
                maxLength={props.maxLength}
                onChange={handleChange}
                disabled={props.disabled}
                readOnly={props.readOnly}
            />
        );
    } else if (props.type === 'textarea') {
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
