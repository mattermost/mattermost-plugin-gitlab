// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from '../issue_attribute_selector';

export type Actions = {
    getLabelOptions: (projectID?: number) =>  Promise<{
        error?: ErrorType;
        data?: Label[];
    }>
}

interface PropTypes {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedLabels: SelectionType[];
    onChange: (labels: OnChangeType) => void;
    actions: Actions;
};

export default class GitlabLabelSelector extends PureComponent<PropTypes> { 
    loadLabels = async () => {
        if (!this.props.projectName) {
            return [];
        }

        const options = await this.props.actions.getLabelOptions(this.props.projectID);

        if (options?.error) {
            throw new Error('failed to load labels');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option: Label) => ({
            value: option.name,
            label: option.name,
        }));
    };

    render() {
        return (
            <div className='form-group margin-bottom x3'>
                <IssueAttributeSelector
                    {...this.props}
                    selection={this.props.selectedLabels}
                    label='Labels'
                    isMulti={true}
                    onChange={this.props.onChange}
                    loadOptions={this.loadLabels}
                />
            </div>
        );
    }
}
