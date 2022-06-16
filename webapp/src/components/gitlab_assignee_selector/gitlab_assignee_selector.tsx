// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from '../issue_attribute_selector';
import {Assignee, AssigneeSelection} from '../../types/gitlab_assignee_selector'

interface PropTypes {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedAssignees: AssigneeSelection[];
    onChange: (assignees: AssigneeSelection[]) => void;
    actions: {
        getAssigneeOptions: (projectID: any) =>  Promise<{
            error: any;
            data?: undefined;
        } | {
            data: any;
            error?: undefined;
        }>
    };
};

export default class GitlabAssigneeSelector extends PureComponent<PropTypes> {
    loadAssignees = async () => {
        if (this.props.projectName === '') {
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
                    label= 'Assignees'
                    isMulti={true}
                    onChange={this.props.onChange}
                    selection={this.props.selectedAssignees}
                    loadOptions={this.loadAssignees}
                />
            </div>
        );
    }
}
