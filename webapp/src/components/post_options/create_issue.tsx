// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import GitLabIcon from 'src/images/icons/gitlab';

export default function CreateIssuePostMenuAction(): JSX.Element {
    return (
        <>
            <GitLabIcon
                type='menu'
                testId='gitlab-post-menu-create-issue'
            />
            {'Create GitLab Issue'}
        </>
    );
}
