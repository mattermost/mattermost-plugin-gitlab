// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';

import debounce from 'debounce-promise';
import AsyncSelect from 'react-select/async';

import {getStyleForReactSelect} from 'utils/styles';
import Client from 'client';

const searchDebounceDelay = 400;

export default class GitlabIssueSelector extends PureComponent {
    static propTypes = {
        name: PropTypes.string,
        required: PropTypes.bool,
        theme: PropTypes.object.isRequired,
        onChange: PropTypes.func.isRequired,
        error: PropTypes.string,
        value: PropTypes.object,
        addValidate: PropTypes.func.isRequired,
        removeValidate: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            invalid: false,
            error: null,
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

    componentDidUpdate(prevProps, prevState) {
        if (prevState.invalid && this.props.value !== prevProps.value) {
            this.setState({invalid: false}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    handleIssueSearchTermChange = (inputValue) => {
        return this.debouncedSearchIssues(inputValue);
    };

    searchIssues = async (text) => {
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
        } catch (e) {
            this.setState({error: e.message});
            return [];
        }
    };

    debouncedSearchIssues = debounce(this.searchIssues, searchDebounceDelay);

    onChange = (e) => {
        const value = e?.value ?? '';
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
        const requiredStar = (
            <span
                className={'error-text'}
                style={{marginLeft: '3px'}}
            >
                {'*'}
            </span>
        );

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
            <div className={'form-group margin-bottom x3'}>
                {serverError}
                <label
                    className={'control-label'}
                    htmlFor={'issue'}
                >
                    {'GitLab Issue'}
                </label>
                {this.props.required && requiredStar}
                <AsyncSelect
                    name={'issue'}
                    placeholder={'Search for issues containing text...'}
                    onChange={this.onChange}
                    required={true}
                    disabled={false}
                    isMulti={false}
                    isClearable={true}
                    defaultOptions={true}
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
            </div>
        );
    }
}
