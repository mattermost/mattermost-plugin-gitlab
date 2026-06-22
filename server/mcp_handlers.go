// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	internGitlab "github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

// ============================================================================
// Issues
// ============================================================================

func (p *Plugin) handleGetIssue(ctx context.Context, _ *mcp.CallToolRequest, in GetIssueInput) (*mcp.CallToolResult, GetIssueOutput, error) {
	if in.ProjectPath == "" {
		return nil, GetIssueOutput{}, fmt.Errorf("project_path is required")
	}
	if in.IssueIID <= 0 {
		return nil, GetIssueOutput{}, fmt.Errorf("issue_iid must be a positive integer")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, GetIssueOutput{}, err
	}

	owner, repo, err := splitProjectPath(in.ProjectPath)
	if err != nil {
		return nil, GetIssueOutput{}, err
	}

	issue, err := p.GitlabClient.GetIssueByID(ctx, info, owner, repo, in.IssueIID, token)
	if err != nil {
		return nil, GetIssueOutput{}, fmt.Errorf("failed to get issue: %w", err)
	}

	return nil, GetIssueOutput{Issue: issueToSummary(issue.Issue)}, nil
}

// handleListIssues returns either the caller's assigned issues or the results
// of a keyword search. assigned_to_me (or an empty search) selects the assigned
// list; otherwise the search term is used.
func (p *Plugin) handleListIssues(ctx context.Context, _ *mcp.CallToolRequest, in ListIssuesInput) (*mcp.CallToolResult, ListIssuesOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListIssuesOutput{}, err
	}

	var issues []*internGitlab.Issue
	if in.AssignedToMe || in.Search == "" {
		issues, err = p.GitlabClient.ListAssignedIssues(ctx, info, token)
	} else {
		issues, err = p.GitlabClient.SearchIssues(ctx, info, in.Search, token)
	}
	if err != nil {
		return nil, ListIssuesOutput{}, fmt.Errorf("failed to list issues: %w", err)
	}

	return nil, ListIssuesOutput{Issues: issuesToSummaries(issues)}, nil
}

