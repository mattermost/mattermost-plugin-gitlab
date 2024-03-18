// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';
import {getAssigneeOptions} from 'src/actions';
import {useOptions} from 'src/hooks/use_options';

type PropTypes = {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedAssignees: SelectionType[];
    onChange: (assignees: OnChangeType) => void;
};

const GitlabAssigneeSelector = ({projectID, projectName, theme, selectedAssignees, onChange}: PropTypes) => {
    const returnType = ['id', 'username'];
    const errorMessage = 'failed to load assignees';

    const loadAssignees = useOptions(projectName, getAssigneeOptions as FetchIssueAttributeOptionsForProject, returnType, errorMessage, projectID);

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
};

export default GitlabAssigneeSelector;
