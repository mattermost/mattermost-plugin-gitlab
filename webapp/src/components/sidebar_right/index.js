// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';

import {bindActionCreators} from 'redux';

import {getYourPrDetails, getReviewDetails} from '../../actions';
import {id as pluginId} from '../../manifest';

import SidebarRight from './sidebar_right.tsx';

function mapPrsToDetails(prs, details) {
    if (!prs) {
        return [];
    }

    return prs.map((pr) => {
        let foundDetails;
        if (details) {
            foundDetails = details.find((prDetails) => {
                return (pr.project_id === prDetails.project_id) && (pr.sha === prDetails.sha);
            });
        }
        if (!foundDetails) {
            return pr;
        }

        return {
            ...pr,
            status: foundDetails.status,
            approvers: foundDetails.approvers,
            total_reviewers: pr.reviewers.length,
        };
    });
}

function mapStateToProps(state) {
    return {
        username: state[`plugins-${pluginId}`].username,
        reviews: mapPrsToDetails(state[`plugins-${pluginId}`].yourPrs, state[`plugins-${pluginId}`].reviewDetails),
        yourPrs: mapPrsToDetails(state[`plugins-${pluginId}`].yourPrs, state[`plugins-${pluginId}`].yourPrDetails),
        yourAssignments: state[`plugins-${pluginId}`].yourAssignments,
        unreads: state[`plugins-${pluginId}`].unreads,
        org: state[`plugins-${pluginId}`].organization,
        gitlabURL: state[`plugins-${pluginId}`].gitlabURL,
        rhsState: state[`plugins-${pluginId}`].rhsState,
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            getYourPrDetails,
            getReviewDetails,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(SidebarRight);
