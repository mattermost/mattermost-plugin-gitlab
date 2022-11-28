import React from 'react';
import PropTypes from 'prop-types';
import styled from 'styled-components';

import MattermostGitLabSVG from './mattermost_gitlab';
import NoSubscriptionsSVG from './no_subscriptions';

const NotSignedInDiv = styled.div`
    margin: 24px;
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
`;

const Welcome = styled.div`
    font-family: 'Metropolis';
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
    color: var(--center-channel-color);
`;

const Hr = styled.hr`
    width: 244px;
    border-top: 1px solid rgba(var(--center-channel-color-rgb), 0.08);
    margin: 24px 0 0 0;
`;

const MattermostGitLab = styled.div`
    width: 217px;
    height: 72px;
    margin-top: 24px;
`;

const ConnectPrompt = styled.div`
    font-family: 'Metropolis';
    font-size: 22px;
    font-weight: 600;
    line-height: 28px;
    margin-top: 16px;
    color: var(--center-channel-color);
`;

const Connect = styled.a`
    && {
        background-color: var(--button-bg);
        color: var(--button-color);
        font-family: 'Open Sans';
        font-weight: 600;
        padding: 12px 16px 12px 16px;
        border-radius: 4px;
        margin-top: 24px;
        text-decoration: none;

        &:active, &:visited, &:hover {
            color: var(--button-color);
            text-decoration: none;
        }
    }
`;

const NotSignedIn = (props) => {
    const openConnectWindow = (e) => {
        e.preventDefault();
        window.open(`${props.pluginServerRoute}/oauth/connect`, 'Connect Mattermost to GitLab', 'height=570,width=520');
    };

    return (
        <NotSignedInDiv>
            <Welcome>{'Welcome to the Mattermost GitLab plugin'}</Welcome>
            <Hr/>
            <MattermostGitLab>
                <MattermostGitLabSVG/>
            </MattermostGitLab>
            <ConnectPrompt>{'Connect your account'}<br/>{'to get started'}</ConnectPrompt>
            <Connect
                href={`${props.pluginServerRoute}/oauth/connect`}
                onClick={openConnectWindow}
            >
                {'Connect account'}
            </Connect>
        </NotSignedInDiv>
    );
};

NotSignedIn.propTypes = {
    pluginServerRoute: PropTypes.string.isRequired,
};

const UserHeaderContainer = styled.div`
    display: flex;
    flex-direction: row;
    justify-content: flex-start;
    align-items: center;
    background-color: rgba(var(--center-channel-color-rgb), 0.04);
    padding: 12px 20px;
`;

const UserProfile = styled.img`
    margin: 0 8px 0 0;
`;

const UserDetails = styled.div`
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: left;
    font-family: 'Open Sans';
    line-height: 16px;
    text-align: left;
    color: var(--center-channel-color);
`;

const Description = styled.div`
    font-size: 12px;
    font-weight: 400;
    letterSpacing: 0;
`;

const Username = styled.div`
    font-size: 11px;
    font-weight: 600;
    letterSpacing: 0.02em;
`;

const GitLabURL = styled.a`
    display: flex;
    margin-left: auto;
    padding: 10px 16px;
    gap: 10px;
    border: 1px solid var(--button-bg);
    border-radius: 4px;
    font-family: 'Open Sans';
    font-weight: 600;
    font-size: 12px;
    line-height: 10px;
    color: var(--button-bg);
`;

const UserHeader = (props) => (
    <UserHeaderContainer>
        <UserProfile
            className='Avatar Avatar-lg'
            alt='user profile image'
            src={`/api/v4/users/${props.currentUserId}/image`}
        />
        <UserDetails>
            <Description>{'Signed in as'}</Description>
            <Username>{props.username}</Username>
        </UserDetails>
        <GitLabURL
            href={props.gitlabURL}
            target='_new'
        >{'GitLab'}</GitLabURL>
    </UserHeaderContainer>
);

UserHeader.propTypes = {
    currentUserId: PropTypes.string.isRequired,
    username: PropTypes.string.isRequired,
    gitlabURL: PropTypes.string.isRequired,
};

