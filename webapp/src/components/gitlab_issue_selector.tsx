// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useCallback} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';
import {SingleValue} from 'react-select';

import {getStyleForReactSelect} from 'src/utils/styles';
import Client from 'src/client';
import Setting from './setting';

const searchDebounceDelay = 400;

interface PropTypes {
    name: string;
    required: boolean;
    theme: Theme;
    onChange: (value: Issue | null) => void;
    error: string;
    value: Issue | null;
    addValidate: (key: string, validateField: () => boolean) => void;
    removeValidate: (key: string) => void;
};

const GitlabIssueSelector = (props: PropTypes) => {
    const [invalid, setInvalid] = useState(false);
    const [error, setError] = useState('')

    const isValid = useCallback(() => {                
        if (!props.required) {
            return true;
        }
        
        const valid = Boolean(props.value);
        setInvalid(!valid);
        return valid;
    }, [props.value, props.required])

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
    }, [isValid])

    const handleIssueSearchTermChange = useCallback((inputValue: string) => {
        return debouncedSearchIssues(inputValue);
    }, []);

    const searchIssues = useCallback(async (text: string) => {
        const textEncoded = encodeURIComponent(text.trim().replace(/"/g, '\\"'));
        try {
            const issues = await Client.searchIssues(textEncoded);

            if (!Array.isArray(issues)) {
                return [];
            }

            return issues.map((issue) => {
                const projectParts = issue.web_url.split('/');
                let prefix = '';
                // Extract "username/projectName" from the issueURL parts
                if (projectParts.length >= 5) {
                    prefix = `${projectParts[projectParts.length - 5]}/${projectParts[projectParts.length - 4]}`;
                }
                return ({value: issue, label: `${prefix}, #${issue.iid}: ${issue.title}`});
            });
        } catch (e) {
            const err = e as ErrorType;
            setError(err.message);
            return [];
        }
    }, []);

    const debouncedSearchIssues = debounce(searchIssues, searchDebounceDelay);

    const onChange = useCallback((newValue: SingleValue<IssueSelection>) => {
        const value = newValue?.value ?? null;
        props.onChange(value);
    }, [props.onChange])

    const issueError = props.error ? (
        <p className='help-text error-text'>
            <span>{props.error}</span>
        </p>
    ) : null;

    const serverError = error ? (
        <p className='alert alert-danger'>
            <i
                className='fa fa-warning'
                title='Warning Icon'
            />
            <span>{error}</span>
        </p>
    ) : null;

    const requiredMsg = 'This field is required.';
    const validationError = props.required && invalid ? (
        <p className='help-text error-text'>
            <span>{requiredMsg}</span>
        </p>
    ) : null;

    return (
        <Setting
            inputId={props.name}
            label='Gitlab Issue'
            required={props.required}
        >
            <>
                {serverError}
                <AsyncSelect
                    name={'issue'}
                    placeholder={'Search for issues containing text...'}
                    onChange={onChange}
                    isMulti={false}
                    defaultOptions={true}
                    isClearable={true}
                    loadOptions={handleIssueSearchTermChange}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    styles={getStyleForReactSelect(props.theme)}
                />
                {validationError}
                {issueError}
                <div className={'help-text'}>
                    {'Returns issues sorted by most recently created.'} <br/>
                </div>
            </>
        </Setting>
    );
}

export default GitlabIssueSelector;
