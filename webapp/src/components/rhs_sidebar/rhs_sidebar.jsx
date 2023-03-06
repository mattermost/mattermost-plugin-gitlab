import React from 'react';
import PropTypes from 'prop-types';

import MattermostGitLabSVG from './mattermost_gitlab';
import NoSubscriptionsSVG from './no_subscriptions';

import './rhs_sidebar.css';

const NotSignedIn = (props) => {
    const openConnectWindow = (e) => {
        e.preventDefault();
        window.open(`${props.pluginServerRoute}/oauth/connect`, 'Connect Mattermost to GitLab', 'height=570,width=520');
    };

    return (
        <div className='gitlab-rhs-NotSignedInDiv'>
            <div className='gitlab-rhs-Welcome'>{'Welcome to the Mattermost GitLab plugin'}</div>
            <hr className='gitlab-rhs-Hr'/>
            <div className='gitlab-rhs-MattermostGitLab'>
                <MattermostGitLabSVG/>
            </div>
            <div className='gitlab-rhs-ConnectPrompt'>{'Connect your account'}<br/>{'to get started'}</div>
            <a
                className='gitlab-rhs-Connect'
                href={`${props.pluginServerRoute}/oauth/connect`}
                onClick={openConnectWindow}
            >
                {'Connect account'}
            </a>
        </div>
    );
};

NotSignedIn.propTypes = {
    pluginServerRoute: PropTypes.string.isRequired,
};

const UserHeader = (props) => (
    <div className='gitlab-rhs-UserHeaderContainer'>
        <img
            className='gitlab-rhs-UserProfile Avatar Avatar-lg'
            alt='user profile image'
            src={`/api/v4/users/${props.currentUserId}/image`}
        />
        <div className='gitlab-rhs-UserDetails'>
            <div className='gitlab-rhs-Description'>{'Signed in as'}</div>
            <div className='gitlab-rhs-Username'>{props.username}</div>
        </div>
        <a
            className='gitlab-rhs-GitLabURL'
            href={props.gitlabURL}
            target='_new'
        >{'GitLab'}</a>
    </div>
);

UserHeader.propTypes = {
    currentUserId: PropTypes.string.isRequired,
    username: PropTypes.string.isRequired,
    gitlabURL: PropTypes.string.isRequired,
};

const Subscription = (props) => {
    return (
        <div className='gitlab-rhs-SubscriptionContainer'>
            <div>
                <h2 className='gitlab-rhs-SubscriptionHeader'>{'Repository'}</h2>
                <a
                    className='gitlab-rhs-SubscriptionDetails'
                    href={props.url}
                    target='_new'
                >{props.name}</a>
            </div>
            <div>
                <h2 className='gitlab-rhs-SubscriptionHeader'>{'Features'}</h2>
                <div className='gitlab-rhs-Features'>
                    {props.features.map((feature) => (
                        <span key={feature}>{feature}</span>
                    ))}
                </div>
            </div>
        </div>
    );
};

Subscription.propTypes = {
    url: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    features: PropTypes.arrayOf(PropTypes.string).isRequired,
};

const Subscriptions = (props) => {
    if (props.subscriptions.length === 0) {
        return (
            <div className='gitlab-rhs-NoSubscriptionsContainer'>
                <div className='gitlab-rhs-NoSubscriptionsImg'>
                    <NoSubscriptionsSVG/>
                </div>
                <div className='gitlab-rhs-NoSubscriptions'>{'There are no GitLab subscriptions available in this channel.'}</div>
                <div className='gitlab-rhs-UseGitLabSlashCommand'>{'Use the /gitlab slash command to create a subscription.'}</div>
            </div>
        );
    }

    return (
        <div className='gitlab-rhs-Container'>
            <div className='gitlab-rhs-Header'>{'GitLab Subscriptions'}</div>
            {props.subscriptions.map((subscription) => (
                <Subscription
                    key={subscription.repository_url + props.currentChannelId}
                    name={subscription.repository_name}
                    url={subscription.repository_url}
                    features={subscription.features}
                />
            ))}
        </div>
    );
};

Subscriptions.propTypes = {
    currentChannelId: PropTypes.string,
    subscriptions: PropTypes.arrayOf(PropTypes.shape({
        repository_name: PropTypes.string.isRequired,
        repository_url: PropTypes.string.isRequired,
    })).isRequired,
};

export default class RHSSidebar extends React.PureComponent {
    static propTypes = {
        currentUserId: PropTypes.string.isRequired,
        connected: PropTypes.bool.isRequired,
        username: PropTypes.string,
        gitlabURL: PropTypes.string,
        currentChannelId: PropTypes.string,
        currentChannelSubscriptions: PropTypes.arrayOf(PropTypes.shape({
            repository_name: PropTypes.string,
            repository_url: PropTypes.string,
            features: PropTypes.arrayOf(PropTypes.string),
        })).isRequired,
        pluginServerRoute: PropTypes.string.isRequired,
        actions: PropTypes.shape({
            getChannelSubscriptions: PropTypes.func.isRequired,
        }).isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            refreshing: false,
        };
    }

    componentDidMount() {
        if (this.props.connected) {
            this.getData();
        }
    }

    componentDidUpdate(prevProps) {
        if ((this.props.connected && !prevProps.connected) || (this.props.currentChannelId !== prevProps.currentChannelId)) {
            this.getData();
        }
    }

    getData = async (e) => {
        if (this.state.refreshing) {
            return;
        }

        if (e) {
            e.preventDefault();
        }

        this.setState({refreshing: true});
        await this.props.actions.getChannelSubscriptions(this.props.currentChannelId);
        this.setState({refreshing: false});
    }

    render() {
        if (!this.props.connected) {
            return (
                <NotSignedIn
                    pluginServerRoute={this.props.pluginServerRoute}
                />
            );
        }

        return (
            <div>
                <UserHeader
                    currentUserId={this.props.currentUserId}
                    gitlabURL={this.props.gitlabURL}
                    username={this.props.username}
                />
                <Subscriptions
                    currentChannelId={this.props.currentChannelId}
                    subscriptions={this.props.currentChannelSubscriptions}
                />
            </div>
        );
    }
}
