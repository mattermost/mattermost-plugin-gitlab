// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {MouseEvent} from 'react';
import GitLabIcon from 'src/images/icons/gitlab';

export type Actions = {
    open: (postId: string) => {
        type: string;
        data: {
            postId: string;
        };
    };
}

interface PropTypes {
    show: boolean;
    actions: Actions;
    postId: string;
}

const CreateIssuePostMenuAction = ({show, actions, postId}: PropTypes) => {
    const handleClick = (e: MouseEvent<HTMLButtonElement> | Event) => {        
        e.preventDefault();
        actions.open(postId);
    };

    if (!show) {
        return null;
    }

    const content = (
        <button
            className='style--none'
            role='presentation'
            onClick={handleClick}
        >
            <GitLabIcon type='menu'/>
            {'Create GitLab Issue'}
        </button>
    );

    return (
        <li
            className='MenuItem'
            role='menuitem'
        >
            {content}
        </li>
    );
}

export default CreateIssuePostMenuAction;
