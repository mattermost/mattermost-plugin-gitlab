// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from '../issue_attribute_selector';

export type Actions = {
    getAssigneeOptions: (projectID?: number) =>  Promise<{
        error?: ErrorType;
        data?: Assignee[];
    }>
}

interface PropTypes {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedAssignees: SelectionType[];
    onChange: (assignees: OnChangeType) => void;
    actions: Actions;
};

export default class GitlabAssigneeSelector extends PureComponent<PropTypes> {
    loadAssignees = async () => {
        if (!this.props.projectName) {
            return [];
        }

        const options = await this.props.actions.getAssigneeOptions(this.props.projectID);

        if (options.error) {
            throw new Error('Failed to load assignees');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option: Assignee) => ({
            value: option.id,
            label: option.username,
        }));
    };

    render() {
        return (
            <div className='form-group margin-bottom x3'>
                <IssueAttributeSelector
                    {...this.props}
                    selection={this.props.selectedAssignees}
                    label='Assignees'
                    isMulti={true}
                    onChange={this.props.onChange}
                    loadOptions={this.loadAssignees}
                />
            </div>
        );
    }
}
