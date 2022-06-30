// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import IssueAttributeSelector from 'src/components/issue_attribute_selector';
import {getLabelOptions} from 'src/actions';
import {useOptions} from 'src/hooks/use_options';

type PropTypes = {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedLabels: SelectionType[];
    onChange: (labels: OnChangeType) => void;
};

const GitlabLabelSelector = ({projectID, projectName, theme, selectedLabels, onChange}: PropTypes) => {
    const returnType: any = ['name', 'name'];
    const errorMessage = 'failed to load labels';

    const loadLabels = useOptions(projectName, getLabelOptions as GetOptions, returnType , errorMessage, projectID);

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
