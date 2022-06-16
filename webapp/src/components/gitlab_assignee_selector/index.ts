// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {ActionCreatorsMapObject, bindActionCreators, Dispatch} from 'redux';
import {ActionFunc, GenericAction} from 'mattermost-redux/types/actions';

import {getAssigneeOptions} from '../../actions';

import GitlabAssigneeSelector from './gitlab_assignee_selector';

type Actions = {
    getAssigneeOptions: (projectID: any) =>  Promise<{
        error: any;
        data?: undefined;
    } | {
        data: any;
        error?: undefined;
    }>
}

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>) => ({
    actions: bindActionCreators<ActionCreatorsMapObject<ActionFunc | GenericAction>, Actions>({getAssigneeOptions}, dispatch),
});

export default connect(null, mapDispatchToProps)(GitlabAssigneeSelector);
