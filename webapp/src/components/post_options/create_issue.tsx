// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {MouseEvent} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import GitLabIcon from 'src/images/icons/gitlab';
import manifest from 'src/manifest';
import {openCreateIssueModal} from 'src/actions';
import {GlobalState} from 'src/types/global_state';
import {isUserConnectedToGitlab} from 'src/selectors';

type PropTypes = {
    postId: string;
}

const CreateIssuePostMenuAction = ({postId}: PropTypes) => {
    const {show} = useSelector((state: GlobalState) => {
        const post = getPost(state, postId);
        const isPostSystemMessage = Boolean(!post || isSystemMessage(post));

        return {
            show: isUserConnectedToGitlab(state) && !isPostSystemMessage,
        };
    });

    const dispatch = useDispatch();

    const handleClick = (e: MouseEvent<HTMLButtonElement> | Event) => {
        e.preventDefault();
        dispatch(openCreateIssueModal(postId));
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
};

export default CreateIssuePostMenuAction;
