// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect} from 'react';
import ReactSelect, {OnChangeValue} from 'react-select';
import {Theme} from 'mattermost-redux/types/preferences';

import {getStyleForReactSelect} from 'src/utils/styles';
import {usePrevious} from 'src/hooks/use_previous';

import {OnChangeType, SelectionType, ErrorType} from 'src/types/common';

import Setting from './setting';

type PropTypes = {
    isMulti: boolean;
    projectName: string;
    theme: Theme;
    label: string;
    onChange: (value: OnChangeType) => void;
    loadOptions: () => Promise<Array<SelectionType>>,
    selection: OnChangeType;
};

const IssueAttributeSelector = ({isMulti, projectName, theme, label, onChange, loadOptions, selection}: PropTypes) => {
    const [options, setOptions] = useState<SelectionType[]>([]);
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [error, setError] = useState<string>('');

    useEffect(() => {
        if (projectName) {
            loadSelectOptions();
        }
    }, [projectName]);

    const prevProjectName = usePrevious(projectName);

    useEffect(() => {
        if (projectName && prevProjectName !== projectName) {
            loadSelectOptions();
        }
    }, [projectName]);

    const loadSelectOptions = async () => {
        setIsLoading(true);

        try {
            const loadedOptions = await loadOptions();
            filterSelection(loadedOptions);
            setOptions(loadedOptions);
            setIsLoading(false);
            setError('');
        } catch (e) {
            filterSelection([]);
            const err = e as ErrorType;
            setOptions([]);
            setIsLoading(false);
            setError(err.message);
        }
    };

    const filterSelection = (loadedOptions: Array<SelectionType>) => {
        if (!selection) {
            return;
        }

        if (isMulti) {
            const selectionValues = (selection as SelectionType[]).map((s) => s.value);
            const filtered = loadedOptions.filter((option) => selectionValues.includes(option.value));
            onChange(filtered);
            return;
        }

        for (const option of loadedOptions) {
            if (option.value === (selection as SelectionType).value) {
                onChange(option);
                return;
            }
        }

        onChange(null);
    };

    const onChangeHandler = (newValue: OnChangeValue<OnChangeType, boolean>) => {
        onChange(newValue as OnChangeType);
    };

    return (
        <Setting
            label={label}
        >
            <>
                <ReactSelect
                    isMulti={isMulti}
                    isClearable={true}
                    placeholder={'Select...'}
                    noOptionsMessage={() => (projectName ? 'No options' : 'Please select a project first')}
                    closeMenuOnSelect={!isMulti}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    hideSelectedOptions={isMulti}
                    onChange={onChangeHandler}
                    options={options}
                    isLoading={isLoading}
                    styles={getStyleForReactSelect(theme)}
                    value={selection}
                />
                {error && (
                    <p className='alert alert-danger'>
                        <i
                            className='fa fa-warning'
                            title='Warning Icon'
                        />
                        <span> {error}</span>
                    </p>
                )}
            </>
        </Setting>
    );
};

export default IssueAttributeSelector;
