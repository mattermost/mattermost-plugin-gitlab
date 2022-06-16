// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {ActionCreatorsMapObject, bindActionCreators, Dispatch} from 'redux';
import {ActionFunc, GenericAction} from 'mattermost-redux/types/actions';
import {GlobalState} from 'mattermost-redux/types/store';

import {id as pluginId} from '../../manifest';
import {getProjects} from '../../actions';

import {Project} from '../../types/gitlab_project_selector'
import GitlabProjectSelector from './gitlab_project_selector';


type plugin = "plugin"

interface pluginMethods {
    yourProjects: Project[];
}

interface CurrentState extends GlobalState {
    plugin: pluginMethods;
}

type Actions = {
    getProjects: () => Promise<{
        error: any;
        data?: undefined;
    } | {
        data: any;
        error?: undefined;
    }>;
};

function mapStateToProps(state: CurrentState) {
    return {
        yourProjects: state[`plugins-${pluginId}` as plugin].yourProjects,
    };
}

function mapDispatchToProps(dispatch: Dispatch<GenericAction>) {
    return {
        actions: bindActionCreators<ActionCreatorsMapObject<ActionFunc | GenericAction>, Actions>({getProjects}, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(GitlabProjectSelector);