func (p *Plugin) handleCreateIssue(ctx context.Context, _ *mcp.CallToolRequest, in CreateIssueInput) (*mcp.CallToolResult, CreateIssueOutput, error) {
	if in.ProjectPath == "" {
		return nil, CreateIssueOutput{}, fmt.Errorf("project_path is required")
	}
	if in.Title == "" {
		return nil, CreateIssueOutput{}, fmt.Errorf("title is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, CreateIssueOutput{}, err
	}

	owner, repo, err := splitProjectPath(in.ProjectPath)
	if err != nil {
		return nil, CreateIssueOutput{}, err
	}

	req := &gitlab.IssueRequest{
		Title:       in.Title,
		Description: in.Description,
		Assignees:   in.AssigneeIDs,
		Milestone:   in.MilestoneID,
		Labels:      internGitlab.LabelOptions(in.Labels),
	}

	project, err := p.GitlabClient.GetProject(ctx, info, token, owner, repo)
	if err != nil {
		return nil, CreateIssueOutput{}, fmt.Errorf("failed to resolve project %q: %w", in.ProjectPath, err)
	}
	req.ProjectID = project.ID

	issue, err := p.GitlabClient.CreateIssue(ctx, info, req, token)
	if err != nil {
		return nil, CreateIssueOutput{}, fmt.Errorf("failed to create issue: %w", err)
	}

	return nil, CreateIssueOutput{Issue: issueToSummary(issue)}, nil
}

func (p *Plugin) handleUpdateIssue(ctx context.Context, _ *mcp.CallToolRequest, in UpdateIssueInput) (*mcp.CallToolResult, UpdateIssueOutput, error) {
	if in.ProjectPath == "" {
		return nil, UpdateIssueOutput{}, fmt.Errorf("project_path is required")
	}
	if in.IssueIID <= 0 {
		return nil, UpdateIssueOutput{}, fmt.Errorf("issue_iid must be a positive integer")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, UpdateIssueOutput{}, err
	}

	opts := &gitlab.UpdateIssueOptions{
		Title:       in.Title,
		Description: in.Description,
		StateEvent:  in.StateEvent,
		MilestoneID: in.MilestoneID,
	}
	if in.Labels != nil {
		labels := internGitlab.LabelOptions(in.Labels)
		opts.Labels = &labels
	}
	if in.AssigneeIDs != nil {
		opts.AssigneeIDs = &in.AssigneeIDs
	}

	issue, err := p.GitlabClient.UpdateIssue(ctx, info, token, in.ProjectPath, in.IssueIID, opts)
	if err != nil {
		return nil, UpdateIssueOutput{}, fmt.Errorf("failed to update issue: %w", err)
	}

	return nil, UpdateIssueOutput{Issue: issueToSummary(issue)}, nil
}

// ============================================================================
// Comments
// ============================================================================

// handleAddComment posts a note to an issue or merge request, routing on
// target_type so a single tool covers both surfaces.
func (p *Plugin) handleAddComment(ctx context.Context, _ *mcp.CallToolRequest, in AddCommentInput) (*mcp.CallToolResult, AddCommentOutput, error) {
	if in.ProjectPath == "" {
		return nil, AddCommentOutput{}, fmt.Errorf("project_path is required")
	}
	if in.TargetIID <= 0 {
		return nil, AddCommentOutput{}, fmt.Errorf("target_iid must be a positive integer")
	}
	if in.Body == "" {
		return nil, AddCommentOutput{}, fmt.Errorf("body is required")
	}

	var urlKind string
	switch in.TargetType {
	case "issue":
		urlKind = "issues"
	case "merge_request":
		urlKind = "merge_requests"
	default:
		return nil, AddCommentOutput{}, fmt.Errorf("target_type must be 'issue' or 'merge_request'")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, AddCommentOutput{}, err
	}

	var note *internGitlab.Note
	if in.TargetType == "issue" {
		note, err = p.GitlabClient.AddIssueNote(ctx, info, token, in.ProjectPath, in.TargetIID, in.Body)
	} else {
		note, err = p.GitlabClient.AddMergeRequestNote(ctx, info, token, in.ProjectPath, in.TargetIID, in.Body)
	}
	if err != nil {
		return nil, AddCommentOutput{}, fmt.Errorf("failed to add comment: %w", err)
	}

	return nil, AddCommentOutput{
		NoteID: note.ID,
		Body:   note.Body,
		WebURL: noteWebURL(p.getConfiguration().GitlabURL, in.ProjectPath, urlKind, in.TargetIID, note.ID),
	}, nil
}

// ============================================================================
// Merge Requests
// ============================================================================

func (p *Plugin) handleGetMergeRequest(ctx context.Context, _ *mcp.CallToolRequest, in GetMergeRequestInput) (*mcp.CallToolResult, GetMergeRequestOutput, error) {
	if in.ProjectPath == "" {
		return nil, GetMergeRequestOutput{}, fmt.Errorf("project_path is required")
	}
	if in.MergeRequestID <= 0 {
		return nil, GetMergeRequestOutput{}, fmt.Errorf("merge_request_iid must be a positive integer")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, GetMergeRequestOutput{}, err
	}

	owner, repo, err := splitProjectPath(in.ProjectPath)
	if err != nil {
		return nil, GetMergeRequestOutput{}, err
	}

	mr, err := p.GitlabClient.GetMergeRequestByID(ctx, info, owner, repo, in.MergeRequestID, token)
	if err != nil {
		return nil, GetMergeRequestOutput{}, fmt.Errorf("failed to get merge request: %w", err)
	}

	return nil, GetMergeRequestOutput{MergeRequest: mrToSummary(mr.MergeRequest)}, nil
}

// handleListMergeRequests returns the caller's assigned MRs (default), the MRs
// awaiting their review, or keyword search results.
func (p *Plugin) handleListMergeRequests(ctx context.Context, _ *mcp.CallToolRequest, in ListMergeRequestsInput) (*mcp.CallToolResult, ListMergeRequestsOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListMergeRequestsOutput{}, err
	}

	var mrs []*internGitlab.MergeRequest
	switch {
	case in.Search != "" && !in.AssignedToMe && !in.ReviewRequested:
		mrs, err = p.GitlabClient.SearchMergeRequests(ctx, info, token, in.Search)
	case in.ReviewRequested:
		mrs, err = p.GitlabClient.ListReviewRequests(ctx, info, token)
	default:
		mrs, err = p.GitlabClient.ListAssignedMergeRequests(ctx, info, token)
	}
	if err != nil {
		return nil, ListMergeRequestsOutput{}, fmt.Errorf("failed to list merge requests: %w", err)
	}

	return nil, ListMergeRequestsOutput{MergeRequests: mrsToSummaries(mrs)}, nil
}

// ============================================================================
// Projects
// ============================================================================