const Feature = styled.span`
    font-family: 'Open Sans';
    font-weight: 600;
    font-size: 12px;
    background-color: rgba(var(--center-channel-color-rgb), 0.08);
    color: var(--center-channel-color);
    padding: 2px 5px;
    border-radius: 4px;
    line-height: 16px;
`;

const SubscriptionContainer = styled.div`
    background-color: var(--center-channel-bg);
    padding: 14px 16px 14px 16px;
    border: 1px solid rgba(var(--center-channel-color-rgb), 0.04);
    box-shadow: 0px 2px 3px 0px rgba(0, 0, 0, 0.08);
    transition: box-shadow 0.3s ease-in-out;
    border-radius: 4px;
    margin-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 16px;

    &:hover {
        box-shadow: 0px 4px 6px 0px rgba(0, 0, 0, 0.12);
    }
`;

const SubscriptionHeader = styled.h2`
    font-family: 'Open Sans';
    font-size: 12px;
    font-weight: 600;
    line-height: 16px;
    letterSpacing: 0.02em;
    textTransform: uppercase;
    margin: 0 0 4px 0;
    color: rgba(var(--center-channel-color-rgb), 0.72);
`;

const SubscriptionDetails = styled.a`
    font-family: 'Open Sans';
    font-size: 14px;
    font-weight: 400;
    line-height: 20px;
    color: var(--button-bg);
`;

const Features = styled.div`
    display: flex;
    flex-wrap: wrap;
    flex-direction: row;
    justify-content: flex-start;
    align-content: flex-start;
    gap: 4px;
`;

const Subscription = (props) => {
    return (
        <SubscriptionContainer>
            <div>
                <SubscriptionHeader>{'Repository'}</SubscriptionHeader>
                <SubscriptionDetails
                    href={props.url}
                    target='_new'
                >{props.name}</SubscriptionDetails>
            </div>
            <div>
                <SubscriptionHeader>{'Features'}</SubscriptionHeader>
                <Features>
                    {props.features.map((feature) => (
                        <Feature key={feature}>{feature}</Feature>
                    ))}
                </Features>
            </div>
        </SubscriptionContainer>
    );
};

Subscription.propTypes = {
    url: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    features: PropTypes.arrayOf(PropTypes.string).isRequired,
};

const Container = styled.div`
    margin: 24px;
`;

const NoSubscriptionsContainer = styled(Container)`
    display: flex;
    flex-direction: column;
    align-items: center;
`;

const NoSubscriptionsImg = styled.div`
    width: 264px;
    margin: 10px 0 30px 0;
`;

const NoSubscriptions = styled.div`
    font-family: 'Metropolis';
    font-size: 18px;
    font-weight: 600;
    line-height: 24px;
    text-align: center;
    color: var(--center-channel-color);
`;

const UseGitLabSlashCommand = styled.div`
    font-family: 'Open Sans';
    font-size: 14px;
    font-weight: 400;
    line-height: 20px;
    text-align: center;
    color: var(--center-channel-color);
    margin: 8px 0 0 0;
`;

const Header = styled.div`
    font-family: 'Metropolis';
    font-size: 16px;
    font-weight: 400;
    line-height: 24px;
    color: var(--center-channel-color);
`;

const Subscriptions = (props) => {
    if (props.subscriptions.length === 0) {
        return (
            <NoSubscriptionsContainer>
                <NoSubscriptionsImg>
                    <NoSubscriptionsSVG/>
                </NoSubscriptionsImg>
                <NoSubscriptions>{'There are no GitLab subscriptions available in this channel.'}</NoSubscriptions>
                <UseGitLabSlashCommand>{'Use the /gitlab slash command to create a subscription.'}</UseGitLabSlashCommand>
            </NoSubscriptionsContainer>
        );
    }

    return (
        <Container>
            <Header>{'GitLab Subscriptions'}</Header>
            {props.subscriptions.map((subscription) => (
                <Subscription
                    key={subscription.repository_url + props.currentChannelId}
                    name={subscription.repository_name}
                    url={subscription.repository_url}
                    features={subscription.features}
                />
            ))}
        </Container>
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
            return <NotSignedIn pluginServerRoute={this.props.pluginServerRoute}/>;
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
