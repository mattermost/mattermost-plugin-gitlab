// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';
import {SingleValue} from 'react-select';

import {getStyleForReactSelect} from '../utils/styles';
import Client from '../client';
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

export const getUsernameAndProjectName=(webUrl: string) => {
    const projectParts = webUrl.split('/');

    // Extract "username/projectName" from the issueURL parts
    if (projectParts.length >= 5) {
        return`${projectParts[3]}/${projectParts[4]}`;
    }    
    return '';
}

const GitlabIssueSelector = ({name, required, theme, onChange, error, value, addValidate, removeValidate
}: PropTypes) => {
    const [invalid, setInvalid] = useState(false);
    const [responseError, setResponseError] = useState('')

    const isValid = () => {                
        if (!required) {
            return true;
        }
        
        const valid = Boolean(value);
        setInvalid(!valid);
        return valid;
    };

    useEffect(() => {
        return () => {            
            if (removeValidate && name) {
                removeValidate(name);
            }
        }
    }, [])

    useEffect(() => {
        if (addValidate && name) {            
            addValidate(name, isValid);
        }
        if (invalid) {            
            isValid();
        }
    }, [isValid])

    const handleIssueSearchTermChange = (inputValue: string) => {
        return debouncedSearchIssues(inputValue);
    };

    const searchIssues = async (text: string) => {
        const textEncoded = encodeURIComponent(text.trim().replace(/\\/g, '\\\\').replace(/"/g, '\\"'));
        try {
            const issues = await Client.searchIssues(textEncoded);

            if (!Array.isArray(issues)) {
                return [];
            }

            return issues.map((issue) => {
                return ({value: issue, label: `${getUsernameAndProjectName(issue.web_url)}, #${issue.iid}: ${issue.title}`});
            });
        } catch (e) {
            const err = e as ErrorType;
            setResponseError(err.message);
            return [];
        }
    };

    const debouncedSearchIssues = debounce(searchIssues, searchDebounceDelay);

    const handleOnChange = (newValue: SingleValue<IssueSelection>) => {
        const value = newValue?.value ?? null;
        onChange(value);
    };

    const issueError = error ? (
        <p className='help-text error-text'>
            <span>{error}</span>
        </p>
    ) : null;

    const serverError = responseError ? (
        <p className='alert alert-danger'>
            <i
                className='fa fa-warning'
                title='Warning Icon'
            />
            <span>{responseError}</span>
        </p>
    ) : null;

    const requiredMsg = 'This field is required.';
    const validationError = required && invalid ? (
        <p className='help-text error-text'>
            <span>{requiredMsg}</span>
        </p>
    ) : null;

    return (
        <Setting
            inputId={name}
            label='Gitlab Issue'
            required={required}
        >
            <>
                {serverError}
                <AsyncSelect
                    name={'issue'}
                    placeholder={'Search for issues containing text...'}
                    onChange={handleOnChange}
                    isMulti={false}
                    defaultOptions={true}
                    isClearable={true}
                    loadOptions={handleIssueSearchTermChange}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    styles={getStyleForReactSelect(theme)}
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
