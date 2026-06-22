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

// mcpConnect validates the OAuth token and returns a connected GitLab client.
// Centralising the nil/empty-token guard keeps every MCP method from panicking
// on a missing token.
func (g *gitlab) mcpConnect(token *oauth2.Token) (*internGitlab.Client, error) {
	if token == nil || token.AccessToken == "" {
		return nil, fmt.Errorf("missing OAuth token")
	}
	return g.GitlabConnect(*token)
}

func (g *gitlab) UpdateIssue(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID string, issueIID int, opts *UpdateIssueOptions) (*internGitlab.Issue, error) {
	if opts == nil {
		return nil, fmt.Errorf("update issue options are required")
	}
	client, err := g.mcpConnect(token)
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
	client, err := g.mcpConnect(token)
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
	client, err := g.mcpConnect(token)
	if err != nil {
		return nil, err
	}

	var (
		result []*internGitlab.MergeRequest
		resp   *internGitlab.Response
	)
	if g.gitlabGroup == "" {
		result, resp, err = client.Search.MergeRequests(search, &internGitlab.SearchOptions{}, internGitlab.WithContext(ctx))
	} else {
		result, resp, err = client.Search.MergeRequestsByGroup(g.gitlabGroup, search, &internGitlab.SearchOptions{}, internGitlab.WithContext(ctx))
	}
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to search merge requests: %w", err)
	}

	return result, nil
}

func (g *gitlab) AddMergeRequestNote(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID string, mrIID int, body string) (*internGitlab.Note, error) {
	client, err := g.mcpConnect(token)
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

// ListAssignedIssues returns the open issues assigned to the calling user. It
// wraps the client-based helper so MCP callers never hold a raw client.
func (g *gitlab) ListAssignedIssues(ctx context.Context, user *UserInfo, token *oauth2.Token) ([]*internGitlab.Issue, error) {
	client, err := g.mcpConnect(token)
	if err != nil {
		return nil, err
	}
	return g.GetYourAssignedIssues(ctx, user, client)
}

// ListAssignedMergeRequests returns the open merge requests assigned to the
// calling user.
func (g *gitlab) ListAssignedMergeRequests(ctx context.Context, user *UserInfo, token *oauth2.Token) ([]*internGitlab.MergeRequest, error) {
	client, err := g.mcpConnect(token)
	if err != nil {
		return nil, err
	}
	return g.GetYourAssignedPrs(ctx, user, client)
}

// ListReviewRequests returns the open merge requests awaiting the calling
// user's review.
func (g *gitlab) ListReviewRequests(ctx context.Context, user *UserInfo, token *oauth2.Token) ([]*internGitlab.MergeRequest, error) {
	client, err := g.mcpConnect(token)
	if err != nil {
		return nil, err
	}
	return g.GetReviews(ctx, user, client)
}
