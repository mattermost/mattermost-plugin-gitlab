// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {ActionCreatorsMapObject, bindActionCreators, Dispatch} from 'redux';
import {ActionFunc, GenericAction} from 'mattermost-redux/types/actions';

import {getMilestoneOptions} from 'src/actions';

import GitlabMilestoneSelector, {Actions} from './gitlab_milestone_selector';

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>) => ({
    actions: bindActionCreators<ActionCreatorsMapObject<ActionFunc | GenericAction>, Actions>({getMilestoneOptions}, dispatch),
});

export default connect(null,mapDispatchToProps)(GitlabMilestoneSelector);
