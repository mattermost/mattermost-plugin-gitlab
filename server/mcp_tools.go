// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-plugin-agents/external/pluginmcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Common types -----------------------------------------------------------

type PaginationInput struct {
	Page    int `json:"page,omitempty" jsonschema:"Page number (1-based, default 1)"`
	PerPage int `json:"per_page,omitempty" jsonschema:"Number of results per page (default 20, max 100)"`
}

// --- Issue types ------------------------------------------------------------

type GetIssueInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format (e.g. mygroup/myproject)"`
	IssueIID    int    `json:"issue_iid" jsonschema:"Internal issue number (IID) shown in the GitLab UI, e.g. 42"`
}

type IssueSummary struct {
	ID           int      `json:"id" jsonschema:"GitLab issue database ID"`
	IID          int      `json:"iid" jsonschema:"Issue number within the project (shown in the UI)"`
	ProjectID    int      `json:"project_id"`
	Title        string   `json:"title"`
	State        string   `json:"state" jsonschema:"open or closed"`
	Description  string   `json:"description,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Assignees    []string `json:"assignees,omitempty" jsonschema:"GitLab usernames of assignees"`
	Milestone    string   `json:"milestone,omitempty" jsonschema:"Milestone title if set"`
	WebURL       string   `json:"web_url"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

type GetIssueOutput struct {
	Issue IssueSummary `json:"issue"`
}

type ListMyAssignedIssuesOutput struct {
	Issues []IssueSummary `json:"issues"`
}

type SearchIssuesInput struct {
	Search string `json:"search" jsonschema:"Full-text search term to find issues by title or description"`
}

type SearchIssuesOutput struct {
	Issues []IssueSummary `json:"issues"`
}

type CreateIssueInput struct {
	ProjectPath string   `json:"project_path" jsonschema:"Full project path in namespace/project format (e.g. mygroup/myproject)"`
	Title       string   `json:"title" jsonschema:"Issue title (required)"`
	Description string   `json:"description,omitempty" jsonschema:"Optional issue description (Markdown supported)"`
	Labels      []string `json:"labels,omitempty" jsonschema:"Optional list of label names to apply"`
	AssigneeIDs []int    `json:"assignee_ids,omitempty" jsonschema:"Optional list of GitLab user IDs to assign. Use list_project_members to look up IDs."`
	MilestoneID int      `json:"milestone_id,omitempty" jsonschema:"Optional milestone ID. Use list_project_milestones to look up IDs."`
}

type CreateIssueOutput struct {
	Issue IssueSummary `json:"issue"`
}

type UpdateIssueInput struct {
	ProjectPath string   `json:"project_path" jsonschema:"Full project path in namespace/project format (e.g. mygroup/myproject)"`
	IssueIID    int      `json:"issue_iid" jsonschema:"Internal issue number (IID)"`
	Title       *string  `json:"title,omitempty" jsonschema:"New title (omit to leave unchanged)"`
	Description *string  `json:"description,omitempty" jsonschema:"New description (omit to leave unchanged)"`
	StateEvent  *string  `json:"state_event,omitempty" jsonschema:"'close' to close the issue, 'reopen' to reopen it (omit to leave state unchanged)"`
	Labels      []string `json:"labels,omitempty" jsonschema:"Replacement label set. Omit to leave labels unchanged. Send an empty array to clear all labels."`
	AssigneeIDs []int    `json:"assignee_ids,omitempty" jsonschema:"Replacement assignee list (GitLab user IDs). Omit to leave unchanged. Send an empty array to clear all assignees."`
	MilestoneID *int     `json:"milestone_id,omitempty" jsonschema:"New milestone ID, or 0 to remove the milestone (omit to leave unchanged)"`
}

type UpdateIssueOutput struct {
	Issue IssueSummary `json:"issue"`
}

type AddIssueCommentInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
	IssueIID    int    `json:"issue_iid" jsonschema:"Internal issue number (IID)"`
	Body        string `json:"body" jsonschema:"Comment text (Markdown supported)"`
}

