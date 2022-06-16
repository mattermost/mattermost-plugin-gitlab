// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';

import {Milestone, MilestoneSelection} from '../../types/gitlab_milestone_selector'
import IssueAttributeSelector from '../issue_attribute_selector';

interface PropTypes {
    projectID?: number;
    projectName: string;
    theme: Theme;
    selectedMilestone: MilestoneSelection | null;
    onChange: (milestone: MilestoneSelection) => void;
    actions: {
        getMilestoneOptions: (projectID: any) =>  Promise<{
            error: any;
            data?: undefined;
        } | {
            data: any;
            error?: undefined;
        }>
    };
};

export default class GitlabMilestoneSelector extends PureComponent<PropTypes> {
    loadMilestones = async () => {
        if (this.props.projectName === '') {
            return [];
        }

        const options = await this.props.actions.getMilestoneOptions(this.props.projectID);

        if (options.error) {
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

    render() {
        return (
            <div className='form-group margin-bottom x3'>
                <IssueAttributeSelector
                    {...this.props}
                    label='Milestone'
                    onChange={this.props.onChange}
                    isMulti={false}
                    selection={this.props.selectedMilestone}
                    loadOptions={this.loadMilestones}
                />
            </div>
        );
    }
}
