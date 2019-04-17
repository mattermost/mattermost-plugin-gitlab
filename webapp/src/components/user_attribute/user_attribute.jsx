import React from 'react';
import PropTypes from 'prop-types';

export default class UserAttribute extends React.PureComponent {
    static propTypes = {
        id: PropTypes.string.isRequired,
        username: PropTypes.string,
        enterpriseURL: PropTypes.string,
        actions: PropTypes.shape({
            getGitlabUser: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);
        props.actions.getGitlabUser(props.id);
    }

    render() {
        const username = this.props.username;
        let baseURL = 'https://gitlab.com';
        if (this.props.enterpriseURL) {
            baseURL = this.props.enterpriseURL;
        }

        if (!username) {
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
