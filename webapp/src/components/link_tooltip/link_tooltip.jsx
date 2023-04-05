import React, {useEffect, useState} from 'react';
import PropTypes from 'prop-types';
import ReactMarkdown from 'react-markdown';
import Octicon, {GitMergeIcon, GitPullRequestIcon, IssueClosedIcon, IssueOpenedIcon} from '@primer/octicons-react';

import {useDispatch} from 'react-redux';

import {logError} from 'mattermost-redux/actions/errors';

import Client from '../../client';
import {validateGitlabURL} from '../../utils/regex_utils';
import {isValidUrl} from '../../utils/url_utils';

import './tooltip.css';

const STATE_COLOR_MAP = {
    OPENED_COLOR: '#28a745',
    CLOSED_COLOR: '#cb2431',
    MERGED_COLOR: '#6f42c1',
};

const STATE_TYPES = {
    OPENED: 'opened',
    CLOSED: 'closed',
};

const LINK_TYPES = {
    MERGE_REQUESTS: 'merge_requests',
    ISSUES: 'issues',
};

export const getInfoAboutLink = (href, hostname) => {
    const linkInfo = href.split(`${hostname}/`)[1].split('/');
    if (linkInfo.length >= 5) {
        return {
            owner: linkInfo[0],
            repo: linkInfo[1],
            type: linkInfo[3],
            number: linkInfo[4],
        };
    }
    return {};
};

export const LinkTooltip = ({href, connected, gitlabURL}) => {
    const [data, setData] = useState(null);
    const dispatch = useDispatch();
    useEffect(() => {
        if (!isValidUrl(href)) {
            return;
        }

        const url = new URL(href);
        const init = async () => {
            if (url.origin === gitlabURL && validateGitlabURL(href)) {
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
                    dispatch(logError(`link type ${data.type} is not supported to display a tooltip`));
                }

                if (res) {
                    res = {...res, owner: linkInfo.owner, repo: linkInfo.repo, type: linkInfo.type};
                    setData(res);
                }
            }
        };

        if (!connected) {
            return;
        }

        init();
    }, [connected, href]);

    const getIconElement = () => {
        let color;
        let iconType;
        const {OPENED_COLOR, CLOSED_COLOR, MERGED_COLOR} = STATE_COLOR_MAP;
        switch (data.type) {
        case LINK_TYPES.MERGE_REQUESTS:
            color = OPENED_COLOR;
            iconType = GitPullRequestIcon;
            if (data.state === STATE_TYPES.CLOSED) {
                if (data.merged) {
                    color = MERGED_COLOR;
                    iconType = GitMergeIcon;
                } else {
                    color = CLOSED_COLOR;
                }
            }
            break;
        case LINK_TYPES.ISSUES:
            color = data.state === STATE_TYPES.OPENED ? OPENED_COLOR : CLOSED_COLOR;
            iconType = data.state === STATE_TYPES.OPENED ? IssueOpenedIcon : IssueClosedIcon;
            break;
        default:
            dispatch(logError(`link type ${data.type} is not supported to display a tooltip`));
        }
        const icon = (
            <span style={{color}}>
                <Octicon
                    icon={iconType}
                    size='small'
                    verticalAlign='middle'
                />
            </span>
        );
        return icon;
    };

    if (!data) {
        return null;
    }

    const date = new Date(data.created_at).toDateString();

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

                <div className='body d-flex mt-2'>
                    <span className='pt-1 pb-1 pr-2'>
                        {getIconElement()}
                    </span>

                    <div className='tooltip-info mt-1'>
                        <a href={href}>
                            <h5 className='mr-1'>{data.title}</h5>
                            <span className='mr-number'>{`#${data.iid}`}</span>
                        </a>
                        <div className='markdown-text mt-1 mb-1'>
                            <ReactMarkdown
                                source={data.description}
                                disallowedTypes={['heading']}
                                linkTarget='_blank'
                            />
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

                        <div className='see-more mt-1'>
                            <a
                                href={href}
                                target='_blank'
                                rel='noopener noreferrer'
                            >
                                {'See more'}
                            </a>
                        </div>

                        {/* Labels */}
                        <div className='labels mt-3'>
                            {data.labels && data.labels_with_details?.length && data.labels_with_details.map((label) => {
                                return (
                                    <span
                                        key={label.name}
                                        className='label mr-1'
                                        title={label.description}
                                        style={{backgroundColor: label.color}}
                                    >
                                        <span style={{color: label.text_color}}>{label.name}</span>
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

LinkTooltip.propTypes = {
    href: PropTypes.string.isRequired,
    connected: PropTypes.bool.isRequired,
    gitlabURL: PropTypes.string.isRequired,
};
