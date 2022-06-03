import React from 'react';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';
import {Badge} from "react-bootstrap";
import * as CSS from 'csstype';
import {GitPullRequestIcon, IssueOpenedIcon, IconProps} from '@primer/octicons-react';
import SignIcon from '../../images/icons/sign';
import {formatTimeSince} from '../../utils/date_utils';
import {GitlabItemsProps, Label} from "../../types/gitlab_items"

export const notificationReasons: Record<string, string> = {
  assigned: 'You were assigned to the issue/merge request',
  review_requested: 'You were requested to review a merge request.',
  mentioned: 'You were specifically @mentioned in the content.',
  build_failed: 'Gitlab build was failed.',
  marked: 'Task is marked as done.',
  approval_required: 'Your approval is required on this issue/merge request.',
  unmergeable: 'This merge request can not be merged.',
  directly_addressed: 'You were directly addressed.',
  merge_train_removed: 'A merge train was removed.',
  attention_required: 'Your attention is required on the issue/merge request.',
};

function GitlabItems({item, theme}: GitlabItemsProps) {
    const style = getStyle(theme);

    const repoName = item.references?.full ?? item.project?.path_with_namespace ?? '';
    const userName = item.author?.username ?? '';

    let number: JSX.Element | null = null;
    if (item.iid) {
        const iconProps: IconProps = {
            size: 'small',
            verticalAlign: 'text-bottom',
        };
        const icon = item.merge_status ?
            <GitPullRequestIcon {...iconProps} /> : // item is a pull request
            <IssueOpenedIcon {...iconProps} />;
        number = (
            <strong>
                <span style={{...style.icon}}>{icon}</span>
                {`#${item.iid}`}
            </strong>
        );
    }

    const titleText = item.title ?? item.target?.title ?? '';

    let title: JSX.Element | null = <>{titleText}</>;
    if (item.web_url || item.target_url) {
        title = (
            <a
                href={item.web_url ?? item.target_url}
                target='_blank'
                rel='noopener noreferrer'
                style={style.itemTitle}
            >
                {titleText}
            </a>
        );
        if (item.iid) {
            number = (
                <strong>
                    <a href={item.web_url} target='_blank' rel='noopener noreferrer'>
                        {number}
                    </a>
                </strong>
            );
        }
    }

    const milestone: JSX.Element | null = item.milestone ? (
        <span
            style={{
                ...style.milestoneIcon,
                ...style.icon,
                ...((item.created_at || userName) && {
                    paddingLeft: 10,
                }),
            }}
        >
            <SignIcon />
            {item.milestone.title}
        </span>
    ) : null;

    let labels: JSX.Element[] | null = item.labels ? getGitlabLabels(item.labels) : null;

    return (
        <div key={item.id} style={style.container}>
            <div>
                <strong>
                    {title}
                </strong>
            </div>
            <div>
                {number}
                <span className='light'>{`(${repoName})`}</span>
            </div>
            {labels}
            <div className='light' style={style.subtitle}>
                {item.created_at && `Opened ${formatTimeSince(item.created_at)} ago ${userName && ` by ${userName}.`}`}
                {milestone}
            </div>
            <div className="light" style={style.subtitle}>
                {item.action_name ? (
                    <>
                        <div>{item.updated_at && `${formatTimeSince(item.updated_at)} ago`}</div>
                        {notificationReasons[item.action_name]}
                    </>
                ) : null}
            </div>
        </div>
    );
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        container: {
            padding: '15px',
            borderTop: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
        },
        itemTitle: {
            color: theme.centerChannelColor,
            lineHeight: 1.7,
            fontWeight: 'bold',
        },
        subtitle: {
            margin: '5px 0 0 0',
            fontSize: '13px',
        },
        subtitleSecondLine: {
            fontSize: '13px',
        },
        icon: {
            top: '3px',
            position: 'relative',
            display: 'inline-flex',
            alignItems: 'center',
            marginRight: '6px',
        },
        iconSuccess: {
            color: theme.onlineIndicator,
        },
        iconPending: {
            color: theme.awayIndicator,
        },
        iconFailed: {
            color: theme.dndIndicator,
        },
        iconChangesRequested: {
            color: theme.dndIndicator,
        },
        conflictIcon: {
            color: theme.dndIndicator,
        },
        milestoneIcon: {
            color: theme.centerChannelColor,
        },
    };
});

const getGitlabLabels = (labels: Label[]) => {
    return labels.map((label) => {
        return (
            <Badge
                key={label.id}
                style={{
                    ...itemStyle,
                    ...{
                        backgroundColor: `${label.color}`,
                        color: `${label.text_color}`,
                    },
                }}
            >
                {label.name}
            </Badge>
        );
    });
}

const itemStyle: CSS.Properties = {
    margin: '4px 5px 0 0',
    padding: '3px 8px',
    display: 'inline-flex',
    borderRadius: '3px',
    position: 'relative',
};

export default GitlabItems;
