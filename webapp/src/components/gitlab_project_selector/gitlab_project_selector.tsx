// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import {Project, ProjectSelection as Selection} from '../../types/gitlab_project_selector'
import {LabelSelection as ProjectSelection} from '../../types/gitlab_label_selector'
import ReactSelectSetting from '../react_select_setting';

interface PropTypes {
    yourProjects: Project[];
    theme: Theme;
    required: boolean;
    onChange: (project: Selection) => void;
    actions: {
        getProjects: () => Promise<{
            error: any;
            data?: undefined;
        } | {
            data: any;
            error?: undefined;
        }>;
    };
    value?: string;
    addValidate: (key: string, validateField: () => boolean) => void;
    removeValidate: (key: string) => void;
};

interface StateTypes {
    invalid: boolean;
    error: null | string;
    isLoading: boolean;
}

const initialState = {
    invalid: false,
    error: null,
    isLoading: false,
};

export default class GitlabProjectSelector extends PureComponent<PropTypes, StateTypes> { 
    constructor(props: PropTypes) {
        super(props);
        this.state = initialState;
    }

    componentDidMount() {
        this.loadProjects();
    }

    loadProjects = async () => {
        this.setState({isLoading:true})
        await this.props.actions.getProjects();
        this.setState({isLoading:false});
    }

    onChange = (_: string, name: string) => {
        const project = this.props.yourProjects.find((p: Project) => p.path_with_namespace === name);
        this.props.onChange({name, project_id: project?.id});
    }

    render() {
        const projectOptions = this.props.yourProjects.map((item: Project) => ({value: item.path_with_namespace, label: item.path_with_namespace}));
        
        return (
            <div className={'form-group margin-bottom x3'}>
                <ReactSelectSetting
                    name={'project'}
                    label={'Project'}
                    limitOptions={true}
                    required={true}
                    onChange={this.onChange}
                    options={projectOptions}
                    key={'project'}
                    isLoading={this.state.isLoading}
                    theme={this.props.theme}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                    value={projectOptions.find((option: ProjectSelection) => option.value === this.props.value)}
                />
                <div className={'help-text'}>
                    {'Returns GitLab projects connected to the user account'} <br/>
                </div>
            </div>
        );
    }
}