type AddIssueCommentOutput struct {
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

type ListMyAssignedMergeRequestsOutput struct {
	MergeRequests []MergeRequestSummary `json:"merge_requests"`
}

type ListMyReviewRequestsOutput struct {
	MergeRequests []MergeRequestSummary `json:"merge_requests"`
}

type SearchMergeRequestsInput struct {
	Search string `json:"search" jsonschema:"Full-text search term to find merge requests by title or description"`
}

type SearchMergeRequestsOutput struct {
	MergeRequests []MergeRequestSummary `json:"merge_requests"`
}

type CreateMergeRequestInput struct {
	ProjectPath  string   `json:"project_path" jsonschema:"Full project path of the source project in namespace/project format"`
	Title        string   `json:"title" jsonschema:"Merge request title (required)"`
	Description  string   `json:"description,omitempty" jsonschema:"Optional MR description (Markdown supported)"`
	SourceBranch string   `json:"source_branch" jsonschema:"Branch to merge from (required)"`
	TargetBranch string   `json:"target_branch" jsonschema:"Branch to merge into (required, e.g. main or master)"`
	AssigneeIDs  []int    `json:"assignee_ids,omitempty" jsonschema:"Optional list of GitLab user IDs to assign. Use list_project_members to look up IDs."`
	ReviewerIDs  []int    `json:"reviewer_ids,omitempty" jsonschema:"Optional list of GitLab user IDs to request review from"`
	Labels       []string `json:"labels,omitempty" jsonschema:"Optional list of label names to apply"`
	MilestoneID  *int     `json:"milestone_id,omitempty" jsonschema:"Optional milestone ID"`
}

type CreateMergeRequestOutput struct {
	MergeRequest MergeRequestSummary `json:"merge_request"`
}

type AddMergeRequestCommentInput struct {
	ProjectPath    string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
	MergeRequestID int    `json:"merge_request_iid" jsonschema:"Internal merge request number (IID)"`
	Body           string `json:"body" jsonschema:"Comment text (Markdown supported)"`
}

type AddMergeRequestCommentOutput struct {
	NoteID int    `json:"note_id" jsonschema:"ID of the newly created note/comment"`
	Body   string `json:"body"`
}

// --- Project types ----------------------------------------------------------

type ListMyProjectsOutput struct {
	Projects []ProjectSummary `json:"projects"`
}

type GetProjectInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format (e.g. mygroup/myproject)"`
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

type GetProjectOutput struct {
	Project ProjectSummary `json:"project"`
}

// --- Pipeline types ---------------------------------------------------------

type RunPipelineInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
	Ref         string `json:"ref" jsonschema:"Branch name or tag to run the pipeline on (e.g. main)"`
}

type PipelineSummary struct {
	ID        int    `json:"id"`
	Status    string `json:"status" jsonschema:"Pipeline status: pending, running, passed, failed, canceled, skipped"`
	Ref       string `json:"ref" jsonschema:"Branch or tag name"`
	SHA       string `json:"sha,omitempty" jsonschema:"Commit SHA"`
	WebURL    string `json:"web_url"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type RunPipelineOutput struct {
	Pipeline PipelineSummary `json:"pipeline"`
}

type ListProjectPipelinesInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
	Ref         string `json:"ref,omitempty" jsonschema:"Optional branch or tag to filter pipelines"`
	Status      string `json:"status,omitempty" jsonschema:"Optional status filter: pending, running, passed, failed, canceled, skipped"`
	PaginationInput
}

type ListProjectPipelinesOutput struct {
	Pipelines []PipelineSummary `json:"pipelines"`
}

// --- Todo types -------------------------------------------------------------

type TodoSummary struct {
	ID          int    `json:"id"`
	ActionName  string `json:"action_name" jsonschema:"What triggered this todo, e.g. assigned, mentioned, review_requested"`
	TargetType  string `json:"target_type" jsonschema:"Type of the target object, e.g. Issue or MergeRequest"`
	TargetTitle string `json:"target_title"`
	ProjectPath string `json:"project_path,omitempty"`
	WebURL      string `json:"web_url,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type GetMyTodosOutput struct {
	Todos []TodoSummary `json:"todos"`
}

// --- LHS dashboard types ----------------------------------------------------

type LHSDataOutput struct {
	AssignedMergeRequests []MergeRequestSummary `json:"assigned_merge_requests"`
	ReviewRequests        []MergeRequestSummary `json:"review_requests"`
	AssignedIssues        []IssueSummary        `json:"assigned_issues"`
	Todos                 []TodoSummary         `json:"todos"`
}

// --- Label / milestone types ------------------------------------------------

type ListProjectLabelsInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
}

type LabelSummary struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color,omitempty" jsonschema:"Hex color code (e.g. #428BCA)"`
	Description string `json:"description,omitempty"`
}

type ListProjectLabelsOutput struct {
	Labels []LabelSummary `json:"labels"`
}

type ListProjectMilestonesInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
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

type ListProjectMilestonesOutput struct {
	Milestones []MilestoneSummary `json:"milestones"`
}

// --- User / member types ----------------------------------------------------

type GetMyGitLabUserOutput struct {
	ID        int    `json:"id" jsonschema:"GitLab user database ID"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	WebURL    string `json:"web_url"`
}

type ListProjectMembersInput struct {
	ProjectPath string `json:"project_path" jsonschema:"Full project path in namespace/project format"`
}

type ProjectMemberSummary struct {
	ID          int    `json:"id" jsonschema:"GitLab user ID — use this value for assignee_ids and reviewer_ids"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	AccessLevel int    `json:"access_level" jsonschema:"Access level: 10=Guest, 20=Reporter, 30=Developer, 40=Maintainer, 50=Owner"`
}