// handleGetProjects lists the caller's accessible projects, or returns a single
// project when project_path is supplied.
func (p *Plugin) handleGetProjects(ctx context.Context, _ *mcp.CallToolRequest, in GetProjectsInput) (*mcp.CallToolResult, GetProjectsOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, GetProjectsOutput{}, err
	}

	if in.ProjectPath != "" {
		owner, repo, splitErr := splitProjectPath(in.ProjectPath)
		if splitErr != nil {
			return nil, GetProjectsOutput{}, splitErr
		}
		project, projErr := p.GitlabClient.GetProject(ctx, info, token, owner, repo)
		if projErr != nil {
			return nil, GetProjectsOutput{}, fmt.Errorf("failed to get project: %w", projErr)
		}
		return nil, GetProjectsOutput{Projects: []ProjectSummary{projectToSummary(project)}}, nil
	}

	projects, err := p.GitlabClient.GetYourProjects(ctx, info, token)
	if err != nil {
		return nil, GetProjectsOutput{}, fmt.Errorf("failed to list projects: %w", err)
	}

	summaries := make([]ProjectSummary, 0, len(projects))
	for _, project := range projects {
		summaries = append(summaries, projectToSummary(project))
	}

	return nil, GetProjectsOutput{Projects: summaries}, nil
}

// handleGetProjectMetadata returns a project's labels, milestones, or members
// depending on the requested kind.
func (p *Plugin) handleGetProjectMetadata(ctx context.Context, _ *mcp.CallToolRequest, in GetProjectMetadataInput) (*mcp.CallToolResult, GetProjectMetadataOutput, error) {
	if in.ProjectPath == "" {
		return nil, GetProjectMetadataOutput{}, fmt.Errorf("project_path is required")
	}

	switch in.Kind {
	case "labels", "milestones", "members":
	default:
		return nil, GetProjectMetadataOutput{}, fmt.Errorf("kind must be 'labels', 'milestones', or 'members'")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, GetProjectMetadataOutput{}, err
	}

	var out GetProjectMetadataOutput
	switch in.Kind {
	case "labels":
		labels, lErr := p.GitlabClient.GetLabels(ctx, info, in.ProjectPath, token)
		if lErr != nil {
			return nil, GetProjectMetadataOutput{}, fmt.Errorf("failed to list labels: %w", lErr)
		}
		out.Labels = make([]LabelSummary, 0, len(labels))
		for _, l := range labels {
			out.Labels = append(out.Labels, LabelSummary{
				ID:          l.ID,
				Name:        l.Name,
				Color:       l.Color,
				Description: l.Description,
			})
		}
	case "milestones":
		milestones, mErr := p.GitlabClient.GetMilestones(ctx, info, in.ProjectPath, token)
		if mErr != nil {
			return nil, GetProjectMetadataOutput{}, fmt.Errorf("failed to list milestones: %w", mErr)
		}
		out.Milestones = make([]MilestoneSummary, 0, len(milestones))
		for _, m := range milestones {
			ms := MilestoneSummary{
				ID:          m.ID,
				IID:         m.IID,
				Title:       m.Title,
				Description: m.Description,
				State:       m.State,
			}
			if m.DueDate != nil {
				ms.DueDate = m.DueDate.String()
			}
			if m.StartDate != nil {
				ms.StartDate = m.StartDate.String()
			}
			out.Milestones = append(out.Milestones, ms)
		}
	case "members":
		members, memErr := p.GitlabClient.GetProjectMembers(ctx, info, in.ProjectPath, token)
		if memErr != nil {
			return nil, GetProjectMetadataOutput{}, fmt.Errorf("failed to list project members: %w", memErr)
		}
		out.Members = make([]ProjectMemberSummary, 0, len(members))
		for _, m := range members {
			out.Members = append(out.Members, ProjectMemberSummary{
				ID:          m.ID,
				Username:    m.Username,
				Name:        m.Name,
				AccessLevel: int(m.AccessLevel),
			})
		}
	}

	return nil, out, nil
}

// ============================================================================
// User
// ============================================================================

