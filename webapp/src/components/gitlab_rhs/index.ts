// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import manifest from '../../manifest';

import GitLabRHS from './gitlab_rhs';

const {id} = manifest;

function mapStateToProps(state: any) {
    return {
        rhsViewType: state[`plugins-${id}`].rhsViewType,
    };
}

export default connect(mapStateToProps)(GitLabRHS);
