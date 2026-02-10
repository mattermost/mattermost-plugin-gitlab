// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

// CreateIssueAuditParams holds request audit data for the createIssue transaction.
type CreateIssueAuditParams struct {
	MattermostUserID string   `json:"mattermost_user_id"`
	GitlabUsername   string   `json:"gitlab_username"`
	ProjectID        int      `json:"project_id"`
	Title            string   `json:"title"`
	DescriptionLen   int      `json:"description_length"`
	MilestoneID      int      `json:"milestone_id"`
	AssigneeIDs      []int    `json:"assignee_ids"`
	Labels           []string `json:"labels"`
	PostID           string   `json:"post_id"`
	ChannelID        string   `json:"channel_id"`
}

func (p CreateIssueAuditParams) Auditable() map[string]any {
	return map[string]any{
		"mattermost_user_id": p.MattermostUserID, "gitlab_username": p.GitlabUsername,
		"project_id": p.ProjectID, "title": p.Title, "description_length": p.DescriptionLen,
		"milestone_id": p.MilestoneID, "assignee_ids": p.AssigneeIDs, "labels": p.Labels,
		"post_id": p.PostID, "channel_id": p.ChannelID,
	}
}

// CreateIssueAuditResult holds GitLab response for createIssue (for AddEventResultState).
type CreateIssueAuditResult struct {
	ProjectID int    `json:"project_id"`
	IssueIID  int    `json:"issue_iid"`
	WebURL    string `json:"web_url"`
}

func (p CreateIssueAuditResult) Auditable() map[string]any {
	return map[string]any{"project_id": p.ProjectID, "issue_iid": p.IssueIID, "web_url": p.WebURL}
}

// AttachCommentToIssueAuditParams holds request audit data for the attachCommentToIssue transaction.
type AttachCommentToIssueAuditParams struct {
	MattermostUserID      string `json:"mattermost_user_id"`
	GitlabUsername        string `json:"gitlab_username"`
	ProjectID             int    `json:"project_id"`
	IssueIID              int    `json:"issue_iid"`
	PostID                string `json:"post_id"`
	CommentAuthorUsername string `json:"comment_author_username"`
}

func (p AttachCommentToIssueAuditParams) Auditable() map[string]any {
	return map[string]any{
		"mattermost_user_id": p.MattermostUserID, "gitlab_username": p.GitlabUsername,
		"project_id": p.ProjectID, "issue_iid": p.IssueIID, "post_id": p.PostID,
		"comment_author_username": p.CommentAuthorUsername,
	}
}

// AttachCommentToIssueAuditResult holds GitLab response for attachCommentToIssue (for AddEventResultState).
type AttachCommentToIssueAuditResult struct {
	ProjectID int `json:"project_id"`
	NoteID    int `json:"note_id"`
}

func (p AttachCommentToIssueAuditResult) Auditable() map[string]any {
	return map[string]any{"project_id": p.ProjectID, "note_id": p.NoteID}
}
