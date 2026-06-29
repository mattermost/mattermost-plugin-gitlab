// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-agents/external/pluginmcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func ptr[T any](v T) *T { return &v }

// --- Issue types ------------------------------------------------------------

type GetIssueInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format (e.g. mygroup/myproject)"`
	IssueIID    int    `json:"issue_iid" jsonschema:"Internal issue number (IID) shown in the GitLab UI, e.g. 42"`
}

type IssueSummary struct {
	ID          int      `json:"id" jsonschema:"GitLab issue database ID"`
	IID         int      `json:"iid" jsonschema:"Issue number within the project (shown in the UI)"`
	ProjectID   int      `json:"project_id"`
	Title       string   `json:"title"`
	State       string   `json:"state" jsonschema:"open or closed"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Assignees   []string `json:"assignees,omitempty" jsonschema:"GitLab usernames of assignees"`
	Milestone   string   `json:"milestone,omitempty" jsonschema:"Milestone title if set"`
	WebURL      string   `json:"web_url"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type GetIssueOutput struct {
	Issue IssueSummary `json:"issue"`
}

type ListIssuesInput struct {
	Search       string `json:"search,omitempty" jsonschema:"Keyword to search issue titles and descriptions. When omitted, the issues assigned to you are returned instead."`
	AssignedToMe bool   `json:"assigned_to_me,omitempty" jsonschema:"Force the assigned-to-me list even when a search term is given"`
}

type ListIssuesOutput struct {
	Issues []IssueSummary `json:"issues"`
}

type CreateIssueInput struct {
	ProjectPath string   `json:"project_path" jsonschema:"Full project path in namespace/project format (e.g. mygroup/myproject)"`
	Title       string   `json:"title" jsonschema:"Issue title (required)"`
	Description string   `json:"description,omitempty" jsonschema:"Optional issue description (Markdown supported)"`
	Labels      []string `json:"labels,omitempty" jsonschema:"Optional list of label names to apply"`
	AssigneeIDs []int    `json:"assignee_ids,omitempty" jsonschema:"Optional list of GitLab user IDs to assign"`
	MilestoneID int      `json:"milestone_id,omitempty" jsonschema:"Optional milestone ID"`
}

type CreateIssueOutput struct {
	Issue IssueSummary `json:"issue"`
}

type UpdateIssueInput struct {
	ProjectPath string   `json:"project_path" jsonschema:"Full project path in namespace/project format"`
	IssueIID    int      `json:"issue_iid" jsonschema:"Internal issue number (IID)"`
	Title       *string  `json:"title,omitempty" jsonschema:"New title (omit to leave unchanged)"`
	Description *string  `json:"description,omitempty" jsonschema:"New description (omit to leave unchanged)"`
	StateEvent  *string  `json:"state_event,omitempty" jsonschema:"close or reopen (omit to leave state unchanged)"`
	Labels      []string `json:"labels,omitempty" jsonschema:"Replacement label set. Omit to leave unchanged, send an empty array to clear."`
	AssigneeIDs []int    `json:"assignee_ids,omitempty" jsonschema:"Replacement assignee list (user IDs). Omit to leave unchanged, send an empty array to clear."`
	MilestoneID *int     `json:"milestone_id,omitempty" jsonschema:"New milestone ID, or 0 to remove the milestone (omit to leave unchanged)"`
}

type UpdateIssueOutput struct {
	Issue IssueSummary `json:"issue"`
}

// --- Comment types ----------------------------------------------------------

type AddCommentInput struct {
	TargetType  string `json:"target_type" jsonschema:"What to comment on: 'issue' or 'merge_request'"`
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
	TargetIID   int    `json:"target_iid" jsonschema:"Internal number (IID) of the issue or merge request"`
	Body        string `json:"body" jsonschema:"Comment text (Markdown supported)"`
}

type AddCommentOutput struct {
	NoteID int    `json:"note_id" jsonschema:"ID of the newly created note/comment"`
	Body   string `json:"body"`
	WebURL string `json:"web_url,omitempty"`
}

// --- Merge request types ----------------------------------------------------

type GetMergeRequestInput struct {
	ProjectPath    string `json:"project_path" jsonschema:"Full project path in namespace/project format (e.g. mygroup/myproject)"`
	MergeRequestID int    `json:"merge_request_iid" jsonschema:"Internal merge request number (IID) shown in the GitLab UI"`
}

type MergeRequestSummary struct {
	ID           int      `json:"id" jsonschema:"GitLab merge request database ID"`
	IID          int      `json:"iid" jsonschema:"Merge request number within the project (shown in the UI)"`
	ProjectID    int      `json:"project_id"`
	Title        string   `json:"title"`
	State        string   `json:"state" jsonschema:"opened, closed, locked, or merged"`
	Description  string   `json:"description,omitempty"`
	SourceBranch string   `json:"source_branch"`
	TargetBranch string   `json:"target_branch"`
	Author       string   `json:"author" jsonschema:"GitLab username of the MR author"`
	Assignees    []string `json:"assignees,omitempty" jsonschema:"GitLab usernames of assignees"`
	Reviewers    []string `json:"reviewers,omitempty" jsonschema:"GitLab usernames of reviewers"`
	Labels       []string `json:"labels,omitempty"`
	Milestone    string   `json:"milestone,omitempty"`
	WebURL       string   `json:"web_url"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

type GetMergeRequestOutput struct {
	MergeRequest MergeRequestSummary `json:"merge_request"`
}

type ListMergeRequestsInput struct {
	Search          string `json:"search,omitempty" jsonschema:"Keyword to search MR titles and descriptions"`
	AssignedToMe    bool   `json:"assigned_to_me,omitempty" jsonschema:"List MRs assigned to you (the default when no other filter is set)"`
	ReviewRequested bool   `json:"review_requested,omitempty" jsonschema:"List MRs awaiting your review"`
}

type ListMergeRequestsOutput struct {
	MergeRequests []MergeRequestSummary `json:"merge_requests"`
}

// --- Project types ----------------------------------------------------------

type GetProjectsInput struct {
	ProjectPath string `json:"project_path,omitempty" jsonschema:"Full project path (namespace/project) to fetch a single project. Omit to list the projects you can access."`
}

type ProjectSummary struct {
	ID                int    `json:"id" jsonschema:"GitLab project database ID"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace" jsonschema:"Full path including group/subgroup"`
	Description       string `json:"description,omitempty"`
	WebURL            string `json:"web_url"`
	Visibility        string `json:"visibility" jsonschema:"public, internal, or private"`
	DefaultBranch     string `json:"default_branch,omitempty"`
}

type GetProjectsOutput struct {
	Projects []ProjectSummary `json:"projects"`
}

// --- Project metadata types -------------------------------------------------

type GetProjectMetadataInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
	Kind        string `json:"kind" jsonschema:"Which metadata to return: 'labels', 'milestones', or 'members'"`
}

