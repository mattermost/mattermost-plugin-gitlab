// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react';
import {useDispatch} from 'react-redux';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';
import {getAssigneeOptions} from 'src/actions';

interface PropTypes {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedAssignees: SelectionType[];
    onChange: (assignees: OnChangeType) => void;
};

const GitlabAssigneeSelector = ({projectID, projectName, theme, selectedAssignees, onChange}: PropTypes) => {
    const dispatch = useDispatch();

    const loadAssignees = useCallback(async () => {
        if (!projectName) {
            return [];
        }

        const options = await getAssigneeOptions(projectID)(dispatch);

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
    }, [projectID]);

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