type ListProjectMembersOutput struct {
	Members []ProjectMemberSummary `json:"members"`
}

// --- Tool registration ------------------------------------------------------

func (p *Plugin) registerTools(s *pluginmcp.Server) {
	// Issues
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_issue",
		Description: "Retrieve details of a single GitLab issue by project path and issue IID (the number shown in the GitLab UI). Returns title, state, description, labels, assignees, milestone, and web URL.",
	}, p.handleGetIssue)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_my_assigned_issues",
		Description: "List all open GitLab issues currently assigned to the calling user. Respects the plugin's configured namespace restriction.",
	}, p.handleListMyAssignedIssues)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "search_issues",
		Description: "Full-text search for GitLab issues by title or description. Searches within the configured namespace (group or whole instance).",
	}, p.handleSearchIssues)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "create_issue",
		Description: "Create a new GitLab issue in a project. Use list_project_labels to find valid label names, list_project_milestones for milestone IDs, and list_project_members for assignee user IDs.",
	}, p.handleCreateIssue)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "update_issue",
		Description: "Update an existing GitLab issue. Only fields explicitly provided are changed; omitted fields remain as-is. Use state_event 'close' or 'reopen' to change issue state.",
	}, p.handleUpdateIssue)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "add_issue_comment",
		Description: "Add a comment (note) to an existing GitLab issue. Markdown is supported in the body.",
	}, p.handleAddIssueComment)

	// Merge Requests
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_merge_request",
		Description: "Retrieve details of a single GitLab merge request by project path and MR IID. Returns title, state, branches, author, assignees, reviewers, labels, and web URL.",
	}, p.handleGetMergeRequest)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_my_assigned_merge_requests",
		Description: "List all open GitLab merge requests currently assigned to the calling user.",
	}, p.handleListMyAssignedMRs)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_my_review_requests",
		Description: "List all open GitLab merge requests where the calling user has been requested as a reviewer.",
	}, p.handleListMyReviewRequests)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "search_merge_requests",
		Description: "Full-text search for GitLab merge requests by title or description.",
	}, p.handleSearchMergeRequests)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "create_merge_request",
		Description: "Create a new GitLab merge request. source_branch and target_branch are required. Use list_project_members to look up assignee and reviewer user IDs.",
	}, p.handleCreateMergeRequest)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "add_merge_request_comment",
		Description: "Add a comment (note) to an existing GitLab merge request. Markdown is supported in the body.",
	}, p.handleAddMergeRequestComment)

	// Projects
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_my_projects",
		Description: "List GitLab projects the calling user has access to. If the plugin is configured with a group restriction, only projects within that group are returned.",
	}, p.handleListMyProjects)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_project",
		Description: "Get details for a specific GitLab project by its full path (namespace/project).",
	}, p.handleGetProject)

	// Pipelines
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "run_pipeline",
		Description: "Trigger a new CI/CD pipeline for a given project and branch or tag ref.",
	}, p.handleRunPipeline)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_project_pipelines",
		Description: "List recent CI/CD pipelines for a project. Optionally filter by ref (branch/tag) and status (pending, running, passed, failed, canceled, skipped).",
	}, p.handleListProjectPipelines)

	// Todos
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_my_todos",
		Description: "List the calling user's GitLab to-do items — issues and merge requests that require their attention (assigned, mentioned, review requested, etc.).",
	}, p.handleGetMyTodos)

	// Dashboard
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_gitlab_dashboard",
		Description: "Return a combined overview of the calling user's GitLab workload in a single call: assigned merge requests, review requests, assigned issues, and todos. Useful for a quick situational awareness check.",
	}, p.handleGetGitLabDashboard)

	// Labels and milestones
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_project_labels",
		Description: "List all labels defined for a GitLab project. Use this to find valid label names before creating or updating issues and merge requests.",
	}, p.handleListProjectLabels)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_project_milestones",
		Description: "List active milestones for a GitLab project. Includes group-level milestones when the project belongs to a group.",
	}, p.handleListProjectMilestones)

	// User / metadata
	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "get_my_gitlab_user",
		Description: "Return the calling user's GitLab profile: username, name, email, and web URL. Useful for agents to identify who they are acting as.",
	}, p.handleGetMyGitLabUser)

	pluginmcp.AddTool(s, &mcp.Tool{
		Name:        "list_project_members",
		Description: "List all members of a GitLab project with their user IDs and access levels. Use the returned id values as assignee_ids or reviewer_ids when creating issues or merge requests.",
	}, p.handleListProjectMembers)
}
