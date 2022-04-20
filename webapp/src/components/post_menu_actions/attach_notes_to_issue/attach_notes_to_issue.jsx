import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';


export default class AttachNotesToIssuePostMenuAction extends PureComponent {
    static propTypes = {
        isSystemMessage: PropTypes.bool.isRequired,
        open: PropTypes.func.isRequired,
        postId: PropTypes.string,
        connected: PropTypes.bool.isRequired,
    };

    static defaultTypes = {
        locale: 'en',
    };

    handleClick = (e) => {
        const {open, postId} = this.props;
        e.preventDefault();
        open(postId);
    };

    render() {
        if (this.props.isSystemMessage || !this.props.connected) {
            return null;
        }

        const content = (
            <button
                className='style--none'
                role='presentation'
                onClick={this.handleClick}
            >
                <i className='fa fa-gitlab fa-lg'/>&nbsp;
                {'Attach to Gitlab Issue'}
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