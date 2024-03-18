import React, {PureComponent, ReactElement} from 'react';
import {Tooltip, OverlayTrigger} from 'react-bootstrap';

import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {Placement} from 'react-bootstrap/esm/Overlay';

import {Theme} from 'mattermost-redux/types/preferences';

import {RHSStates, connectUsingBrowserMessage} from '../../constants';
import {isDesktopApp} from '../../utils/user_agent';

import {GitLabIssuesIcon, GitLabMergeRequestIcon, GitLabReviewsIcon, GitLabTodosIcon} from '../../utils/icons';

interface SidebarButtonsProps {
    theme: Theme;
    connected: boolean;
    username: string;
    org: string;
    clientId: string;
    gitlabURL: string;
    reviews?: unknown[];
    todos?: unknown[];
    yourAssignedPrs?: unknown[];
    yourAssignedIssues?: unknown[];
    isTeamSidebar: boolean;
    pluginServerRoute: string;
    showRHSPlugin: () => void;
    actions: {
        updateRHSState: (rhsState: string) => void;
        sendEphemeralPost: (message: string) => void;
        getLHSData: () => Promise<void>;
    };
}

interface SidebarButtonsState {
    refreshing: boolean;
}

export default class SidebarButtons extends PureComponent<SidebarButtonsProps, SidebarButtonsState> {
    constructor(props: SidebarButtonsProps) {
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

    componentDidUpdate(prevProps: SidebarButtonsProps) {
        if (this.props.connected && !prevProps.connected) {
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
        await this.props.actions.getLHSData();
        this.setState({refreshing: false});
    };

    openConnectWindow = (e: React.MouseEvent<HTMLAnchorElement, MouseEvent>): void => {
        e.preventDefault();
        if (isDesktopApp()) {
            this.props.actions.sendEphemeralPost(connectUsingBrowserMessage);
            return;
        }
        window.open(`${this.props.pluginServerRoute}/oauth/connect`, 'Connect Mattermost to GitLab', 'height=570,width=520');
    };

    openRHS = (rhsState: string): void => {
        this.props.actions.updateRHSState(rhsState);
        this.props.showRHSPlugin();
    };

    render(): ReactElement {
        const style = getStyle(this.props.theme);
        const isTeamSidebar = this.props.isTeamSidebar;

        let container = style.containerHeader;
        let button = style.buttonHeader;
        let placement: Placement = 'bottom';
        if (isTeamSidebar) {
            placement = 'right';
            button = style.buttonTeam;
            container = style.containerTeam;
        }

        if (!this.props.connected) {
            if (isTeamSidebar) {
                return (
                    <OverlayTrigger
                        key='gitlabConnectLink'
                        placement={placement}
                        overlay={<Tooltip id='reviewTooltip'>{'Connect to your GitLab instance'}</Tooltip>}
                    >
                        <a
                            href={`${this.props.pluginServerRoute}/oauth/connect`}
                            onClick={this.openConnectWindow}
                            style={button}
                        >
                            <i className='fa fa-gitlab fa-2x'/>
                        </a>
                    </OverlayTrigger>
                );
            }
        }

        const baseURL = this.props.gitlabURL || 'https://gitlab.com';
        const reviews = this.props.reviews || [];
        const yourAssignedPrs = this.props.yourAssignedPrs || [];
        const todos = this.props.todos || [];
        const yourAssignedIssues = this.props.yourAssignedIssues || [];
        const refreshClass = this.state.refreshing ? ' fa-spin' : '';

        return (
            <div style={container}>
                <a
                    key='gitlabHeader'
                    href={baseURL}
                    target='_blank'
                    rel='noopener noreferrer'
                    style={button}
                >
                    <i className='fa fa-gitlab fa-lg'/>
                </a>
                <OverlayTrigger
                    key='gitlabYourAssignedPrsLink'
                    placement={placement}
                    overlay={<Tooltip id='yourAssignedPrsTooltip'>{'Merge requests assigned'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.PRS)}
                        style={button}
                    >
                        <GitLabMergeRequestIcon fill={changeOpacity(this.props.theme.sidebarText, 0.6)}/>
                        <span style={style.buttonCount}>{yourAssignedPrs.length}</span>
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='gitlabReviewsLink'
                    placement={placement}
                    overlay={<Tooltip id='reviewTooltip'>{'Merge requests needing review'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.REVIEWS)}
                        style={button}
                    >
                        <GitLabReviewsIcon fill={changeOpacity(this.props.theme.sidebarText, 0.6)}/>
                        <span style={style.buttonCount}>{reviews.length}</span>
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='gitlabIssuesLink'
                    placement={placement}
                    overlay={<Tooltip id='issuesTooltip'>{'Issues'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.ISSUES)}
                        style={button}
                    >
                        <GitLabIssuesIcon fill={changeOpacity(this.props.theme.sidebarText, 0.6)}/>
                        <span style={style.buttonCount}>{yourAssignedIssues.length}</span>
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='gitlabTodosLink'
                    placement={placement}
                    overlay={<Tooltip id='todosTooltip'>{'To-Do list'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.TODOS)}
                        style={button}
                    >
                        <GitLabTodosIcon fill={changeOpacity(this.props.theme.sidebarText, 0.6)}/>
                        <span style={style.buttonCount}>{todos.length}</span>
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='gitlabRefreshButton'
                    placement={placement}
                    overlay={<Tooltip id='refreshTooltip'>{'Refresh'}</Tooltip>}
                >
                    <a
                        href='#'
                        style={button}
                        onClick={this.getData}
                    >
                        <i className={'fa fa-refresh' + refreshClass}/>
                    </a>
                </OverlayTrigger>
            </div>
        );
    }
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        buttonTeam: {
            color: changeOpacity(theme.sidebarText, 0.6),
            display: 'block',
            marginBottom: '10px',
            width: '100%',
        },
        buttonHeader: {
            color: changeOpacity(theme.sidebarText, 0.6),
            textAlign: 'center',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
        },
        buttonCount: {
            marginLeft: '2px',
        },
        containerHeader: {
            marginTop: '10px',
            marginBottom: '5px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-around',
            padding: '0 10px',
        },
    };
});