type LabelSummary struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color,omitempty" jsonschema:"Hex color code (e.g. #428BCA)"`
	Description string `json:"description,omitempty"`
}

type MilestoneSummary struct {
	ID          int    `json:"id"`
	IID         int    `json:"iid"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	State       string `json:"state" jsonschema:"active or closed"`
	DueDate     string `json:"due_date,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
}

type ProjectMemberSummary struct {
	ID          int    `json:"id" jsonschema:"GitLab user ID — use this value for assignee_ids"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	AccessLevel int    `json:"access_level" jsonschema:"Access level: 10=Guest, 20=Reporter, 30=Developer, 40=Maintainer, 50=Owner"`
}

type GetProjectMetadataOutput struct {
	Labels     []LabelSummary         `json:"labels,omitempty"`
	Milestones []MilestoneSummary     `json:"milestones,omitempty"`
	Members    []ProjectMemberSummary `json:"members,omitempty"`
}

// --- User types -------------------------------------------------------------

type GetMyGitLabUserOutput struct {
	ID        int    `json:"id" jsonschema:"GitLab user database ID"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	WebURL    string `json:"web_url"`
}

// --- Tool registration ------------------------------------------------------

// registerTools registers the GitLab MCP tools. The set is intentionally kept
// small (every tool's schema is injected into each LLM call, see the pluginmcp
// budget of ~10 tools): related read/search/list operations are merged into
// single tools with mode flags rather than exposed as separate tools.
func (p *Plugin) registerTools(s *pluginmcp.Server) {
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true}
	additive := &mcp.ToolAnnotations{DestructiveHint: ptr(false)}
	// update_issue overwrites existing fields and can close issues, so it is
	// classified as destructive rather than reusing the additive annotation.
	destructive := &mcp.ToolAnnotations{DestructiveHint: ptr(true)}

	// Issues
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_issue",
		Description: "Fetch one issue's full details by project path and IID. For a keyword search or your assigned issues use list_issues.",
		Annotations: readOnly,
	}, p.handleGetIssue)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_issues",
		Description: "List the issues assigned to you, or search issues by keyword. Returns issue summaries (capped to GitLab's default page size). For a single issue use get_issue.",
		Annotations: readOnly,
	}, p.handleListIssues)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "create_issue",
		Description: "Create an issue in a project and return it. Resolve label names and assignee IDs first with get_project_metadata.",
		Annotations: additive,
	}, p.handleCreateIssue)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "update_issue",
		Description: "Change an existing issue's fields or open/close state and return it; omitted fields are left untouched. To only add a comment use add_comment.",
		Annotations: destructive,
	}, p.handleUpdateIssue)

	// Comments (issues and merge requests)
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "add_comment",
		Description: "Post a comment on an issue or merge request, selected via target_type, and return the created note.",
		Annotations: additive,
	}, p.handleAddComment)

	// Merge requests
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_merge_request",
		Description: "Fetch one merge request's full details by project path and IID. For lists or keyword search use list_merge_requests.",
		Annotations: readOnly,
	}, p.handleGetMergeRequest)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_merge_requests",
		Description: "List the merge requests assigned to you (default), awaiting your review, or matching a keyword search (capped to GitLab's default page size). For a single MR use get_merge_request.",
		Annotations: readOnly,
	}, p.handleListMergeRequests)

	// Projects
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_projects",
		Description: "List the projects you can access, or fetch a single project when project_path is set (results capped to GitLab's default page size).",
		Annotations: readOnly,
	}, p.handleGetProjects)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_project_metadata",
		Description: "Fetch a project's labels, milestones, or members (choose with kind). Use it to resolve label names and user IDs before create_issue or update_issue.",
		Annotations: readOnly,
	}, p.handleGetProjectMetadata)

	// User
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_my_gitlab_user",
		Description: "Return your own GitLab identity (id, username, name). Use the id for assignee_ids when creating or updating issues.",
		Annotations: readOnly,
	}, p.handleGetMyGitLabUser)
}
