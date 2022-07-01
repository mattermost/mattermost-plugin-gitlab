// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {MouseEvent, useCallback} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';

import GitLabIcon from 'src/images/icons/gitlab';
import {id as pluginId} from 'src/manifest';
import {openCreateIssueModal} from 'src/actions';
import {GlobalState} from 'src/types/global_state';

type PropTypes = {
    postId: string;
}

const CreateIssuePostMenuAction = ({postId}: PropTypes) => {
    const {show} = useSelector((state: GlobalState) => {
        const post = getPost(state, postId);
        const systemMessage = Boolean(!post || isSystemMessage(post));
    
        return {
            show: state[`plugins-${pluginId}` as plugin].connected && !systemMessage,
        };
    })

    const dispatch = useDispatch();

    const handleClick = useCallback((e: MouseEvent<HTMLButtonElement> | Event) => {        
        e.preventDefault();
        dispatch(openCreateIssueModal(postId));
    }, [postId]);

    if (!show) {
        return null;
    }

    const content = (
        <button
            className='style-none'
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
