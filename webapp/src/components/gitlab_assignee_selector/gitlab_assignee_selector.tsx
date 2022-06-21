// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';

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

const GitlabAssigneeSelector = ({projectID, projectName, theme, selectedAssignees, onChange, actions}: PropTypes) => {
    const loadAssignees = async () => {
        if (!projectName) {
            return [];
        }

        const options = await actions.getAssigneeOptions(projectID);

        if (options?.error) {
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

    return (
        <div className='form-group margin-bottom x3'>
            <IssueAttributeSelector
                theme={theme}
                projectName={projectName}
                selection={selectedAssignees}
                label='Assignees'
                isMulti={true}
                onChange={onChange}
                loadOptions={loadAssignees}
            />
        </div>
    );
}

export default GitlabAssigneeSelector;
