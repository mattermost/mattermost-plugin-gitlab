// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import GitLabIcon from 'src/images/icons/gitlab';

export default function CreateIssuePostMenuAction(): JSX.Element {
    return (
        <span data-testid='gitlab-post-menu-create-issue'>
            <GitLabIcon type='menu'/>
            {'Create GitLab Issue'}
        </span>
    );
}
