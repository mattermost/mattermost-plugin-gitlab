// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useState} from 'react';
import ReactSelect, {SingleValue} from 'react-select';
import AsyncSelect from 'react-select/async';
import {Theme} from 'mattermost-redux/types/preferences';

import {getStyleForReactSelect} from 'src/utils/styles';
import Setting from './setting';

const MAX_NUM_OPTIONS = 100;

interface PropTypes {
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

const ReactSelectSetting = (props: PropTypes) => {
    const [invalid, setInvalid] = useState(false);

    useEffect(() => {
        return () => {
            if (props.removeValidate && props.name) {
                props.removeValidate(props.name);
            }
        }
    }, [])

    useEffect(() => {
        if (props.addValidate && props.name) {
            props.addValidate(props.name, isValid);
        }
        if (invalid) {
            isValid();
        }
    }, [props.value])

    const handleChange = useCallback((value: SingleValue<SelectionType>) => {             
        const newValue = value?.value ?? '';
        props.onChange(props.name, newValue as string);
    }, [props.onChange, props.name]);

    const filterOptions = useCallback((input: string) => {
        let options = props.options;
        if (input) {
            options = options.filter((x) => x.label.toLowerCase().includes(input.toLowerCase()));
        }

        return Promise.resolve(options.slice(0, MAX_NUM_OPTIONS));
    }, [props.options]);

    const isValid = useCallback(() => {
        if (!props.required) {
            return true;
        }

        const valid = Boolean(props.value);

        setInvalid(!valid);
        return valid;
    }, [props.value, props.required]);

    const requiredMsg = 'This field is required.';
    let validationError = null;
    if (props.required && invalid) {
        validationError = (
            <p className='help-text error-text'>
                <span>{requiredMsg}</span>
            </p>
        );
    }

    let selectComponent = null;
    if (props.limitOptions && props.options.length > MAX_NUM_OPTIONS) {
        // The parent component has let us know that we may have a large number of options, and that
        // the dataset is static. In this case, we use the AsyncSelect component and synchronous func
        // filterOptions() to limit the number of options being rendered at a given time.
        selectComponent = (
            <AsyncSelect
                loadOptions={filterOptions}
                defaultOptions={true}
                isClearable={true}
                menuPortalTarget={document.body}
                menuPlacement='auto'
                onChange={handleChange}
                isLoading={props.isLoading}
                styles={getStyleForReactSelect(props.theme)}
            />
        );
    } else {
        selectComponent = (
            <ReactSelect
                options={props.options}
                menuPortalTarget={document.body}
                menuPlacement='auto'
                isClearable={true}
                isLoading={props.isLoading}
                onChange={handleChange}
                styles={getStyleForReactSelect(props.theme)}
            />
        );
    }

    return (
        <Setting
            inputId={props.name}
            {...props}
        >
            <>
                {selectComponent}
                {validationError}
            </>
        </Setting>
    );
}

export default ReactSelectSetting;
