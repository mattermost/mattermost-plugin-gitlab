import React, {useState} from 'react';
import PropTypes from 'prop-types';

import MattermostGitLab from './mattermost_gitlab';
import NoSubscriptions from './no_subscriptions';

const notSignedInStyle = {
    margin: '24px',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    textAlign: 'center',
};

const welcomeStyle = {
    fontFamily: 'Metropolis',
    fontSize: '16px',
    fontWeight: '600',
    lineHeight: '24px',
    color: 'var(--center-channel-color)',
};

const hrStyle = {
    width: '244px',
    borderTop: '1px solid rgba(var(--center-channel-color-rgb), 0.08)',
    margin: '24px 0 0 0',
};

const mattermostGitLabStyle = {
    width: '217px',
    height: '72px',
    marginTop: '24px',
};

const connectPromptStyle = {
    fontFamily: 'Metropolis',
    fontSize: '22px',
    fontWeight: '600',
    lineHeight: '28px',
    marginTop: '16px',
    color: 'var(--center-channel-color)',
};

const connectStyle = {
    backgroundColor: 'var(--button-bg)',
    color: 'var(--button-color)',
    fontFamily: 'Open Sans',
    fontWeight: '600',
    padding: '12px 16px 12px 16px',
    borderRadius: '4px',
    marginTop: '24px',
};

const NotSignedIn = (props) => {
    const openConnectWindow = (e) => {
        e.preventDefault();
        window.open(`${props.pluginServerRoute}/oauth/connect`, 'Connect Mattermost to GitLab', 'height=570,width=520');
    };

    return (
        <div style={notSignedInStyle}>
            <div style={welcomeStyle}>{'Welcome to the Mattermost GitLab plugin'}</div>
            <hr style={hrStyle}/>
            <div style={mattermostGitLabStyle}>
                <MattermostGitLab/>
            </div>
            <div style={connectPromptStyle}>{'Connect your account'}<br/>{'to get started'}</div>
            <a
                style={connectStyle}
                href={`${props.pluginServerRoute}/oauth/connect`}
                onClick={openConnectWindow}
            >
                {'Connect account'}
            </a>
        </div>
    );
};

NotSignedIn.propTypes = {
    pluginServerRoute: PropTypes.string,
};

const userHeaderStyle = {
    display: 'flex',
    flexDirection: 'row',
    justifyContent: 'flex-start',
    alignItems: 'center',
    backgroundColor: 'rgba(var(--center-channel-color-rgb), 0.04)',
    padding: '12px 20px',
};

const userProfileStyle = {
    margin: '0 8px 0 0',
};

const userDetailsStyle = {
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'center',
    alignItems: 'left',
    fontFamily: 'Open Sans',
    lineHeight: '16px',
    textAlign: 'left',
    color: 'var(--center-channel-color)',
};

const descriptionStyle = {
    fontSize: '12px',
    fontWeight: '400',
    letterSpacing: 0,
};

const usernameStyle = {
    fontSize: '11px',
    fontWeight: '600',
    letterSpacing: '0.02em',
};

const gitlabURLStyle = {
    display: 'flex',
    marginLeft: 'auto',
    padding: '10px 16px',
    gap: '10px',
    border: '1px solid var(--button-bg)',
    borderRadius: '4px',
    fontFamily: 'Open Sans',
    fontWeight: '600',
    fontSize: '12px',
    lineHeight: '10px',
    color: 'var(--button-bg)',
};

const UserHeader = (props) => (
    <div style={userHeaderStyle}>
        <img
            className='Avatar Avatar-lg'
            style={userProfileStyle}
            alt='user profile image'
            src={`/api/v4/users/${props.currentUserId}/image`}
        />
        <div style={userDetailsStyle}>
            <div style={descriptionStyle}>{'Signed in as'}</div>
            <div style={usernameStyle}>{props.username}</div>
        </div>
        <a
            href={props.gitlabURL}
            style={gitlabURLStyle}
            target='_new'
        >{'GitLab'}</a>
    </div>
);

UserHeader.propTypes = {
    currentUserId: PropTypes.string,
    username: PropTypes.string,
    gitlabURL: PropTypes.string,
};

const subscriptionStyle = {
    backgroundColor: 'var(--center-channel-bg)',
    background: 'linear-gradient(0deg, rgba(var(--center-channel-color-rgb, 0.04), rgba(var(--center-channel-color-rgb, 0.04)), linear-gradient(0deg, var(--center-channel-bg), var(--center-channel-bg))',
    padding: '14px 16px 14px 16px',
    border: '1px solid rgba(var(--center-channel-color-rgb), 0.04)',
    boxShadow: '0px 2px 3px 0px rgba(0, 0, 0, 0.08)',
    transition: 'box-shadow 0.3s ease-in-out',
    borderRadius: '4px',
    marginTop: '12px',
};

