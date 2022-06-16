// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';
import {SingleValue} from 'react-select';

import {getStyleForReactSelect} from '../utils/styles';
import {Issue, IssueSelection} from 'src/types/attach_comment_to_issue';
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

interface StateTypes {
    invalid: boolean;
    error: string;
}

export default class GitlabIssueSelector extends PureComponent<PropTypes, StateTypes> {
    constructor(props: PropTypes) {
        super(props);
        this.state = {
            invalid: false,
            error: '',
        };
    }

    componentDidMount() {
        if (this.props.addValidate && this.props.name) {
            this.props.addValidate(this.props.name, this.isValid);
        }
    }

    componentWillUnmount() {
        if (this.props.removeValidate && this.props.name) {
            this.props.removeValidate(this.props.name);
        }
    }

    componentDidUpdate() {
        if (this.state.invalid) {
            this.isValid();
        }
    }

    handleIssueSearchTermChange = (inputValue: string) => {
        return this.debouncedSearchIssues(inputValue);
    };

    searchIssues = async (text: string) => {
        const textEncoded = encodeURIComponent(text.trim().replace(/"/g, '\\"'));
        try {
            const issues = await Client.searchIssues(textEncoded);

            if (!Array.isArray(issues)) {
                return [];
            }

            return issues.map((issue) => {
                const projectParts = issue.web_url.split('/');
                let prefix = '';
                if (projectParts.length >= 5) {
                    prefix = `${projectParts[projectParts.length - 5]}/${projectParts[projectParts.length - 4]}`;
                }
                return ({value: issue, label: `${prefix}, #${issue.iid}: ${issue.title}`});
            });
        } catch (e: any) {
            this.setState({error: e.message});
            return [];
        }
    };

    debouncedSearchIssues = debounce(this.searchIssues, searchDebounceDelay);

    onChange = (newValue: SingleValue<IssueSelection>) => {
        const value = newValue?.value ?? null;
        this.props.onChange(value);
    }

    isValid = () => {
        if (!this.props.required) {
            return true;
        }

        const valid = Boolean(this.props.value);
        this.setState({invalid: !valid});
        return valid;
    };

    render() {
        let issueError = null;
        if (this.props.error) {
            issueError = (
                <p className='help-text error-text'>
                    <span>{this.props.error}</span>
                </p>
            );
        }

        let serverError;
        if (this.state.error) {
            serverError = (
                <p className='alert alert-danger'>
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span>{this.state.error}</span>
                </p>
            );
        }

        const requiredMsg = 'This field is required.';
        let validationError = null;
        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        return (
            <Setting
                inputId={this.props.name}
                label='Gitlab Issue'
                required={this.props.required}
            >
                <>
                    {serverError}
                    <AsyncSelect
                        name={'issue'}
                        placeholder={'Search for issues containing text...'}
                        onChange={this.onChange}
                        isMulti={false}
                        defaultOptions={true}
                        isClearable={true}
                        loadOptions={this.handleIssueSearchTermChange}
                        menuPortalTarget={document.body}
                        menuPlacement='auto'
                        styles={getStyleForReactSelect(this.props.theme)}
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
}
