// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import GitLabIcon from 'src/images/icons/gitlab';

export default function CreateIssuePostMenuAction(): JSX.Element {
    return (
        <li
            className='MenuItem'
            role='menuitem'
        >
            <button
                className='style--none'
                role='presentation'
            >
                <span className='MenuItem__icon'>
                    <GitLabIcon type='menu'/>
                </span>
                {'Create GitLab Issue'}
            </button>
        </li>
    );
}
