// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Theme} from 'mattermost-redux/types/preferences';
import { useDispatch } from 'react-redux';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';
import {getLabelOptions} from 'src/actions';

interface PropTypes {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedLabels: SelectionType[];
    onChange: (labels: OnChangeType) => void;
};

const GitlabLabelSelector = ({projectID, projectName, theme, selectedLabels, onChange}: PropTypes) => {
    const dispatch = useDispatch();
    
    const loadLabels = async () => {
        if (!projectName) {
            return [];
        }

        const options = await getLabelOptions(projectID)(dispatch);

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
