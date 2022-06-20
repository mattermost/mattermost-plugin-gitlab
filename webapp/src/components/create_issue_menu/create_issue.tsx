// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {MouseEvent, PureComponent} from 'react';
import GitLabIcon from '../../images/icons/gitlab';

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

export default class CreateIssuePostMenuAction extends PureComponent<PropTypes> {
    handleClick = (e: MouseEvent<HTMLButtonElement> | Event) => {        
        const {postId} = this.props;
        e.preventDefault();
        this.props.actions.open(postId);
    };

    render() {
        if (!this.props.show) {
            return null;
        }

        const content = (
            <button
                className='style--none'
                role='presentation'
                onClick={this.handleClick}
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
}
