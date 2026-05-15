// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gitlab

import (
	"context"
	"fmt"

	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

// UpdateIssueOptions contains the fields that can be updated on an existing issue.
type UpdateIssueOptions struct {
	Title       *string
	Description *string
	// StateEvent is "close" or "reopen".
	StateEvent  *string
	AssigneeIDs *[]int
	Labels      *internGitlab.LabelOptions
	MilestoneID *int
}

// CreateMergeRequestOptions contains the required and optional fields for creating
// a new merge request.
type CreateMergeRequestOptions struct {
	Title        string
	Description  string
	SourceBranch string
	TargetBranch string
	AssigneeIDs  []int
	ReviewerIDs  []int
	Labels       internGitlab.LabelOptions
	MilestoneID  *int
}

func (g *gitlab) UpdateIssue(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID string, issueIID int, opts *UpdateIssueOptions) (*internGitlab.Issue, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}
	if err = g.ensureProjectInAllowedGroup(ctx, client, projectID); err != nil {
		return nil, err
	}

	updateOpts := &internGitlab.UpdateIssueOptions{
		Title:       opts.Title,
		Description: opts.Description,
		StateEvent:  opts.StateEvent,
		AssigneeIDs: opts.AssigneeIDs,
		Labels:      opts.Labels,
		MilestoneID: opts.MilestoneID,
	}

	issue, resp, err := client.Issues.UpdateIssue(projectID, issueIID, updateOpts, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update issue: %w", err)
	}

	return issue, nil
}

func (g *gitlab) AddIssueNote(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID string, issueIID int, body string) (*internGitlab.Note, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}
	if err = g.ensureProjectInAllowedGroup(ctx, client, projectID); err != nil {
		return nil, err
	}

	note, resp, err := client.Notes.CreateIssueNote(
		projectID,
		issueIID,
		&internGitlab.CreateIssueNoteOptions{Body: &body},
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to add issue comment: %w", err)
	}

	return note, nil
}

func (g *gitlab) SearchMergeRequests(ctx context.Context, user *UserInfo, token *oauth2.Token, search string) ([]*internGitlab.MergeRequest, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	if g.gitlabGroup == "" {
		result, resp, err := client.Search.MergeRequests(search, &internGitlab.SearchOptions{}, internGitlab.WithContext(ctx))
		if respErr := checkResponse(resp); respErr != nil {
			return nil, respErr
		}
		if err != nil {
			return nil, fmt.Errorf("failed to search merge requests: %w", err)
		}
		return result, nil
	}

	result, resp, err := client.Search.MergeRequestsByGroup(g.gitlabGroup, search, &internGitlab.SearchOptions{}, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to search merge requests: %w", err)
	}

	return result, nil
}

func (g *gitlab) CreateMergeRequest(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID string, opts *CreateMergeRequestOptions) (*internGitlab.MergeRequest, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}
	if err = g.ensureProjectInAllowedGroup(ctx, client, projectID); err != nil {
		return nil, err
	}

	createOpts := &internGitlab.CreateMergeRequestOptions{
		Title:        &opts.Title,
		Description:  &opts.Description,
		SourceBranch: &opts.SourceBranch,
		TargetBranch: &opts.TargetBranch,
	}
	if len(opts.AssigneeIDs) > 0 {
		createOpts.AssigneeIDs = &opts.AssigneeIDs
	}
	if len(opts.ReviewerIDs) > 0 {
		createOpts.ReviewerIDs = &opts.ReviewerIDs
	}
	if len(opts.Labels) > 0 {
		createOpts.Labels = &opts.Labels
	}
	if opts.MilestoneID != nil {
		createOpts.MilestoneID = opts.MilestoneID
	}

	mr, resp, err := client.MergeRequests.CreateMergeRequest(projectID, createOpts, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create merge request: %w", err)
	}

	return mr, nil
}

func (g *gitlab) AddMergeRequestNote(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID string, mrIID int, body string) (*internGitlab.Note, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}
	if err = g.ensureProjectInAllowedGroup(ctx, client, projectID); err != nil {
		return nil, err
	}

	note, resp, err := client.Notes.CreateMergeRequestNote(
		projectID,
		mrIID,
		&internGitlab.CreateMergeRequestNoteOptions{Body: &body},
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to add merge request comment: %w", err)
	}

	return note, nil
}

func (g *gitlab) ListProjectPipelines(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID string, ref string, status string, page int, perPage int) ([]*internGitlab.PipelineInfo, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}
	if err = g.ensureProjectInAllowedGroup(ctx, client, projectID); err != nil {
		return nil, err
	}

	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	opts := &internGitlab.ListProjectPipelinesOptions{
		ListOptions: internGitlab.ListOptions{Page: page, PerPage: perPage},
	}
	if ref != "" {
		opts.Ref = &ref
	}
	if status != "" {
		bs := internGitlab.BuildStateValue(status)
		opts.Status = &bs
	}

	pipelines, resp, err := client.Pipelines.ListProjectPipelines(projectID, opts, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}

	return pipelines, nil
}
