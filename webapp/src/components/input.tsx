// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

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

interface StateTypes {
    invalid: boolean;
}

export default class Input extends PureComponent<PropTypes, StateTypes> {   
    constructor(props: PropTypes) {
        super(props);
        this.state = {invalid: false};
    }

    handleChange = (e: React.ChangeEvent<HTMLInputElement> | React.ChangeEvent<HTMLTextAreaElement>) => {
        if (this.props.onChange) {
            this.props.onChange(e.target.value);
        }
    };

    render() {
        const value = this.props.value ?? '';

        let input = null;
        if (this.props.type === 'input') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type='text'
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        } else if (this.props.type === 'number') {
            input = (
                <input
                    id={this.props.id}
                    className='form-control'
                    type='number'
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        } else if (this.props.type === 'textarea') {
            input = (
                <textarea
                    style={{resize: 'none'}}
                    id={this.props.id}
                    className='form-control'
                    rows= {5}
                    placeholder={this.props.placeholder}
                    value={value}
                    maxLength={this.props.maxLength}
                    onChange={this.handleChange}
                    disabled={this.props.disabled}
                    readOnly={this.props.readOnly}
                />
            );
        }

        return (
            <Setting
                label={this.props.label}
                inputId={this.props.id}
                required={this.props.required}
            >
                {input}
            </Setting>
        );
    }
}
