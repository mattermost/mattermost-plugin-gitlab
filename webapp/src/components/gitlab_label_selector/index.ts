// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {ActionCreatorsMapObject, bindActionCreators, Dispatch} from 'redux';
import {ActionFunc, GenericAction} from 'mattermost-redux/types/actions';

import {getLabelOptions} from '../../actions';

import GitlabLabelSelector, {Actions} from './gitlab_label_selector';

const mapDispatchToProps = (dispatch: Dispatch<GenericAction>) => ({
    actions: bindActionCreators<ActionCreatorsMapObject<ActionFunc | GenericAction>, Actions>({getLabelOptions}, dispatch),
});

export default connect(null, mapDispatchToProps)(GitlabLabelSelector);
