// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect} from 'react';
import Scrollbars from 'react-custom-scrollbars';
import {useDispatch, useSelector} from 'react-redux';

import {Theme} from 'mattermost-redux/types/preferences';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {getYourPrDetails, getReviewDetails} from 'src/actions';
import {RHSStates} from 'src/constants';
import {getSidebarData} from 'src/selectors';
import {Item} from 'src/types/gitlab_items';
import {usePrevious} from 'src/hooks/usePrevious';

import GitlabItems from './gitlab_items';

const AUTO_HIDE_TIMEOUT = 500;
const AUTO_HIDE_DURATION = 500;

interface Props {
    username: string;
    org: string;
    gitlabURL: string;
    reviews: Item[];
    unreads: Item[],
    yourPrs: Item[],
    yourAssignments: Item[],
    rhsState: string,
    theme: Theme,
}

export const renderView = (props: Props) => (
    <div
        {...props}
        className='scrollbar--view'
    />
);

export const renderThumbHorizontal = (props: Props) => (
    <div
        {...props}
        className='scrollbar--horizontal'
    />
);

export const renderThumbVertical = (props: Props) => (
    <div
        {...props}
        className='scrollbar--vertical'
    />
);

function shouldUpdateDetails(prs: Item[], prevPrs: Item[], targetState: string, currentState: string) {
    if (currentState !== targetState) {
        return false;
    }

    if (prs.length !== prevPrs.length) {
        return true;
    }

    for (let i = 0; i < prs.length; i++) {
        if (prs[i].id !== prevPrs[i].id) {
            return true;
        }
    }

    return false;
}

function SidebarRight({theme}: {theme: Theme}) {
    const sidebarData = useSelector(getSidebarData);
    const {username, yourAssignments, org, unreads, gitlabURL, rhsState, reviews, yourPrs} = sidebarData;

    const dispatch = useDispatch();

    const prevPrs = usePrevious<Item[]>(yourPrs)
    const prevReviews = usePrevious<Item[]>(reviews)

    useEffect(() => {
        if (yourPrs && (!prevPrs || shouldUpdateDetails(yourPrs, prevPrs, RHSStates.PRS, rhsState))) {
            dispatch(getYourPrDetails(yourPrs));
        }
    }, [yourPrs, rhsState, prevPrs]);

    useEffect(() => {
        if (reviews && (!prevReviews || shouldUpdateDetails(reviews, prevReviews, RHSStates.REVIEWS, rhsState))) {
            dispatch(getReviewDetails(reviews));
        }
    }, [reviews, rhsState, prevReviews]);

    const style = getStyle(theme);
    const baseURL = gitlabURL || 'https://gitlab.com';
    let orgQuery = '/dashboard'; //default == all orgs
    if (org) {
        orgQuery = `/groups/${org}/-`;
    }

    let title = '';
    let gitlabItems: Item[] = [];
    let listUrl = '';

    switch (rhsState) {
    case RHSStates.PRS:
        gitlabItems = yourPrs;
        title = 'Your Open Merge Requests';
        listUrl = `${baseURL}${orgQuery}/merge_requests?state=opened&author_username=${username}`;
        break;
    case RHSStates.REVIEWS:
        gitlabItems = reviews;
        listUrl = `${baseURL}${orgQuery}/merge_requests?reviewer_username=${username}`;
        title = 'Merge Requests Needing Review';
        break;
    case RHSStates.UNREADS:
        gitlabItems = unreads;
        title = 'Unread Messages';
        listUrl = `${baseURL}/dashboard/todos`;
        break;
    case RHSStates.ASSIGNMENTS:
        gitlabItems = yourAssignments;
        title = 'Your Assignments';
        listUrl = `${baseURL}${orgQuery}/issues?assignee_username=${username}`;
        break;
    default:
        break;
    }

    let renderedGitlabItems: React.ReactNode = <div style={style.container}>{'You have no active items'}</div>;
    if (gitlabItems?.length) {
        renderedGitlabItems = gitlabItems.map((item) => (
            <GitlabItems
                key={item.id}
                item={item}
                theme={theme}
            />
        ));
    }

    return (
        <Scrollbars
            autoHide={true}
            autoHideTimeout={AUTO_HIDE_TIMEOUT} // Hide delay in ms
            autoHideDuration={AUTO_HIDE_DURATION} // Duration for hide animation in ms.
            renderThumbHorizontal={renderThumbHorizontal}
            renderThumbVertical={renderThumbVertical}
            renderView={renderView}
        >
            <div style={style.sectionHeader}>
                <strong>
                    <a
                        href={listUrl}
                        target='_blank'
                        rel='noopener noreferrer'
                    >
                        {title}
                    </a>
                </strong>
            </div>
            {renderedGitlabItems}
        </Scrollbars>
    );
}

const getStyle = makeStyleFromTheme((theme) => {
    return {
        container: {
            padding: '15px',
            borderTop: `1px solid ${changeOpacity(theme.centerChannelColor, 0.2)}`,
        },
        sectionHeader: {
            padding: '15px',
        },
    };
});

export default SidebarRight;
