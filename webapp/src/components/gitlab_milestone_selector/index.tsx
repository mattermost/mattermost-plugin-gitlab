// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback} from 'react';
import {useDispatch} from 'react-redux';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';
import {getMilestoneOptions} from 'src/actions';

type PropTypes = {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedMilestone: SelectionType | null;
    onChange: (milestone: OnChangeType) => void;
};

const GitlabMilestoneSelector = ({projectID, projectName, theme, selectedMilestone, onChange}: PropTypes) => {
    const dispatch = useDispatch();
    
    const loadMilestones = useCallback(async () => {
        if (!projectName) {
            return [];
        }

        const options = await getMilestoneOptions(projectID)(dispatch);

        if (options?.error) {
            throw new Error('failed to load milestones');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option: Milestone) => ({
            value: option.id,
            label: option.title,
        }));
    }, [projectID]);

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
}

export default GitlabMilestoneSelector;
