// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent, MouseEvent} from 'react';
import PropTypes from 'prop-types';

import GitLabIcon from 'src/images/icons/gitlab';

interface PropTypes {
    postId: string;
    show: boolean;
    actions: {
        open: (postId: any) => {
            type: string;
            data: {
                postId: any;
            };
        };
    };
};

export default class AttachCommentToIssuePostMenuAction extends PureComponent<PropTypes> {
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
                {'Attach to GitLab Issue'}
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
