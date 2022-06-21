// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';

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

const GitlabLabelSelector = ({projectID, projectName, theme, selectedLabels, onChange, actions}: PropTypes) => { 
    const loadLabels = async () => {
        if (!projectName) {
            return [];
        }

        const options = await actions.getLabelOptions(projectID);

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

    return (
        <div className='form-group margin-bottom x3'>
            <IssueAttributeSelector
                theme={theme}
                projectName={projectName}
                selection={selectedLabels}
                label='Labels'
                isMulti={true}
                onChange={onChange}
                loadOptions={loadLabels}
            />
        </div>
    );
}

export default GitlabLabelSelector;
