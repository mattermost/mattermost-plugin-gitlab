// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {ActionCreatorsMapObject, bindActionCreators, Dispatch} from 'redux';
import {ActionFunc, GenericAction} from 'mattermost-redux/types/actions';

import {getAssigneeOptions} from 'src/actions';

import GitlabAssigneeSelector, {Actions} from './gitlab_assignee_selector';

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>) => ({
    actions: bindActionCreators<ActionCreatorsMapObject<ActionFunc | GenericAction>, Actions>({getAssigneeOptions}, dispatch),
});

export default connect(null, mapDispatchToProps)(GitlabAssigneeSelector);
