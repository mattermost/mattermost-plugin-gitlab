// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import ReactSelect, {MultiValue, SingleValue} from 'react-select';
import {Theme} from 'mattermost-redux/types/preferences';

import {getStyleForReactSelect} from '../utils/styles';
import Setting from './setting';
import {LabelSelection} from 'src/types/gitlab_label_selector';
import {AssigneeSelection} from 'src/types/gitlab_assignee_selector';
import {MilestoneSelection} from 'src/types/gitlab_milestone_selector';

interface PropTypes {
    isMulti: boolean;
    projectName: string;
    theme: Theme;
    label: string;
    onChange: (value: any) => void;
    loadOptions: () => Promise<any>,
    selection: MilestoneSelection | AssigneeSelection[] | LabelSelection[] | null; 
};

interface StateTypes {
    options: any; 
    isLoading: boolean;
    error: string;
}

export default class IssueAttributeSelector extends PureComponent<PropTypes, StateTypes> {
    constructor(props: PropTypes) {
        super(props);
        this.state = {
            options: [],
            isLoading: false,
            error: '',
        };
    }

    componentDidMount() {
        if (this.props.projectName) {
            this.loadOptions();
        }
    }

    componentDidUpdate(prevProps: PropTypes) {
        if (this.props.projectName && prevProps.projectName !== this.props.projectName) {
            this.loadOptions();
        }
    }

    loadOptions = async () => {
        this.setState({isLoading: true});

        try {
            const options = await this.props.loadOptions();

            this.setState({
                options,
                isLoading: false,
                error: '',
            });
        } catch (err: any) {
            this.setState({
                options: [],
                error: err.message,
                isLoading: false,
            });
        }
    };

    onChange = (selection: MultiValue<AssigneeSelection | LabelSelection> | SingleValue<AssigneeSelection | LabelSelection>) => {
        if (this.props.isMulti) {
            this.props.onChange(selection ?? []);
        }
        this.props.onChange(selection);
    };

    render() {
        const noOptionsMessage = this.props.projectName ? 'No options' : 'Please select a project first';

        return (
            <Setting {...this.props}>
                <>
                    <ReactSelect
                        isMulti={this.props.isMulti}
                        isClearable={true}
                        placeholder={'Select...'}
                        noOptionsMessage={() => noOptionsMessage}
                        closeMenuOnSelect={!this.props.isMulti}
                        hideSelectedOptions={this.props.isMulti}
                        onChange={this.onChange}
                        options={this.state.options}
                        isLoading={this.state.isLoading}
                        styles={getStyleForReactSelect(this.props.theme)}
                    />
                    {this.state.error && (
                        <p className='alert alert-danger'>
                            <i
                                className='fa fa-warning'
                                title='Warning Icon'
                            />
                            <span> {this.state.error}</span>
                        </p>
                    )}
                </>
            </Setting>
        );
    }
}
