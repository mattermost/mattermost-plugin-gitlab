// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback, useMemo} from 'react';
import ReactSelect, {OnChangeValue} from 'react-select';
import {Theme} from 'mattermost-redux/types/preferences';

import {getStyleForReactSelect} from 'src/utils/styles';
import {usePrevious} from 'src/hooks/use_previous';
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
    }, [])

    const prevProjectName = usePrevious(projectName)

    useEffect(() => {
        if (projectName && prevProjectName !== projectName) {
            loadSelectOptions();
        }
    }, [projectName])

    const loadSelectOptions = useCallback(async () => {
        setIsLoading(true);

        try {
            const options = await loadOptions();
            filterSelection(options);
            setOptions(options);
            setIsLoading(false);
            setError('');
        } catch (e) {
            filterSelection([]);
            const err = e as ErrorType;
            setOptions([]);
            setIsLoading(false);
            setError(err.message);
        }
    }, [loadOptions]);

    const filterSelection = useCallback((options: Array<SelectionType>) => {
        if (!selection) {
            return;
        }

        if (isMulti) {
            const selectionValues = (selection as SelectionType[]).map((s) => s.value)
            const filtered = options.filter((option) => selectionValues.includes(option.value));
            onChange(filtered);
            return;
        }

        for (const option of options) {
            if (option.value === (selection as SelectionType).value) {
                onChange(option);
                return;
            }
        }

        onChange(null);
    }, [selection, isMulti, onChange])

    const onChangeHandler =  useCallback((newValue: OnChangeValue<OnChangeType, boolean>) => {
        onChange(newValue as OnChangeType)
    }, [onChange]);

    const noOptionsMessage = useMemo(() => projectName ? 'No options' : 'Please select a project first', [projectName]);

    return (
        <Setting
            label={label}
        >
            <>
                <ReactSelect
                    isMulti={isMulti}
                    isClearable={true}
                    placeholder={'Select...'}
                    noOptionsMessage={() => noOptionsMessage}
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
}

export default IssueAttributeSelector;
