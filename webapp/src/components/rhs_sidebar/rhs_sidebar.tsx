import React from 'react';

import {isDesktopApp} from 'src/utils/user_agent';
import {connectUsingBrowserMessage} from 'src/constants';

import MattermostGitLabSVG from './mattermost_gitlab';
import NoSubscriptionsSVG from './no_subscriptions';

import './rhs_sidebar.css';

const NotSignedIn = (props: NotSignedInProps) => {
    const openConnectWindow = (e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => {
        e.preventDefault();
        if (isDesktopApp()) {
            props.sendEphemeralPost(connectUsingBrowserMessage);
            return;
        }
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

interface NotSignedInProps {
    pluginServerRoute: string;
    sendEphemeralPost: (message: string) => void;
}

const UserHeader = (props: UserHeaderProps) => (
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

interface UserHeaderProps {
    currentUserId: string;
    username: string;
    gitlabURL: string;
}

const Subscription = (props: SubscriptionProps) => {
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

interface SubscriptionProps {
    url: string;
    name: string;
    features: string[];
}

const Subscriptions = (props: SubscriptionsProps) => {
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

interface SubscriptionsProps {
    currentChannelId?: string;
    subscriptions: Subscription[];
}

interface Subscription {
    repository_name: string;
    repository_url: string;
    features: string[];
}

interface RhsSidebarProps {
    currentUserId: string,
    connected: boolean,
    username?: string,
    gitlabURL?: string,
    currentChannelId?: string,
    currentChannelSubscriptions: Partial<Subscription>[],
    pluginServerRoute: string,
    actions: {
        getChannelSubscriptions: (channel: string) => Promise<any>,
        sendEphemeralPost: (message: string) => void;
    }
}

interface RhsSidebarState {
    refreshing: boolean;
}

export default class RHSSidebar extends React.PureComponent<RhsSidebarProps, RhsSidebarState> {
    constructor(props: RhsSidebarProps) {
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

    componentDidUpdate(prevProps: RhsSidebarProps) {
        if ((this.props.connected && !prevProps.connected) || (this.props.currentChannelId !== prevProps.currentChannelId)) {
            this.getData();
        }
    }

    getData = async (e?: React.MouseEvent<HTMLAnchorElement, MouseEvent>): Promise<void> => {
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
                    sendEphemeralPost={this.props.actions.sendEphemeralPost}
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
