// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {MouseEvent} from 'react';
import GitLabIcon from '../../images/icons/gitlab';

interface PropTypes {
    show: boolean;
    actions: {
        open: (postId: any) => {
            type: string;
            data: {
                postId: any;
            };
        };
    };
    postId: string;
}

export default class CreateIssuePostMenuAction extends React.PureComponent<PropTypes> {
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
