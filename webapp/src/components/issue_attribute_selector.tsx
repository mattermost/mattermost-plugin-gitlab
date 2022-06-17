// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import ReactSelect, {OnChangeValue} from 'react-select';
import {Theme} from 'mattermost-redux/types/preferences';

import {getStyleForReactSelect} from '../utils/styles';
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

interface StateTypes {
    options: Array<SelectionType>; 
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
            this.filterSelection(options);
            this.setState({
                options,
                isLoading: false,
                error: '',
            });
        } catch (e) {
            this.filterSelection([]);
            const err = e as ErrorType;
            this.setState({
                options: [],
                error: err.message,
                isLoading: false,
            });
        }
    };

    filterSelection = (options: Array<SelectionType>) => {
        if (!this.props.selection) {
            return;
        }

        if (this.props.isMulti) {
            const selectionValues = (this.props.selection as SelectionType[]).map((s) => s.value)
            const filtered = options.filter((option) => selectionValues.includes(option.value));
            this.props.onChange(filtered);
            return;
        }

        for (const option of options) {
            if (option.value === (this.props.selection as SelectionType).value) {
                this.props.onChange(option);
                return;
            }
        }

        this.props.onChange(null);
    }

    onChangeHandler =  (newValue: OnChangeValue<OnChangeType, boolean>) => {
        this.props.onChange(newValue as OnChangeType)
    }

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
                        menuPortalTarget={document.body}
                        menuPlacement='auto'
                        hideSelectedOptions={this.props.isMulti}
                        onChange={this.onChangeHandler}
                        options={this.state.options}
                        isLoading={this.state.isLoading}
                        styles={getStyleForReactSelect(this.props.theme)}
                        value={this.props.selection}
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
