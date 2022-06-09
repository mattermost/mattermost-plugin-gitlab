// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import Scrollbars from 'react-custom-scrollbars';
import {Theme} from 'mattermost-redux/types/preferences';
import {makeStyleFromTheme, changeOpacity} from 'mattermost-redux/utils/theme_utils';

import {RHSStates} from '../../constants';
import GitlabItems from './gitlab_items';
import {Item} from '../../types/gitlab_items';

interface PropTypes {
    username: string;
    org: string;
    gitlabURL: string;
    reviews: Item[];
    unreads: Item[],
    yourPrs: Item[],
    yourAssignments: Item[],
    rhsState: string,
    theme: Theme,
    actions:any,
};

export function renderView(props: PropTypes) {
    return (
        <div
            {...props}
            className='scrollbar--view'
        />);
}

export function renderThumbHorizontal(props: PropTypes) {
    return (
        <div
            {...props}
            className='scrollbar--horizontal'
        />);
}

export function renderThumbVertical(props: PropTypes) {
    return (
        <div
            {...props}
            className='scrollbar--vertical'
        />);
}


function mapGitlabItemListToPrList(gilist: Item[]) {
    if (!gilist) {
        return [];
    }

    return gilist.map((pr: Item) => {
        return {sha: pr.sha, project_id: pr.project_id, iid: pr.iid};
    });
}

function shouldUpdateDetails(prs: Item[], prevPrs: Item[], targetState: string, currentState: string, prevState: string) {
    if (currentState !== targetState) {
        return false
    }

    if (currentState !== prevState) {
        return true;
    }

    if (prs.length !== prevPrs.length) {
        return true;
    }

    for (let i = 0; i < prs.length; i++) {
        if (prs[i].id !== prevPrs[i].id) {
            return true;
        }
    }
}

export default class SidebarRight extends React.PureComponent<PropTypes> {

  componentDidMount() {
      if (this.props.yourPrs && this.props.rhsState === RHSStates.PRS) {
          this.props.actions.getYourPrDetails(
              mapGitlabItemListToPrList(this.props.yourPrs),
          );
      }
      if (this.props.reviews && this.props.rhsState === RHSStates.REVIEWS) {
          this.props.actions.getReviewDetails(
              mapGitlabItemListToPrList(this.props.reviews),
          );
      }
  }

  componentDidUpdate(prevProps: PropTypes) {
      if (
          shouldUpdateDetails(
              this.props.yourPrs,
              prevProps.yourPrs,
              RHSStates.PRS,
              this.props.rhsState,
              prevProps.rhsState,
          )
      ) {
          this.props.actions.getYourPrDetails(
              mapGitlabItemListToPrList(this.props.yourPrs),
          );
      }

      if (
          shouldUpdateDetails(
              this.props.reviews,
              prevProps.reviews,
              RHSStates.REVIEWS,
              this.props.rhsState,
              prevProps.rhsState,
          )
      ) {
          this.props.actions.getReviewDetails(
              mapGitlabItemListToPrList(this.props.reviews),
          );
      }
  }

  render() {
      const style = getStyle(this.props.theme)
      const baseURL:string = this.props.gitlabURL ?? 'https://gitlab.com';
      const orgQuery:string = this.props.org ? `+org%3A ${this.props.org}` : '';
    
      let title:string = '';
      let gitlabItems: Item[] = [];
      let listUrl:string = '';

      switch (this.props.rhsState) {
      case RHSStates.PRS:
          gitlabItems = this.props.yourPrs;
          title = 'Your Open Merge Requests';
          listUrl = `${baseURL}/dashboard/merge_requests?state=opened&scope=all&author_username=${this.props.username}&archived=false${orgQuery}`;
          break;
      case RHSStates.REVIEWS:
          gitlabItems = this.props.reviews;
          listUrl = `${baseURL}/dashboard/merge_requests?state=opened&scope=all&assignee_username=${this.props.username}&archived=false${orgQuery}`;
          title = 'Merge Requests Needing Review';
          break;
      case RHSStates.UNREADS:
          gitlabItems = this.props.unreads;
          title = 'Unread Messages';
          listUrl = `${baseURL}/dashboard/todos`;
          break;
      case RHSStates.ASSIGNMENTS:
          gitlabItems = this.props.yourAssignments;
          title = 'Your Assignments';
          listUrl = `${baseURL}/dashboard/issues?state=opened&scope=all&assignee_username=${this.props.username}${orgQuery}`;
          break;
      default:
          break;
      }

      return (
          <React.Fragment>
              <Scrollbars
                  autoHide={true}
                  autoHideTimeout={500}     // Hide delay in ms
                  autoHideDuration={500}     // Duration for hide animation in ms.
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
                  <div>
                      {!gitlabItems.length ? (<div style={style.container}>{'You have no active items'}</div>)
                      : gitlabItems.map((item)=>
                        <GitlabItems
                            key={item.id}
                            item={item}
                            theme={this.props.theme}
                      />
                      )}
                      
                  </div>
              </Scrollbars>
          </React.Fragment>
      );
  }
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
