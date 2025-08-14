// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import GitLabIcon from 'src/images/icons/gitlab';

const AttachCommentToIssuePostMenuAction = () => {
    return (
        <li
            className='MenuItem'
            role='menuitem'
        >
            <button
                className='style--none'
                role='presentation'
            >
                <GitLabIcon type='menu'/>
                {'Attach to GitLab Issue'}
            </button>
        </li>
    );
};

export default AttachCommentToIssuePostMenuAction;
