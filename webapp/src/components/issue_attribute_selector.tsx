// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useRef, useCallback} from 'react';
import ReactSelect, {OnChangeValue} from 'react-select';
import {Theme} from 'mattermost-redux/types/preferences';
import {Post} from 'mattermost-redux/types/posts';

import {getStyleForReactSelect} from 'src/utils/styles';
import Setting from './setting';

interface PropTypes {
    isMulti: boolean;
    projectName: string;
    theme: Theme;
    label: string;
    onChange: (value: OnChangeType) => void;
    loadOptions: () => Promise<Array<SelectionType>>,
    selection: OnChangeType;
};

export const UsePrevious = (value: string | Post | null | undefined) => {
    const ref: React.MutableRefObject<string | Post | null | undefined> = useRef();
    // Store current value in ref
    useEffect(() => {
      ref.current = value;
    }, [value]); // Only re-run if value changes
    // Return previous value (happens before update in useEffect above)
    return ref.current;
}

const IssueAttributeSelector = (props: PropTypes) => {
    const [options, setOptions] = useState<SelectionType[]>([]);
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [error, setError] = useState<string>('');

    useEffect(() => {
        if (props.projectName) {
            loadOptions();
        }
    }, [])

    const prevProjectName = UsePrevious(props.projectName)

    useEffect(() => {
        if (props.projectName && prevProjectName !== props.projectName) {
            loadOptions();
        }
    }, [props])

    const loadOptions = useCallback(async () => {
        setIsLoading(true);

        try {
            const options = await props.loadOptions();
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
    }, [props.loadOptions]);

    const filterSelection = useCallback((options: Array<SelectionType>) => {
        if (!props.selection) {
            return;
        }

        if (props.isMulti) {
            const selectionValues = (props.selection as SelectionType[]).map((s) => s.value)
            const filtered = options.filter((option) => selectionValues.includes(option.value));
            props.onChange(filtered);
            return;
        }

        for (const option of options) {
            if (option.value === (props.selection as SelectionType).value) {
                props.onChange(option);
                return;
            }
        }

        props.onChange(null);
    }, [props.selection, props.isMulti, props.onChange])

    const onChangeHandler =  useCallback((newValue: OnChangeValue<OnChangeType, boolean>) => {
        props.onChange(newValue as OnChangeType)
    }, [props.onChange]);

    const noOptionsMessage = props.projectName ? 'No options' : 'Please select a project first';

    return (
        <Setting {...props}>
            <>
                <ReactSelect
                    isMulti={props.isMulti}
                    isClearable={true}
                    placeholder={'Select...'}
                    noOptionsMessage={() => noOptionsMessage}
                    closeMenuOnSelect={!props.isMulti}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    hideSelectedOptions={props.isMulti}
                    onChange={onChangeHandler}
                    options={options}
                    isLoading={isLoading}
                    styles={getStyleForReactSelect(props.theme)}
                    value={props.selection}
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
