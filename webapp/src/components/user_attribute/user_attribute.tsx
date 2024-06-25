import React, {PureComponent} from 'react';

import type {UserAttributeProps} from '.';

export default class UserAttribute extends PureComponent<UserAttributeProps> {
    componentDidMount() {
        this.props.actions.getGitlabUser(this.props.id);
    }

    render() {
        const username = this.props.username;
        const baseURL = this.props.gitlabURL;

        if (!username || !baseURL) {
            return null;
        }

        return (
            <div style={style.container}>
                <a
                    href={baseURL + '/' + username}
                    target='_blank'
                    rel='noopener noreferrer'
                >
                    <i className='fa fa-gitlab'/>{' ' + username}
                </a>
            </div>
        );
    }
}

const style = {
    container: {
        margin: '5px 0',
    },
};
