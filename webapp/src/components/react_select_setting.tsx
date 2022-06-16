// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import ReactSelect, {SingleValue} from 'react-select';
import AsyncSelect from 'react-select/async';

import Setting from './setting';
import {getStyleForReactSelect} from '../utils/styles';
import {LabelSelection as ProjectSelection} from 'src/types/gitlab_label_selector';

const MAX_NUM_OPTIONS = 100;

interface PropTypes {
    name: string;
    onChange: (name: string, value: string) => void,
    label: string;
    theme: Theme;
    options: ProjectSelection[],
    isLoading: boolean;
    value?: ProjectSelection;
    addValidate: (key: string, validateField: () => boolean) => void;
    removeValidate: (key: string) => void;
    required: boolean;
    limitOptions: Boolean;
};

interface StateTypes {
    invalid: boolean;
}

export default class ReactSelectSetting extends PureComponent<PropTypes, StateTypes> {
    constructor(props: PropTypes) {
        super(props);
        this.state = {invalid: false};
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

    handleChange = (value: SingleValue<ProjectSelection>) => {             
        const newValue = value?.value ?? '';
        this.props.onChange(this.props.name, newValue);
    };

    filterOptions = (input: string) => {
        let options = this.props.options;
        if (input) {
            options = options.filter((x) => x.label.toLowerCase().includes(input.toLowerCase()));
        }

        return Promise.resolve(options.slice(0, MAX_NUM_OPTIONS));
    };

    isValid = () => {
        if (!this.props.required) {
            return true;
        }

        const valid = Boolean(this.props.value);

        this.setState({invalid: !valid});
        return valid;
    };

    render() {
        const requiredMsg = 'This field is required.';
        let validationError = null;
        if (this.props.required && this.state.invalid) {
            validationError = (
                <p className='help-text error-text'>
                    <span>{requiredMsg}</span>
                </p>
            );
        }

        let selectComponent = null;
        if (this.props.limitOptions && this.props.options.length > MAX_NUM_OPTIONS) {
            // The parent component has let us know that we may have a large number of options, and that
            // the dataset is static. In this case, we use the AsyncSelect component and synchronous func
            // this.filterOptions() to limit the number of options being rendered at a given time.
            selectComponent = (
                <AsyncSelect
                    loadOptions={this.filterOptions}
                    defaultOptions={true}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    onChange={this.handleChange}
                    isLoading={this.props.isLoading}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
            );
        } else {
            selectComponent = (
                <ReactSelect
                    options={this.props.options}
                    menuPortalTarget={document.body}
                    menuPlacement='auto'
                    isLoading={this.props.isLoading}
                    onChange={this.handleChange}
                    styles={getStyleForReactSelect(this.props.theme)}
                />
            );
        }

        return (
            <Setting
                inputId={this.props.name}
                {...this.props}
            >
                <>
                    {selectComponent}
                    {validationError}
                </>
            </Setting>
        );
    }
}