func (p *Plugin) handleGetMyGitLabUser(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, GetMyGitLabUserOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, GetMyGitLabUserOutput{}, err
	}

	user, err := p.GitlabClient.GetUserDetails(ctx, info, token)
	if err != nil {
		return nil, GetMyGitLabUserOutput{}, fmt.Errorf("failed to get GitLab user: %w", err)
	}

	return nil, GetMyGitLabUserOutput{
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.Name,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		WebURL:    user.WebURL,
	}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func issueToSummary(issue *internGitlab.Issue) IssueSummary {
	if issue == nil {
		return IssueSummary{}
	}
	s := IssueSummary{
		ID:          issue.ID,
		IID:         issue.IID,
		ProjectID:   issue.ProjectID,
		Title:       issue.Title,
		State:       issue.State,
		Description: issue.Description,
		WebURL:      issue.WebURL,
		Labels:      issue.Labels,
	}
	for _, a := range issue.Assignees {
		if a != nil {
			s.Assignees = append(s.Assignees, a.Username)
		}
	}
	if issue.Milestone != nil {
		s.Milestone = issue.Milestone.Title
	}
	if issue.CreatedAt != nil {
		s.CreatedAt = issue.CreatedAt.String()
	}
	if issue.UpdatedAt != nil {
		s.UpdatedAt = issue.UpdatedAt.String()
	}
	return s
}

func issuesToSummaries(issues []*internGitlab.Issue) []IssueSummary {
	out := make([]IssueSummary, 0, len(issues))
	for _, i := range issues {
		if i != nil {
			out = append(out, issueToSummary(i))
		}
	}
	return out
}

func mrToSummary(mr *internGitlab.MergeRequest) MergeRequestSummary {
	if mr == nil {
		return MergeRequestSummary{}
	}
	s := MergeRequestSummary{
		ID:           mr.ID,
		IID:          mr.IID,
		ProjectID:    mr.ProjectID,
		Title:        mr.Title,
		State:        mr.State,
		Description:  mr.Description,
		SourceBranch: mr.SourceBranch,
		TargetBranch: mr.TargetBranch,
		WebURL:       mr.WebURL,
		Labels:       mr.Labels,
	}
	if mr.Author != nil {
		s.Author = mr.Author.Username
	}
	for _, a := range mr.Assignees {
		if a != nil {
			s.Assignees = append(s.Assignees, a.Username)
		}
	}
	for _, r := range mr.Reviewers {
		if r != nil {
			s.Reviewers = append(s.Reviewers, r.Username)
		}
	}
	if mr.Milestone != nil {
		s.Milestone = mr.Milestone.Title
	}
	if mr.CreatedAt != nil {
		s.CreatedAt = mr.CreatedAt.String()
	}
	if mr.UpdatedAt != nil {
		s.UpdatedAt = mr.UpdatedAt.String()
	}
	return s
}

func mrsToSummaries(mrs []*internGitlab.MergeRequest) []MergeRequestSummary {
	out := make([]MergeRequestSummary, 0, len(mrs))
	for _, mr := range mrs {
		if mr != nil {
			out = append(out, mrToSummary(mr))
		}
	}
	return out
}

func projectToSummary(p *internGitlab.Project) ProjectSummary {
	if p == nil {
		return ProjectSummary{}
	}
	s := ProjectSummary{
		ID:                p.ID,
		Name:              p.Name,
		PathWithNamespace: p.PathWithNamespace,
		Description:       p.Description,
		WebURL:            p.WebURL,
		Visibility:        string(p.Visibility),
		DefaultBranch:     p.DefaultBranch,
	}
	return s
}

// noteWebURL builds a GitLab note permalink. Returns "" when the base URL or
// project path is missing so we don't emit a half-formed link to the agent.
func noteWebURL(baseURL, projectPath, kind string, parentIID, noteID int) string {
	if baseURL == "" || projectPath == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/-/%s/%d#note_%d", strings.TrimRight(baseURL, "/"), projectPath, kind, parentIID, noteID)
}

// splitProjectPath splits "namespace/project" into owner and repo.
// It also handles nested groups like "group/subgroup/project".
func splitProjectPath(projectPath string) (owner, repo string, err error) {
	if projectPath == "" {
		return "", "", fmt.Errorf("project_path must be in namespace/project format")
	}
	owner, repo = splitProjectPathParts(projectPath)
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("project_path %q must be in namespace/project format (e.g. mygroup/myproject)", projectPath)
	}
	return owner, repo, nil
}

// splitProjectPathParts splits the last segment off the path as the repo name,
// with everything before it as the owner/namespace. Returns ("", path) when no
// slash is present.
func splitProjectPathParts(projectPath string) (owner, repo string) {
	for i := len(projectPath) - 1; i >= 0; i-- {
		if projectPath[i] == '/' {
			return projectPath[:i], projectPath[i+1:]
		}
	}
	return "", projectPath
}
