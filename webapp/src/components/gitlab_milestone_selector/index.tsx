// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';
import {getMilestoneOptions} from 'src/actions';
import {useOptions} from 'src/hooks/use_options';
import {Milestone} from 'src/types/gitlab_types';
import {FetchIssueAttributeOptionsForProject, MilestoneSelectionType, OnChangeType} from 'src/types/common';

type PropTypes = {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedMilestone: MilestoneSelectionType | null;
    onChange: (milestone: OnChangeType) => void;
};

const GitlabMilestoneSelector = ({projectID, projectName, theme, selectedMilestone, onChange}: PropTypes) => {
    const returnType: [string, string] = ['id', 'title'];
    const errorMessage = 'failed to load milestones';

    const loadMilestones = useOptions({projectName, getOptions: getMilestoneOptions as FetchIssueAttributeOptionsForProject<Milestone>, returnType, errorMessage, projectID});

    return (
        <div className='form-group margin-bottom x3'>
            <IssueAttributeSelector
                theme={theme}
                projectName={projectName}
                selection={selectedMilestone}
                label='Milestone'
                onChange={onChange}
                isMulti={false}
                loadOptions={loadMilestones}
            />
        </div>
    );
};

export default GitlabMilestoneSelector;
