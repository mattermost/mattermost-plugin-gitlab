// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import GitLabIcon from 'src/images/icons/gitlab';

const AttachCommentToIssuePostMenuAction = () => {
    return (
        <span data-testid='gitlab-post-menu-attach-comment'>
            <GitLabIcon type='menu'/>
            {'Attach to GitLab Issue'}
        </span>
    );
};

export default AttachCommentToIssuePostMenuAction;
