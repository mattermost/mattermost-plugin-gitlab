// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"

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

func (p *Plugin) handleListMyAssignedIssues(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ListMyAssignedIssuesOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListMyAssignedIssuesOutput{}, err
	}

	client, err := p.GitlabClient.GitlabConnect(*token)
	if err != nil {
		return nil, ListMyAssignedIssuesOutput{}, fmt.Errorf("failed to connect to GitLab: %w", err)
	}

	issues, err := p.GitlabClient.GetYourAssignedIssues(ctx, info, client)
	if err != nil {
		return nil, ListMyAssignedIssuesOutput{}, fmt.Errorf("failed to list assigned issues: %w", err)
	}

	return nil, ListMyAssignedIssuesOutput{Issues: issuesToSummaries(issues)}, nil
}

func (p *Plugin) handleSearchIssues(ctx context.Context, _ *mcp.CallToolRequest, in SearchIssuesInput) (*mcp.CallToolResult, SearchIssuesOutput, error) {
	if in.Search == "" {
		return nil, SearchIssuesOutput{}, fmt.Errorf("search term is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, SearchIssuesOutput{}, err
	}

	issues, err := p.GitlabClient.SearchIssues(ctx, info, in.Search, token)
	if err != nil {
		return nil, SearchIssuesOutput{}, fmt.Errorf("failed to search issues: %w", err)
	}

	return nil, SearchIssuesOutput{Issues: issuesToSummaries(issues)}, nil
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

	req := &gitlab.IssueRequest{
		Title:       in.Title,
		Description: in.Description,
		Assignees:   in.AssigneeIDs,
		Milestone:   in.MilestoneID,
		Labels:      internGitlab.LabelOptions(in.Labels),
	}

	owner, repo := splitProjectPathParts(in.ProjectPath)
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

func (p *Plugin) handleAddIssueComment(ctx context.Context, _ *mcp.CallToolRequest, in AddIssueCommentInput) (*mcp.CallToolResult, AddIssueCommentOutput, error) {
	if in.ProjectPath == "" {
		return nil, AddIssueCommentOutput{}, fmt.Errorf("project_path is required")
	}
	if in.IssueIID <= 0 {
		return nil, AddIssueCommentOutput{}, fmt.Errorf("issue_iid must be a positive integer")
	}
	if in.Body == "" {
		return nil, AddIssueCommentOutput{}, fmt.Errorf("body is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, AddIssueCommentOutput{}, err
	}

	note, err := p.GitlabClient.AddIssueNote(ctx, info, token, in.ProjectPath, in.IssueIID, in.Body)
	if err != nil {
		return nil, AddIssueCommentOutput{}, fmt.Errorf("failed to add issue comment: %w", err)
	}

	return nil, AddIssueCommentOutput{NoteID: note.ID, Body: note.Body}, nil
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

func (p *Plugin) handleListMyAssignedMRs(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ListMyAssignedMergeRequestsOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListMyAssignedMergeRequestsOutput{}, err
	}

	client, err := p.GitlabClient.GitlabConnect(*token)
	if err != nil {
		return nil, ListMyAssignedMergeRequestsOutput{}, fmt.Errorf("failed to connect to GitLab: %w", err)
	}

	mrs, err := p.GitlabClient.GetYourAssignedPrs(ctx, info, client)
	if err != nil {
		return nil, ListMyAssignedMergeRequestsOutput{}, fmt.Errorf("failed to list assigned merge requests: %w", err)
	}

	return nil, ListMyAssignedMergeRequestsOutput{MergeRequests: mrsToSummaries(mrs)}, nil
}

func (p *Plugin) handleListMyReviewRequests(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ListMyReviewRequestsOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListMyReviewRequestsOutput{}, err
	}

	client, err := p.GitlabClient.GitlabConnect(*token)
	if err != nil {
		return nil, ListMyReviewRequestsOutput{}, fmt.Errorf("failed to connect to GitLab: %w", err)
	}

	mrs, err := p.GitlabClient.GetReviews(ctx, info, client)
	if err != nil {
		return nil, ListMyReviewRequestsOutput{}, fmt.Errorf("failed to list review requests: %w", err)
	}

	return nil, ListMyReviewRequestsOutput{MergeRequests: mrsToSummaries(mrs)}, nil
}

func (p *Plugin) handleSearchMergeRequests(ctx context.Context, _ *mcp.CallToolRequest, in SearchMergeRequestsInput) (*mcp.CallToolResult, SearchMergeRequestsOutput, error) {
	if in.Search == "" {
		return nil, SearchMergeRequestsOutput{}, fmt.Errorf("search term is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, SearchMergeRequestsOutput{}, err
	}

	mrs, err := p.GitlabClient.SearchMergeRequests(ctx, info, token, in.Search)
	if err != nil {
		return nil, SearchMergeRequestsOutput{}, fmt.Errorf("failed to search merge requests: %w", err)
	}

	return nil, SearchMergeRequestsOutput{MergeRequests: mrsToSummaries(mrs)}, nil
}

func (p *Plugin) handleCreateMergeRequest(ctx context.Context, _ *mcp.CallToolRequest, in CreateMergeRequestInput) (*mcp.CallToolResult, CreateMergeRequestOutput, error) {
	if in.ProjectPath == "" {
		return nil, CreateMergeRequestOutput{}, fmt.Errorf("project_path is required")
	}
	if in.Title == "" {
		return nil, CreateMergeRequestOutput{}, fmt.Errorf("title is required")
	}
	if in.SourceBranch == "" {
		return nil, CreateMergeRequestOutput{}, fmt.Errorf("source_branch is required")
	}
	if in.TargetBranch == "" {
		return nil, CreateMergeRequestOutput{}, fmt.Errorf("target_branch is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, CreateMergeRequestOutput{}, err
	}

	opts := &gitlab.CreateMergeRequestOptions{
		Title:        in.Title,
		Description:  in.Description,
		SourceBranch: in.SourceBranch,
		TargetBranch: in.TargetBranch,
		AssigneeIDs:  in.AssigneeIDs,
		ReviewerIDs:  in.ReviewerIDs,
		Labels:       internGitlab.LabelOptions(in.Labels),
		MilestoneID:  in.MilestoneID,
	}

	mr, err := p.GitlabClient.CreateMergeRequest(ctx, info, token, in.ProjectPath, opts)
	if err != nil {
		return nil, CreateMergeRequestOutput{}, fmt.Errorf("failed to create merge request: %w", err)
	}

	return nil, CreateMergeRequestOutput{MergeRequest: mrToSummary(mr)}, nil
}

func (p *Plugin) handleAddMergeRequestComment(ctx context.Context, _ *mcp.CallToolRequest, in AddMergeRequestCommentInput) (*mcp.CallToolResult, AddMergeRequestCommentOutput, error) {
	if in.ProjectPath == "" {
		return nil, AddMergeRequestCommentOutput{}, fmt.Errorf("project_path is required")
	}
	if in.MergeRequestID <= 0 {
		return nil, AddMergeRequestCommentOutput{}, fmt.Errorf("merge_request_iid must be a positive integer")
	}
	if in.Body == "" {
		return nil, AddMergeRequestCommentOutput{}, fmt.Errorf("body is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, AddMergeRequestCommentOutput{}, err
	}

	note, err := p.GitlabClient.AddMergeRequestNote(ctx, info, token, in.ProjectPath, in.MergeRequestID, in.Body)
	if err != nil {
		return nil, AddMergeRequestCommentOutput{}, fmt.Errorf("failed to add merge request comment: %w", err)
	}

	return nil, AddMergeRequestCommentOutput{NoteID: note.ID, Body: note.Body}, nil
}

// ============================================================================
// Projects
// ============================================================================

func (p *Plugin) handleListMyProjects(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ListMyProjectsOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListMyProjectsOutput{}, err
	}

	projects, err := p.GitlabClient.GetYourProjects(ctx, info, token)
	if err != nil {
		return nil, ListMyProjectsOutput{}, fmt.Errorf("failed to list projects: %w", err)
	}

	summaries := make([]ProjectSummary, 0, len(projects))
	for _, project := range projects {
		summaries = append(summaries, projectToSummary(project))
	}

	return nil, ListMyProjectsOutput{Projects: summaries}, nil
}

func (p *Plugin) handleGetProject(ctx context.Context, _ *mcp.CallToolRequest, in GetProjectInput) (*mcp.CallToolResult, GetProjectOutput, error) {
	if in.ProjectPath == "" {
		return nil, GetProjectOutput{}, fmt.Errorf("project_path is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, GetProjectOutput{}, err
	}

	owner, repo := splitProjectPathParts(in.ProjectPath)
	project, err := p.GitlabClient.GetProject(ctx, info, token, owner, repo)
	if err != nil {
		return nil, GetProjectOutput{}, fmt.Errorf("failed to get project: %w", err)
	}

	return nil, GetProjectOutput{Project: projectToSummary(project)}, nil
}

// ============================================================================
// Pipelines
// ============================================================================

func (p *Plugin) handleRunPipeline(ctx context.Context, _ *mcp.CallToolRequest, in RunPipelineInput) (*mcp.CallToolResult, RunPipelineOutput, error) {
	if in.ProjectPath == "" {
		return nil, RunPipelineOutput{}, fmt.Errorf("project_path is required")
	}
	if in.Ref == "" {
		return nil, RunPipelineOutput{}, fmt.Errorf("ref is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, RunPipelineOutput{}, err
	}

	pipeline, err := p.GitlabClient.TriggerProjectPipeline(info, token, in.ProjectPath, in.Ref)
	if err != nil {
		return nil, RunPipelineOutput{}, fmt.Errorf("failed to run pipeline: %w", err)
	}

	return nil, RunPipelineOutput{Pipeline: PipelineSummary{
		ID:     pipeline.PipelineID,
		Status: pipeline.Status,
		Ref:    pipeline.Ref,
		SHA:    pipeline.SHA,
		WebURL: pipeline.WebURL,
	}}, nil
}

func (p *Plugin) handleListProjectPipelines(ctx context.Context, _ *mcp.CallToolRequest, in ListProjectPipelinesInput) (*mcp.CallToolResult, ListProjectPipelinesOutput, error) {
	if in.ProjectPath == "" {
		return nil, ListProjectPipelinesOutput{}, fmt.Errorf("project_path is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListProjectPipelinesOutput{}, err
	}

	pipelines, err := p.GitlabClient.ListProjectPipelines(ctx, info, token, in.ProjectPath, in.Ref, in.Status, in.Page, in.PerPage)
	if err != nil {
		return nil, ListProjectPipelinesOutput{}, fmt.Errorf("failed to list pipelines: %w", err)
	}

	summaries := make([]PipelineSummary, 0, len(pipelines))
	for _, pl := range pipelines {
		summaries = append(summaries, pipelineInfoToSummary(pl))
	}

	return nil, ListProjectPipelinesOutput{Pipelines: summaries}, nil
}

// ============================================================================
// Todos
// ============================================================================

func (p *Plugin) handleGetMyTodos(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, GetMyTodosOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, GetMyTodosOutput{}, err
	}

	client, err := p.GitlabClient.GitlabConnect(*token)
	if err != nil {
		return nil, GetMyTodosOutput{}, fmt.Errorf("failed to connect to GitLab: %w", err)
	}

	todos, err := p.GitlabClient.GetToDoList(ctx, info, client)
	if err != nil {
		return nil, GetMyTodosOutput{}, fmt.Errorf("failed to get todos: %w", err)
	}

	return nil, GetMyTodosOutput{Todos: todosToSummaries(todos)}, nil
}

// ============================================================================
// Dashboard
// ============================================================================

func (p *Plugin) handleGetGitLabDashboard(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, LHSDataOutput, error) {
	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, LHSDataOutput{}, err
	}

	lhs, err := p.GitlabClient.GetLHSData(ctx, info, token)
	if err != nil {
		return nil, LHSDataOutput{}, fmt.Errorf("failed to get GitLab dashboard: %w", err)
	}

	return nil, LHSDataOutput{
		AssignedMergeRequests: mrsToSummaries(lhs.AssignedPRs),
		ReviewRequests:        mrsToSummaries(lhs.Reviews),
		AssignedIssues:        issuesToSummaries(lhs.AssignedIssues),
		Todos:                 todosToSummaries(lhs.Todos),
	}, nil
}

// ============================================================================
// Labels and Milestones
// ============================================================================

func (p *Plugin) handleListProjectLabels(ctx context.Context, _ *mcp.CallToolRequest, in ListProjectLabelsInput) (*mcp.CallToolResult, ListProjectLabelsOutput, error) {
	if in.ProjectPath == "" {
		return nil, ListProjectLabelsOutput{}, fmt.Errorf("project_path is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListProjectLabelsOutput{}, err
	}

	labels, err := p.GitlabClient.GetLabels(ctx, info, in.ProjectPath, token)
	if err != nil {
		return nil, ListProjectLabelsOutput{}, fmt.Errorf("failed to list labels: %w", err)
	}

	summaries := make([]LabelSummary, 0, len(labels))
	for _, l := range labels {
		summaries = append(summaries, LabelSummary{
			ID:          l.ID,
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		})
	}

	return nil, ListProjectLabelsOutput{Labels: summaries}, nil
}

func (p *Plugin) handleListProjectMilestones(ctx context.Context, _ *mcp.CallToolRequest, in ListProjectMilestonesInput) (*mcp.CallToolResult, ListProjectMilestonesOutput, error) {
	if in.ProjectPath == "" {
		return nil, ListProjectMilestonesOutput{}, fmt.Errorf("project_path is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListProjectMilestonesOutput{}, err
	}

	milestones, err := p.GitlabClient.GetMilestones(ctx, info, in.ProjectPath, token)
	if err != nil {
		return nil, ListProjectMilestonesOutput{}, fmt.Errorf("failed to list milestones: %w", err)
	}

	summaries := make([]MilestoneSummary, 0, len(milestones))
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
		summaries = append(summaries, ms)
	}

	return nil, ListProjectMilestonesOutput{Milestones: summaries}, nil
}

// ============================================================================
// User / Metadata
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

func (p *Plugin) handleListProjectMembers(ctx context.Context, _ *mcp.CallToolRequest, in ListProjectMembersInput) (*mcp.CallToolResult, ListProjectMembersOutput, error) {
	if in.ProjectPath == "" {
		return nil, ListProjectMembersOutput{}, fmt.Errorf("project_path is required")
	}

	info, token, err := p.resolveCaller(ctx)
	if err != nil {
		return nil, ListProjectMembersOutput{}, err
	}

	members, err := p.GitlabClient.GetProjectMembers(ctx, info, in.ProjectPath, token)
	if err != nil {
		return nil, ListProjectMembersOutput{}, fmt.Errorf("failed to list project members: %w", err)
	}

	summaries := make([]ProjectMemberSummary, 0, len(members))
	for _, m := range members {
		summaries = append(summaries, ProjectMemberSummary{
			ID:          m.ID,
			Username:    m.Username,
			Name:        m.Name,
			AccessLevel: int(m.AccessLevel),
		})
	}

	return nil, ListProjectMembersOutput{Members: summaries}, nil
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

func pipelineInfoToSummary(pl *internGitlab.PipelineInfo) PipelineSummary {
	if pl == nil {
		return PipelineSummary{}
	}
	s := PipelineSummary{
		ID:     pl.ID,
		Status: pl.Status,
		Ref:    pl.Ref,
		SHA:    pl.SHA,
		WebURL: pl.WebURL,
	}
	if pl.CreatedAt != nil {
		s.CreatedAt = pl.CreatedAt.String()
	}
	if pl.UpdatedAt != nil {
		s.UpdatedAt = pl.UpdatedAt.String()
	}
	return s
}

func todoToSummary(todo *internGitlab.Todo) TodoSummary {
	if todo == nil {
		return TodoSummary{}
	}
	s := TodoSummary{
		ID:         todo.ID,
		ActionName: string(todo.ActionName),
	}
	if todo.Target != nil {
		s.TargetType = string(todo.TargetType)
		s.TargetTitle = todo.Target.Title
		s.WebURL = todo.Target.WebURL
	}
	if todo.Project != nil {
		s.ProjectPath = todo.Project.PathWithNamespace
	}
	if todo.CreatedAt != nil {
		s.CreatedAt = todo.CreatedAt.String()
	}
	return s
}

func todosToSummaries(todos []*internGitlab.Todo) []TodoSummary {
	out := make([]TodoSummary, 0, len(todos))
	for _, t := range todos {
		if t != nil {
			out = append(out, todoToSummary(t))
		}
	}
	return out
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
