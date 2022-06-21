// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';

export type Actions = {
    getMilestoneOptions: (projectID?: number) =>  Promise<{
        error?: ErrorType;
        data?: Milestone[];
    }>
}

interface PropTypes {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedMilestone: SelectionType | null;
    onChange: (milestone: OnChangeType) => void;
    actions: Actions;
};

const GitlabMilestoneSelector = ({projectID, projectName, theme, selectedMilestone, onChange, actions}: PropTypes) => {
    const loadMilestones = async () => {
        if (!projectName) {
            return [];
        }

        const options = await actions.getMilestoneOptions(projectID);

        if (options?.error) {
            throw new Error('Failed to load milestones');
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option: Milestone) => ({
            value: option.id,
            label: option.title,
        }));
    };

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