const hoveredSubscriptionStyle = {
    ...subscriptionStyle,
    boxShadow: '0px 4px 6px 0px rgba(0, 0, 0, 0.12)',
};

const subscriptionHeaderStyle = {
    fontFamily: 'Open Sans',
    fontSize: '12px',
    fontWeight: '600',
    lineHeight: '16px',
    letterSpacing: '0.02em',
    textTransform: 'uppercase',
    margin: '0 0 4px 0',
    color: 'rgba(var(--center-channel-color-rgb), 0.72)',
};

const subscriptionDetailsStyle = {
    fontFamily: 'Open Sans',
    fontSize: '14px',
    fontWeight: '400',
    lineHeight: '20px',
    color: 'var(--button-bg)',
};

const Subscription = (props) => {
    const [hovering, setHovering] = useState(false);

    let activeSubscriptionStyle = subscriptionStyle;
    if (hovering) {
        activeSubscriptionStyle = hoveredSubscriptionStyle;
    }

    return (
        <div
            style={activeSubscriptionStyle}
            onMouseEnter={() => setHovering(true)}
            onMouseLeave={() => setHovering(false)}
        >
            <h2 style={subscriptionHeaderStyle}>{'Repository'}</h2>
            <a
                style={subscriptionDetailsStyle}
                href={props.url}
                target='_new'
            >{props.name}</a>
        </div>
    );
};

Subscription.propTypes = {
    url: PropTypes.string,
    name: PropTypes.string,
};

const containerStyle = {
    margin: '24px',
};

const noSubscriptionsContainerStyle = {
    ...containerStyle,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
};

const noSubscriptionsImgStyle = {
    width: '264px',
    margin: '10px 0 30px 0',
};

const noSubscriptionsStyle = {
    fontFamily: 'Metropolis',
    fontSize: '18px',
    fontWeight: '600',
    lineHeight: '24px',
    textAlign: 'center',
    color: 'var(--center-channel-color)',
};

const useGitLabSlashCommandStyle = {
    fontFamily: 'Open Sans',
    fontSize: '14px',
    fontWeight: '400',
    lineHeight: '20px',
    textAlign: 'center',
    color: 'var(--center-channel-color)',
    margin: '8px 0 0 0',
};

const headerStyle = {
    fontFamily: 'Metropolis',
    fontSize: '16px',
    fontWeight: '400',
    lineHeight: '24px',
    color: 'var(--center-channel-color)',
};

const Subscriptions = (props) => {
    if (props.subscriptions.length === 0) {
        return (
            <div style={noSubscriptionsContainerStyle}>
                <div style={noSubscriptionsImgStyle}>
                    <NoSubscriptions/>
                </div>
                <div style={noSubscriptionsStyle}>{'There are no GitLab subscriptions available in this channel.'}</div>
                <div style={useGitLabSlashCommandStyle}>{'Use the /gitlab slash command to create a subscription.'}</div>
            </div>
        );
    }

    return (
        <div style={containerStyle}>
            <div style={headerStyle}>{'GitLab Subscriptions'}</div>
            {props.subscriptions.map((subscription) => (
                <Subscription
                    key={subscription.repository_name}
                    name={subscription.repository_name}
                    url={subscription.repository_url}
                />
            ))}
        </div>
    );
};

Subscriptions.propTypes = {
    subscriptions: PropTypes.arrayOf(PropTypes.shape({
        repository_name: PropTypes.string,
        repository_url: PropTypes.string,
    })),
};

export default class RHSSidebar extends React.PureComponent {
    static propTypes = {
        currentUserId: PropTypes.string,
        connected: PropTypes.bool,
        username: PropTypes.string,
        gitlabURL: PropTypes.string,
        currentChannelId: PropTypes.string,
        currentChannelSubscriptions: PropTypes.arrayOf(PropTypes.shape({
            repository_name: PropTypes.string,
            repository_url: PropTypes.string,
        })),
        pluginServerRoute: PropTypes.string,
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
        await Promise.all([
            this.props.actions.getChannelSubscriptions(this.props.currentChannelId),
        ]);
        this.setState({refreshing: false});
    }

    render() {
        if (!this.props.connected) {
            return <NotSignedIn pluginServerRoute={this.props.pluginServerRoute}/>;
        }

        return (
            <div>
                <UserHeader
                    currentUserId={this.props.currentUserId}
                    gitlabURL={this.props.gitlabURL}
                    username={this.props.username}
                />
                <Subscriptions subscriptions={this.props.currentChannelSubscriptions}/>
            </div>
        );
    }
}
