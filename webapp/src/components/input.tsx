// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import Setting from './setting';

type PropTypes = {
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
    testId?: string;
};

const Input = ({id, label, placeholder, value, maxLength, onChange, disabled, required, readOnly, type, testId}: PropTypes) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>) => {
        onChange?.(e.target.value);
    };

    let input = null;
    if (type === 'textarea') {
        input = (
            <textarea
                style={{resize: 'none'}}
                id={id}
                data-testid={testId}
                className='form-control'
                rows={5}
                placeholder={placeholder}
                value={value ?? null}
                maxLength={maxLength}
                onChange={handleChange}
                disabled={disabled}
                readOnly={readOnly}
            />
        );
    } else {
        input = (
            <input
                id={id}
                data-testid={testId}
                className='form-control'
                type={type === 'input' ? 'text' : 'number'}
                placeholder={placeholder}
                value={value ?? null}
                maxLength={maxLength}
                onChange={handleChange}
                disabled={disabled}
                readOnly={readOnly}
            />
        );
    }

    return (
        <Setting
            label={label}
            inputId={id}
            required={required}
        >
            {input}
        </Setting>
    );
};

export default Input;
