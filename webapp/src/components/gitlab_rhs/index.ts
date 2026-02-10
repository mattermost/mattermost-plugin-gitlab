// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {getTheme} from 'mattermost-redux/selectors/entities/preferences';

import {GlobalState} from '../../types/store';
import {getPluginState} from '../../selectors';

import GitLabRHS from './gitlab_rhs';

function mapStateToProps(state: GlobalState) {
    return {
        rhsViewType: getPluginState(state).rhsViewType,
        theme: getTheme(state),
    };
}

export default connect(mapStateToProps)(GitLabRHS);
