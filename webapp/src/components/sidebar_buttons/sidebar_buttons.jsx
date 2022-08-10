import React from 'react';
import {Tooltip, OverlayTrigger} from 'react-bootstrap';
import PropTypes from 'prop-types';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {RHSStates} from 'src/constants';

export default class SidebarButtons extends React.PureComponent {
    static propTypes = {
        theme: PropTypes.object.isRequired,
        connected: PropTypes.bool,
        username: PropTypes.string,
        org: PropTypes.string,
        clientId: PropTypes.string,
        gitlabURL: PropTypes.string,
        reviews: PropTypes.arrayOf(PropTypes.object),
        unreads: PropTypes.arrayOf(PropTypes.object),
        yourPrs: PropTypes.arrayOf(PropTypes.object),
        yourAssignments: PropTypes.arrayOf(PropTypes.object),
        isTeamSidebar: PropTypes.bool,
        pluginServerRoute: PropTypes.string.isRequired,
        showRHSPlugin: PropTypes.func.isRequired,
        actions: PropTypes.shape({
            getReviews: PropTypes.func.isRequired,
            getUnreads: PropTypes.func.isRequired,
            getYourPrs: PropTypes.func.isRequired,
            getYourAssignments: PropTypes.func.isRequired,
            updateRHSState: PropTypes.func.isRequired,
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
        if (this.props.connected && !prevProps.connected) {
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
            this.props.actions.getReviews(),
            this.props.actions.getUnreads(),
            this.props.actions.getYourPrs(),
            this.props.actions.getYourAssignments(),
        ]);
        this.setState({refreshing: false});
    };

    openConnectWindow = (e) => {
        e.preventDefault();
        window.open(`${this.props.pluginServerRoute}/oauth/connect`, 'Connect Mattermost to GitLab', 'height=570,width=520');
    };

    openRHS = (rhsState) => {
        this.props.actions.updateRHSState(rhsState);
        this.props.showRHSPlugin();
    };

    render() {
        const style = getStyle(this.props.theme);
        const isTeamSidebar = this.props.isTeamSidebar;

        let container = style.containerHeader;
        let button = style.buttonHeader;
        let placement = 'bottom';
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
                        overlay={<Tooltip id='reviewTooltip'>{'Connect to your GitLab'}</Tooltip>}
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
            return null;
        }

        const baseURL = this.props.gitlabURL || 'https://gitlab.com';
        const reviews = this.props.reviews || [];
        const yourPrs = this.props.yourPrs || [];
        const unreads = this.props.unreads || [];
        const yourAssignments = this.props.yourAssignments || [];
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
                    key='gitlabYourPrsLink'
                    placement={placement}
                    overlay={<Tooltip id='yourPrsTooltip'>{'Your open merge requests'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.PRS)}
                        style={button}
                    >
                        <i className='fa fa-compress'/>
                        {' ' + yourPrs.length}
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
                        <i className='fa fa-code-fork'/>
                        {' ' + reviews.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='gitlabAssignmentsLink'
                    placement={placement}
                    overlay={<Tooltip id='reviewTooltip'>{'Your assignments'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.ASSIGNMENTS)}
                        style={button}
                    >
                        <i className='fa fa-list-ol'/>
                        {' ' + yourAssignments.length}
                    </a>
                </OverlayTrigger>
                <OverlayTrigger
                    key='gitlabUnreadsLink'
                    placement={placement}
                    overlay={<Tooltip id='unreadsTooltip'>{'Unread messages'}</Tooltip>}
                >
                    <a
                        onClick={() => this.openRHS(RHSStates.UNREADS)}
                        style={button}
                    >
                        <i className='fa fa-envelope'/>
                        {' ' + unreads.length}
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
