import React, {useEffect, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';

import {logError} from 'mattermost-redux/actions/errors';

import {GitLabIssueOpenIcon, GitLabMergeRequestIcon, GitLabMergeRequestClosedIcon, GitLabMergedIcon, GitLabIssueClosedIcon} from '../../utils/icons';

import Client from '../../client';
import {getTruncatedText, validateGitlabUrl, isValidUrl, getInfoAboutLink} from '../../utils/tooltip_utils';
import {TooltipData} from 'src/types/gitlab_items';
import {getConnected, getConnectedGitlabUrl} from 'src/selectors';

import './tooltip.css';

const STATE_COLOR_MAP = {
    OPENED_COLOR: '#28a745',
    CLOSED_COLOR: '#cb2431',
    MERGED_COLOR: '#6f42c1',
    ISSUE_CLOSED_COLOR: '#0b5cad',
};

const STATE_TYPES = {
    OPENED: 'opened',
    CLOSED: 'closed',
    MERGED: 'merged',
};

const LINK_TYPES = {
    MERGE_REQUESTS: 'merge_requests',
    ISSUES: 'issues',
};

const TOOLTIP_ICON_DIMENSION = 16;
const TOOLTIP_MAX_TITLE_LENGTH = 70;

type Props = {
    href: string;
    show: boolean;
}

const LinkTooltip = ({href, show}: Props) => {
    const [data, setData] = useState<TooltipData | null>(null);

    const connected = useSelector(getConnected);
    const connectedGitlabUrl = useSelector(getConnectedGitlabUrl);

    const dispatch = useDispatch();
    useEffect(() => {
        if (!connected || !show) {
            return;
        }

        if (!isValidUrl(href)) {
            return;
        }

        const url = new URL(href);
        const gitlabUrl = new URL(connectedGitlabUrl);
        const init = async () => {
            if (url.host === gitlabUrl.host && validateGitlabUrl(href)) {
                const linkInfo = getInfoAboutLink(href, url.hostname);
                let res;
                switch (linkInfo?.type) {
                case LINK_TYPES.ISSUES:
                    res = await Client.getIssue(linkInfo.owner, linkInfo.repo, linkInfo.number);
                    break;
                case LINK_TYPES.MERGE_REQUESTS:
                    res = await Client.getPullRequest(linkInfo.owner, linkInfo.repo, linkInfo.number);
                    break;
                default:
                    dispatch(logError({message: `link type ${linkInfo.type} is not supported to display a tooltip`}));
                    return;
                }

                if (res) {
                    res = {...res, owner: linkInfo.owner, repo: linkInfo.repo, type: linkInfo.type};
                    setData(res);
                }
            }
        };

        init();
    }, [connected, href, show]);

    if (!data || !show) {
        return null;
    }

    const getIconElement = () => {
        let color;
        let icon;
        const {OPENED_COLOR, CLOSED_COLOR, MERGED_COLOR, ISSUE_CLOSED_COLOR} = STATE_COLOR_MAP;
        switch (data.type) {
        case LINK_TYPES.MERGE_REQUESTS:
            icon = (
                <GitLabMergeRequestIcon
                    fill={OPENED_COLOR}
                    width={TOOLTIP_ICON_DIMENSION}
                    height={TOOLTIP_ICON_DIMENSION}
                />
            );
            if (data.state === STATE_TYPES.CLOSED) {
                icon = <GitLabMergeRequestClosedIcon fill={CLOSED_COLOR}/>;
            } else if (data.state === STATE_TYPES.MERGED) {
                icon = <GitLabMergedIcon fill={MERGED_COLOR}/>;
            }
            break;
        case LINK_TYPES.ISSUES:
            color = data.state === STATE_TYPES.OPENED ? OPENED_COLOR : CLOSED_COLOR;
            icon = data.state === STATE_TYPES.OPENED ? <GitLabIssueOpenIcon fill={OPENED_COLOR}/> : <GitLabIssueClosedIcon fill={ISSUE_CLOSED_COLOR}/>;
            break;
        default:
            dispatch(logError({message: `link type ${data.type} is not supported to display a tooltip`}));
        }

        return (
            <span style={{color}}>
                {icon}
            </span>
        );
    };

    const date = new Date(data.created_at).toDateString();
    const {formatText, messageHtmlToComponent} = window.PostUtils;

    return (
        <div className='gitlab-tooltip'>
            <div className='gitlab-tooltip box gitlab-tooltip--large gitlab-tooltip--bottom-left p-4'>
                <div className='header mb-1'>
                    <a
                        title={data.repo}
                        href={href}
                    >
                        {data.repo}
                    </a>
                    {' on '}
                    <span>{date}</span>
                </div>

                <div className='body d-flex mt-1'>
                    <span className='tooltip-icon'>
                        {getIconElement()}
                    </span>

                    <div className='tooltip-info mt-1'>
                        <a href={href}>
                            <h5 className='mr-1'>{getTruncatedText(data.title, TOOLTIP_MAX_TITLE_LENGTH)}</h5>
                            <span className='mr-number'>{`#${data.iid}`}</span>
                        </a>
                        <div className='markdown-text mt-1 mb-1'>
                            {messageHtmlToComponent(formatText(data.description), false)}
                        </div>

                        {data.type === LINK_TYPES.MERGE_REQUESTS && (
                            <div className='base-head mt-1 mr-3'>
                                <span
                                    title={data.target_branch}
                                    className='commit-ref'
                                >
                                    {data.target_branch}
                                </span>
                                <span className='mx-1'>{'←'}</span>
                                <span
                                    title={data.source_branch}
                                    className='commit-ref'
                                >
                                    {data.source_branch}
                                </span>
                            </div>
                        )}

                        {/* Labels */}
                        <div className='labels mt-3'>
                            {data.labels && data.labels_with_details?.length && data.labels_with_details.map((label) => {
                                return (
                                    <span
                                        key={label.name}
                                        className='label mr-1'
                                        title={label.description}
                                        style={{backgroundColor: label.color as string}}
                                    >
                                        <span style={{color: label.text_color as string}}>{label.name}</span>
                                    </span>
                                );
                            })}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default LinkTooltip;
