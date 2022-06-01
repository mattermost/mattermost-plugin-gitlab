import React, {useEffect, useState} from 'react';
import PropTypes from 'prop-types';
import './tooltip.css';
import Octicon, {GitMerge, GitPullRequest, IssueClosed, IssueOpened} from '@primer/octicons-react';
import ReactMarkdown from 'react-markdown';

import Client from '../../client';

export const LinkTooltip = ({href, connected}) => {
    const [data, setData] = useState(null);
    useEffect(() => {
        const init = async () => {
            if (href.includes('gitlab.com/')) {
                const [owner, repo, dash, type, number] = href.split('gitlab.com/')[1].split('/');
                let res;
                switch (type) {
                case 'issues':
                    res = await Client.getIssue(owner, repo, number);
                    break;
                case 'merge_requests':
                    res = await Client.getPullRequest(owner, repo, number);
                    break;
                }
                if (res) {
                    res.owner = owner;
                    res.repo = repo;
                    res.type = type;
                }
                setData(res);
            }
        };
        if (data) {
            return;
        }
        if (connected) {
            init();
        }
    }, []);

    const getIconElement = () => {
        let icon;
        let color;
        let iconType;
        switch (data.type) {
            case 'merge_requests':
                color = '#28a745';
                iconType = GitPullRequest;
                if (data.state === 'closed') {
                    if (data.merged) {
                        color = '#6f42c1';
                        iconType = GitMerge;
                    } else {
                        color = '#cb2431';
                    }
                }
                icon = (
                    <span style={{color}}>
                        <Octicon
                            icon={iconType}
                            size='small'
                            verticalAlign='middle'
                        />
                    </span>
                );
                break;
            case 'issues':
                color = data.state === 'opened' ? '#28a745' : '#cb2431';
                iconType = data.state === 'opened' ? IssueOpened : IssueClosed;
                icon = (
                    <span style={{color}}>
                        <Octicon
                            icon={iconType}
                            size='small'
                            verticalAlign='middle'
                        />
                    </span>
                );
                break;
        }
        return icon;
    };

    if (data) {
        let date = new Date(data.created_at);
        date = date.toDateString();

        return (
            <div className='gitlab-tooltip'>
                <div className='gitlab-tooltip box gitlab-tooltip--large gitlab-tooltip--bottom-left p-4'>
                    <div className='header mb-1'>
                        <a
                            title={data.repo}
                            href={href}
                        >{data.repo}</a>&nbsp;on&nbsp;<span>{date}</span>
                    </div>

                    <div className='body d-flex mt-2'>
                        <span className='pt-1 pb-1 pr-2'>
                            { getIconElement() }
                        </span>

                        {/* info */}
                        <div className='tooltip-info mt-1'>
                            <a href={href}>
                                <h5 className='mr-1'>{data.title}</h5>
                                <span>#{data.number}</span>
                            </a>
                            <div className='markdown-text mt-1 mb-1'>
                                <ReactMarkdown
                                    source={data.description}
                                    disallowedTypes={['heading']}
                                    linkTarget='_blank'
                                />
                            </div>

                            {/* base <- head */}
                            {data.type === 'merge_requests' && (
                                <div className='base-head mt-1 mr-3'>
                                    <span
                                        title={data.target_branch}
                                        className='commit-ref'
                                        style={{maxWidth: '140px'}}
                                    >{data.target_branch}
                                    </span>
                                    <span className='mx-1'>←</span>
                                    <span
                                        title={data.source_branch}
                                        className='commit-ref'
                                    >{data.source_branch}
                                    </span>
                                </div>
                            )}

                            <div className='see-more mt-1'>
                                <a
                                    href={href}
                                    target='_blank'
                                    rel='noopener noreferrer'
                                >See more</a>
                            </div>

                            {/* Labels */}
                            <div className='labels mt-3'>
                                {data.labels && data.labels_with_details && data.labels_with_details.map((label, idx) => {
                                    return (
                                        <span
                                            key={idx}
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
    }
    return null;
};

LinkTooltip.propTypes = {
    href: PropTypes.string.isRequired,
    connected: PropTypes.bool.isRequired,
};
