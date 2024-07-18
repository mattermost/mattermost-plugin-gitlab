// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';
import ReactSelect, {SingleValue} from 'react-select';
import AsyncSelect from 'react-select/async';
import {Theme} from 'mattermost-redux/types/preferences';

import {getStyleForReactSelect} from 'src/utils/styles';

import {SelectionType} from 'src/types/common';

import Setting from './setting';

const MAX_NUM_OPTIONS = 100;

type PropTypes = {
    name: string;
    onChange: (name: string, value: string) => void,
    label: string;
    theme: Theme;
    options: SelectionType[],
    isLoading: boolean;
    value?: SelectionType;
    addValidate: (key: string, validateField: () => boolean) => void;
    removeValidate: (key: string) => void;
    required: boolean;
    limitOptions: boolean;
};

const ReactSelectSetting = ({name, onChange, label, theme, options, isLoading, value, addValidate, removeValidate, required, limitOptions}: PropTypes) => {
    const [invalid, setInvalid] = useState(false);

    useEffect(() => {
        return () => {
            if (removeValidate && name) {
                removeValidate(name);
            }
        };
    }, []);

    useEffect(() => {
        if (addValidate && name) {
            addValidate(name, isValid);
        }
        if (invalid) {
            isValid();
        }
    }, [value]);

    const handleChange = (newValue: SingleValue<SelectionType>) => {
        const updatedValue = newValue?.value ?? '';
        onChange(name, updatedValue as string);
    };

    const filterOptions = (input: string) => {
        let filteredOptions: SelectionType[] = [];
        if (input) {
            filteredOptions = options.filter((x) => x.label.toLowerCase().includes(input.toLowerCase()));
        }

        return Promise.resolve(filteredOptions.slice(0, MAX_NUM_OPTIONS));
    };

    const isValid = () => {
        if (!required) {
            return true;
        }

        const valid = Boolean(value);

        setInvalid(!valid);
        return valid;
    };

    const requiredMsg = 'This field is required.';
    let validationError = null;
    if (required && invalid) {
        validationError = (
            <p className='help-text error-text'>
                <span>{requiredMsg}</span>
            </p>
        );
    }

    let selectComponent = null;
    if (limitOptions && options.length > MAX_NUM_OPTIONS) {
        // The parent component help us know that we may have a large number of options, and that
        // the data-set is static. In this case, we use the AsyncSelect component and synchronous func
        // "filterOptions" to limit the number of options being rendered at a given time.
        selectComponent = (
            <AsyncSelect
                loadOptions={filterOptions}
                defaultOptions={true}
                isClearable={true}
                menuPortalTarget={document.body}
                menuPlacement='auto'
                onChange={handleChange}
                isLoading={isLoading}
                styles={getStyleForReactSelect(theme)}
            />
        );
    } else {
        selectComponent = (
            <ReactSelect
                options={options}
                menuPortalTarget={document.body}
                menuPlacement='auto'
                isClearable={true}
                isLoading={isLoading}
                onChange={handleChange}
                styles={getStyleForReactSelect(theme)}
            />
        );
    }

    return (
        <Setting
            inputId={name}
            label={label}
            required={required}
        >
            <>
                {selectComponent}
                {validationError}
            </>
        </Setting>
    );
};

export default ReactSelectSetting;
