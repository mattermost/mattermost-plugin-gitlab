// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {MouseEvent} from 'react';
import { useDispatch, useSelector } from 'react-redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {isSystemMessage} from 'mattermost-redux/utils/post_utils';
import { GlobalState } from 'mattermost-redux/types/store';

import GitLabIcon from 'src/images/icons/gitlab';
import {id as pluginId} from 'src/manifest';
import {openCreateIssueModal} from 'src/actions';

interface PropTypes {
    postId: string;
}

interface pluginMethods {
    connected: boolean
}

interface CurrentState extends GlobalState {
    plugin: pluginMethods;
}

const CreateIssuePostMenuAction = ({postId}: PropTypes) => {
    const {show} = useSelector((state: CurrentState) => {
        const post = getPost(state, postId);
        const systemMessage = Boolean(!post || isSystemMessage(post));
    
        return {
            show: state[`plugins-${pluginId}` as plugin].connected && !systemMessage,
        };
    })

    const dispatch = useDispatch();

    const handleClick = (e: MouseEvent<HTMLButtonElement> | Event) => {        
        e.preventDefault();
        dispatch(openCreateIssueModal(postId))
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
