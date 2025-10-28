// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/logger"

	"github.com/pkg/errors"
	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

const (
	stateOpened   = "opened"
	scopeAll      = "all"
	getLabelsTrue = true

	perPage = 20
)

type IssueRequest struct {
	ID          int                       `json:"id"`
	IID         int                       `json:"iid"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Milestone   int                       `json:"milestone"`
	ProjectID   int                       `json:"project_id"`
	Assignees   []int                     `json:"assignees"`
	Labels      internGitlab.LabelOptions `json:"labels"`
	PostID      string                    `json:"post_id"`
	ChannelID   string                    `json:"channel_id"`
	Comment     string                    `json:"comment"`
	WebURL      string                    `json:"web_url"`
}

type PRDetails struct {
	IID          int                           `json:"iid"`
	Status       *internGitlab.BuildStateValue `json:"status"`
	SHA          string                        `json:"sha"`
	NumApprovers int                           `json:"num_approvers"`
	ProjectID    int                           `json:"project_id"`
}

type MergeRequest struct {
	*internGitlab.MergeRequest
	LabelsWithDetails []*internGitlab.Label `json:"label_details,omitempty"`
}

type Issue struct {
	*internGitlab.Issue
	LabelsWithDetails []*internGitlab.Label `json:"label_details,omitempty"`
}

type LHSContent struct {
	AssignedPRs    []*internGitlab.MergeRequest `json:"yourAssignedPrs"`
	Reviews        []*internGitlab.MergeRequest `json:"reviews"`
	AssignedIssues []*internGitlab.Issue        `json:"yourAssignedIssues"`
	Todos          []*internGitlab.Todo         `json:"todos"`
}

// Pagination helper for both ListProjects and ListGroupProjects
type listPageFunc func(page, perPage int) ([]*internGitlab.Project, *internGitlab.Response, error)

// NewGroupHook creates a webhook associated with a GitLab group
func (g *gitlab) NewGroupHook(ctx context.Context, user *UserInfo, token *oauth2.Token, groupName string, webhookOptions *AddWebhookOptions) (*WebhookInfo, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	group, resp, err := client.Groups.GetGroup(groupName, nil, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	groupHookOptions := internGitlab.AddGroupHookOptions{
		URL:                      &webhookOptions.URL,
		ConfidentialNoteEvents:   &webhookOptions.ConfidentialNoteEvents,
		PushEvents:               &webhookOptions.PushEvents,
		IssuesEvents:             &webhookOptions.IssuesEvents,
		ConfidentialIssuesEvents: &webhookOptions.ConfidentialIssuesEvents,
		MergeRequestsEvents:      &webhookOptions.MergeRequestsEvents,
		TagPushEvents:            &webhookOptions.TagPushEvents,
		NoteEvents:               &webhookOptions.NoteEvents,
		JobEvents:                &webhookOptions.JobEvents,
		PipelineEvents:           &webhookOptions.PipelineEvents,
		WikiPageEvents:           &webhookOptions.WikiPageEvents,
		DeploymentEvents:         &webhookOptions.DeploymentEvents,
		ReleasesEvents:           &webhookOptions.ReleaseEvents,
		EnableSSLVerification:    &webhookOptions.EnableSSLVerification,
		Token:                    &webhookOptions.Token,
	}

	groupHook, resp, err := client.Groups.AddGroupHook(group.ID, &groupHookOptions, internGitlab.WithContext(ctx))
	if err != nil {
		if resp.StatusCode == http.StatusForbidden {
			return nil, ErrForbidden
		}

		return nil, err
	}

	groupHookInfo := getGroupHookInfo(groupHook)

	return groupHookInfo, nil
}

// NewProjectHook creates a webhook associated with a GitLab project
func (g *gitlab) NewProjectHook(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID interface{}, webhookOptions *AddWebhookOptions) (*WebhookInfo, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	projectHookOptions := internGitlab.AddProjectHookOptions{
		URL:                      &webhookOptions.URL,
		ConfidentialNoteEvents:   &webhookOptions.ConfidentialNoteEvents,
		PushEvents:               &webhookOptions.PushEvents,
		IssuesEvents:             &webhookOptions.IssuesEvents,
		ConfidentialIssuesEvents: &webhookOptions.ConfidentialIssuesEvents,
		MergeRequestsEvents:      &webhookOptions.MergeRequestsEvents,
		TagPushEvents:            &webhookOptions.TagPushEvents,
		NoteEvents:               &webhookOptions.NoteEvents,
		JobEvents:                &webhookOptions.JobEvents,
		PipelineEvents:           &webhookOptions.PipelineEvents,
		WikiPageEvents:           &webhookOptions.WikiPageEvents,
		DeploymentEvents:         &webhookOptions.DeploymentEvents,
		ReleasesEvents:           &webhookOptions.ReleaseEvents,
		EnableSSLVerification:    &webhookOptions.EnableSSLVerification,
		Token:                    &webhookOptions.Token,
	}

	projectHook, resp, err := client.Projects.AddProjectHook(projectID, &projectHookOptions, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	projectHookInfo := getProjectHookInfo(projectHook)

	return projectHookInfo, nil
}

// GetGroupHooks gathers all the group level hooks for a GitLab group.
func (g *gitlab) GetGroupHooks(ctx context.Context, user *UserInfo, token *oauth2.Token, owner string) ([]*WebhookInfo, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	hooks, resp, err := client.Groups.ListGroupHooks(owner, nil, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	var webhooks []*WebhookInfo
	for _, hook := range hooks {
		webhooks = append(webhooks, getGroupHookInfo(hook))
	}

	return webhooks, nil
}

// String produces a multiline bulleted string for displaying webhook information.
func (w *WebhookInfo) String() string {
	var formatedTriggers string
	if w.EnableSSLVerification {
		formatedTriggers += "SSL Verification Enabled\n"
	}

	formatedTriggers += "Triggers:\n"
	if w.PushEvents {
		formatedTriggers += "* Push Events\n"
	}
	if w.TagPushEvents {
		formatedTriggers += "* Tag Push Events\n"
	}
	if w.NoteEvents {
		formatedTriggers += "* Comments\n"
	}
	if w.ConfidentialNoteEvents {
		formatedTriggers += "* Confidential Comments\n"
	}
	if w.IssuesEvents {
		formatedTriggers += "* Issues Events\n"
	}
	if w.ConfidentialIssuesEvents {
		formatedTriggers += "* Confidential Issues Events\n"
	}
	if w.MergeRequestsEvents {
		formatedTriggers += "* Merge Request Events\n"
	}
	if w.JobEvents {
		formatedTriggers += "* Job Events\n"
	}
	if w.PipelineEvents {
		formatedTriggers += "* Pipeline Events\n"
	}
	if w.WikiPageEvents {
		formatedTriggers += "* Wiki Page Events\n"
	}
	if w.ReleaseEvents {
		formatedTriggers += "* Release Events\n"
	}
	if w.DeploymentEvents {
		formatedTriggers += "* Deployment Events\n"
	}

	return "\n\n`" + w.URL + "`\n" + formatedTriggers
}

func getProjectHookInfo(hook *internGitlab.ProjectHook) *WebhookInfo {
	webhook := &WebhookInfo{
		Scope:                    Project,
		URL:                      hook.URL,
		ID:                       hook.ID,
		ConfidentialNoteEvents:   hook.ConfidentialNoteEvents,
		PushEvents:               hook.PushEvents,
		IssuesEvents:             hook.IssuesEvents,
		ConfidentialIssuesEvents: hook.ConfidentialIssuesEvents,
		MergeRequestsEvents:      hook.MergeRequestsEvents,
		TagPushEvents:            hook.TagPushEvents,
		NoteEvents:               hook.NoteEvents,
		JobEvents:                hook.JobEvents,
		PipelineEvents:           hook.PipelineEvents,
		WikiPageEvents:           hook.WikiPageEvents,
		DeploymentEvents:         hook.DeploymentEvents,
		ReleaseEvents:            hook.ReleasesEvents,
		EnableSSLVerification:    hook.EnableSSLVerification,
		CreatedAt:                hook.CreatedAt,
	}
	return webhook
}

func getGroupHookInfo(hook *internGitlab.GroupHook) *WebhookInfo {
	webhook := &WebhookInfo{
		Scope:                    Project,
		URL:                      hook.URL,
		ID:                       hook.ID,
		ConfidentialNoteEvents:   hook.ConfidentialNoteEvents,
		PushEvents:               hook.PushEvents,
		IssuesEvents:             hook.IssuesEvents,
		ConfidentialIssuesEvents: hook.ConfidentialIssuesEvents,
		MergeRequestsEvents:      hook.MergeRequestsEvents,
		TagPushEvents:            hook.TagPushEvents,
		NoteEvents:               hook.NoteEvents,
		JobEvents:                hook.JobEvents,
		PipelineEvents:           hook.PipelineEvents,
		WikiPageEvents:           hook.WikiPageEvents,
		DeploymentEvents:         hook.DeploymentEvents,
		ReleaseEvents:            hook.ReleasesEvents,
		EnableSSLVerification:    hook.EnableSSLVerification,
		CreatedAt:                hook.CreatedAt,
	}
	return webhook
}

// GetProjectHooks gathers all the project level hooks from a single GitLab project.
func (g *gitlab) GetProjectHooks(ctx context.Context, user *UserInfo, token *oauth2.Token, owner string, repo string) ([]*WebhookInfo, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	opt := &internGitlab.ListProjectHooksOptions{
		PerPage: perPage,
		Page:    1,
	}

	var projectHooks []*internGitlab.ProjectHook
	var hooks []*internGitlab.ProjectHook
	var resp *internGitlab.Response
	projectPath := fmt.Sprintf("%s/%s", owner, repo)
	for {
		hooks, resp, err = client.Projects.ListProjectHooks(projectPath, opt)
		if err != nil {
			return nil, err
		}
		projectHooks = append(projectHooks, hooks...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	var webhooks []*WebhookInfo
	for _, hook := range projectHooks {
		webhooks = append(webhooks, getProjectHookInfo(hook))
	}
	return webhooks, nil
}

func (g *gitlab) GetProject(ctx context.Context, user *UserInfo, token *oauth2.Token, owner, repo string) (*internGitlab.Project, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	project, resp, err := client.Projects.GetProject(fmt.Sprintf("%s/%s", owner, repo),
		&internGitlab.GetProjectOptions{},
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (g *gitlab) GetGroup(ctx context.Context, user *UserInfo, token *oauth2.Token, group, subgroup string) (*internGitlab.Group, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	groupData, resp, err := client.Groups.GetGroup(group,
		&internGitlab.GetGroupOptions{},
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	return groupData, nil
}

func (g *gitlab) GetLHSData(ctx context.Context, user *UserInfo, token *oauth2.Token) (*LHSContent, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	grp, ctx := errgroup.WithContext(ctx)

	var reviews []*internGitlab.MergeRequest
	grp.Go(func() error {
		reviews, err = g.GetReviews(ctx, user, client)
		return err
	})

	var issues []*internGitlab.Issue
	grp.Go(func() error {
		issues, err = g.GetYourAssignedIssues(ctx, user, client)
		return err
	})

	var mergeRequests []*internGitlab.MergeRequest
	grp.Go(func() error {
		mergeRequests, err = g.GetYourAssignedPrs(ctx, user, client)
		return err
	})

	var todos []*internGitlab.Todo
	grp.Go(func() error {
		todos, err = g.GetToDoList(ctx, user, client)
		return err
	})

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	return &LHSContent{
		Reviews:        reviews,
		AssignedPRs:    mergeRequests,
		AssignedIssues: issues,
		Todos:          todos,
	}, nil
}

func (g *gitlab) GetReviews(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*internGitlab.MergeRequest, error) {
	opened := stateOpened
	scope := scopeAll
	getLabelDetails := getLabelsTrue
	var mrs []*internGitlab.MergeRequest
	if g.gitlabGroup == "" {
		opt := &internGitlab.ListMergeRequestsOptions{
			ReviewerID:        internGitlab.ReviewerID(user.GitlabUserID),
			State:             &opened,
			Scope:             &scope,
			WithLabelsDetails: &getLabelDetails,
			ListOptions:       internGitlab.ListOptions{Page: 1, PerPage: perPage},
		}
		for {
			current, resp, err := client.MergeRequests.ListMergeRequests(opt)
			if err != nil {
				return nil, err
			}
			mrs = append(mrs, current...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	} else {
		opt := &internGitlab.ListGroupMergeRequestsOptions{
			ReviewerID:        internGitlab.ReviewerID(user.GitlabUserID),
			State:             &opened,
			Scope:             &scope,
			WithLabelsDetails: &getLabelDetails,
			ListOptions:       internGitlab.ListOptions{Page: 1, PerPage: perPage},
		}
		for {
			current, resp, err := client.MergeRequests.ListGroupMergeRequests(g.gitlabGroup, opt)
			if err != nil {
				return nil, err
			}
			mrs = append(mrs, current...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}

	return mrs, nil
}

func (g *gitlab) GetYourAssignedPrs(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*internGitlab.MergeRequest, error) {
	opened := stateOpened
	scope := scopeAll
	getLabelDetails := getLabelsTrue
	var mrs []*internGitlab.MergeRequest
	if g.gitlabGroup == "" {
		opt := &internGitlab.ListMergeRequestsOptions{
			AssigneeID:        internGitlab.AssigneeID(user.GitlabUserID),
			State:             &opened,
			Scope:             &scope,
			WithLabelsDetails: &getLabelDetails,
			ListOptions:       internGitlab.ListOptions{Page: 1, PerPage: perPage},
		}
		for {
			current, resp, err := client.MergeRequests.ListMergeRequests(opt)
			if err != nil {
				return nil, err
			}
			mrs = append(mrs, current...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	} else {
		opt := &internGitlab.ListGroupMergeRequestsOptions{
			AssigneeID:        internGitlab.AssigneeID(user.GitlabUserID),
			State:             &opened,
			Scope:             &scope,
			WithLabelsDetails: &getLabelDetails,
			ListOptions:       internGitlab.ListOptions{Page: 1, PerPage: perPage},
		}
		for {
			current, resp, err := client.MergeRequests.ListGroupMergeRequests(g.gitlabGroup, opt)
			if err != nil {
				return nil, err
			}
			mrs = append(mrs, current...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}

	return mrs, nil
}

func (g *gitlab) GetLabelDetails(client *internGitlab.Client, pid int, labels internGitlab.Labels) ([]*internGitlab.Label, error) {
	// Get list of all labels.
	labelList, resp, err := client.Labels.ListLabels(pid, nil)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't get list of labels in GitLab")
	}

	allLabels := map[string]*internGitlab.Label{}
	for _, label := range labelList {
		allLabels[label.Name] = label
	}

	labelsWithDetails := []*internGitlab.Label{}
	for _, label := range labels {
		if allLabels[label] == nil {
			return nil, errors.Wrap(err, "can't get label in GitLab api")
		}
		labelsWithDetails = append(labelsWithDetails, allLabels[label])
	}

	return labelsWithDetails, nil
}

func (g *gitlab) GetYourPrDetails(ctx context.Context, log logger.Logger, user *UserInfo, token *oauth2.Token, prList []*PRDetails) ([]*PRDetails, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	var result []*PRDetails
	var wg sync.WaitGroup
	for _, pr := range prList {
		wg.Add(1)
		go func(pid, iid int, sha string) {
			defer wg.Done()
			res := g.fetchYourPrDetails(ctx, log, client, pid, iid, sha)
			if res != nil {
				result = append(result, res)
			}
		}(pr.ProjectID, pr.IID, pr.SHA)
	}
	wg.Wait()
	return result, nil
}

func (g *gitlab) fetchYourPrDetails(c context.Context, log logger.Logger, client *internGitlab.Client, pid, iid int, sha string) *PRDetails {
	var commitDetails *internGitlab.Commit
	var approvalDetails *internGitlab.MergeRequestApprovals
	var err error
	var resp *internGitlab.Response
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		commitDetails, resp, err = client.Commits.GetCommit(pid, sha, internGitlab.WithContext(c))
		if respErr := checkResponse(resp); respErr != nil {
			log.WithError(respErr).Warnf("Failed to fetch commit details for PR with project_id %d", pid)
			return
		}
		if err != nil {
			log.WithError(err).Warnf("Failed to fetch commit details for PR with project_id %d", pid)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		approvalDetails, resp, err = client.MergeRequestApprovals.GetConfiguration(pid, iid, internGitlab.WithContext(c))
		if respErr := checkResponse(resp); respErr != nil {
			log.WithError(respErr).Warnf("Failed to fetch approval details for PR with project_id %d", pid)
			return
		}
		if err != nil {
			log.WithError(err).Warnf("Failed to fetch approval details for PR with project_id %d", pid)
			return
		}
	}()

	wg.Wait()
	if commitDetails != nil && approvalDetails != nil {
		return &PRDetails{
			ProjectID:    pid,
			SHA:          sha,
			Status:       commitDetails.Status,
			NumApprovers: len(approvalDetails.ApprovedBy),
		}
	}
	return nil
}

func (g *gitlab) GetYourAssignedIssues(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*internGitlab.Issue, error) {
	opened := stateOpened
	scope := scopeAll
	var issues []*internGitlab.Issue
	getLabelDetails := getLabelsTrue
	if g.gitlabGroup == "" {
		opt := &internGitlab.ListIssuesOptions{
			AssigneeID:       internGitlab.AssigneeID(user.GitlabUserID),
			State:            &opened,
			Scope:            &scope,
			WithLabelDetails: &getLabelDetails,
			ListOptions:      internGitlab.ListOptions{Page: 1, PerPage: perPage},
		}
		for {
			current, resp, err := client.Issues.ListIssues(opt)
			if err != nil {
				return nil, err
			}
			issues = append(issues, current...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	} else {
		opt := &internGitlab.ListGroupIssuesOptions{
			AssigneeID:       internGitlab.AssigneeID(user.GitlabUserID),
			State:            &opened,
			Scope:            &scope,
			WithLabelDetails: &getLabelDetails,
			ListOptions:      internGitlab.ListOptions{Page: 1, PerPage: perPage},
		}
		for {
			current, resp, err := client.Issues.ListGroupIssues(g.gitlabGroup, opt)
			if err != nil {
				return nil, err
			}
			issues = append(issues, current...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}
	return issues, nil
}

func (g *gitlab) GetToDoList(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*internGitlab.Todo, error) {
	var todos []*internGitlab.Todo

	opt := &internGitlab.ListTodosOptions{
		ListOptions: internGitlab.ListOptions{Page: 1, PerPage: perPage},
	}
	for {
		current, resp, err := client.Todos.ListTodos(opt)
		if err != nil {
			return nil, errors.Wrap(err, "can't list todo in GitLab api")
		}
		todos = append(todos, current...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	notifications := make([]*internGitlab.Todo, 0, len(todos))
	for _, todo := range todos {
		if todo == nil {
			continue
		}

		if todo.Project != nil && g.checkGroup(strings.TrimSuffix(todo.Project.PathWithNamespace, "/"+todo.Project.Path)) != nil {
			continue
		}
		notifications = append(notifications, todo)
	}

	return notifications, nil
}

// Helper function for pagination
func paginateAll(ctx context.Context, perPage int, projects []*internGitlab.Project, listFn listPageFunc) ([]*internGitlab.Project, error) {
	page := 1

	for {
		pageProjects, resp, err := listFn(page, perPage)
		if err != nil {
			if respErr := checkResponse(resp); respErr != nil {
				return nil, respErr
			}
			return nil, err
		}

		projects = append(projects, pageProjects...)

		// resp.CurrentPage >= resp.TotalPages: we have fetched the last page
		// resp.NextPage == 0: GitLab did not set a next-page number (no further pages)
		if resp.CurrentPage >= resp.TotalPages || resp.NextPage == 0 {
			break
		}

		page = resp.NextPage
	}

	return projects, nil
}

func (g *gitlab) GetYourProjects(ctx context.Context, user *UserInfo, token *oauth2.Token) ([]*internGitlab.Project, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	const guestLevel = internGitlab.AccessLevelValue(10) // Guest = 10
	const perPage = 100

	var projects []*internGitlab.Project

	if g.gitlabGroup == "" {
		// ─── “No Group” branch: list all projects you belong to
		opts := &internGitlab.ListProjectsOptions{
			Membership:        model.NewPointer(true),
			WithIssuesEnabled: model.NewPointer(true),
			MinAccessLevel:    model.NewPointer(guestLevel),
			ListOptions: internGitlab.ListOptions{
				Page:    1,
				PerPage: perPage,
			},
		}

		listFn := func(page, perPage int) ([]*internGitlab.Project, *internGitlab.Response, error) {
			opts.ListOptions.Page = page
			opts.ListOptions.PerPage = perPage
			return client.Projects.ListProjects(opts, internGitlab.WithContext(ctx))
		}

		return paginateAll(ctx, perPage, projects, listFn)
	}
	// ─── “With Group” branch: list all projects in that group you have access to
	opts := &internGitlab.ListGroupProjectsOptions{
		WithIssuesEnabled: model.NewPointer(true),
		MinAccessLevel:    model.NewPointer(guestLevel),
		ListOptions: internGitlab.ListOptions{
			Page:    1,
			PerPage: perPage,
		},
	}

	listFn := func(page, perPage int) ([]*internGitlab.Project, *internGitlab.Response, error) {
		opts.ListOptions.Page = page
		opts.ListOptions.PerPage = perPage
		return client.Groups.ListGroupProjects(
			g.gitlabGroup,
			opts,
			internGitlab.WithContext(ctx),
		)
	}

	return paginateAll(ctx, perPage, projects, listFn)
}

func (g *gitlab) GetLabels(ctx context.Context, user *UserInfo, projectID string, token *oauth2.Token) ([]*internGitlab.Label, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitLab: %w", err)
	}

	opts := &internGitlab.ListLabelsOptions{
		ListOptions: internGitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var labels []*internGitlab.Label
	for {
		page, resp, err := client.Labels.ListLabels(projectID, opts, internGitlab.WithContext(ctx))
		if respErr := checkResponse(resp); respErr != nil {
			return nil, fmt.Errorf("failed to list labels for project %s: %w", projectID, respErr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list labels for project %s: %w", projectID, err)
		}
		labels = append(labels, page...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return labels, nil
}

// topLevelGroupFromProject returns the top-level group name (first path segment)
// for a project under a group namespace. Returns empty string if not a group.
func topLevelGroupFromProject(p *internGitlab.Project) string {
	if p == nil || p.Namespace == nil || !strings.EqualFold(p.Namespace.Kind, "group") || p.Namespace.FullPath == "" {
		return ""
	}
	first, _, _ := strings.Cut(p.Namespace.FullPath, "/")
	return first
}

func convertGroupMilestones(gms []*internGitlab.GroupMilestone) []*internGitlab.Milestone {
	out := make([]*internGitlab.Milestone, 0, len(gms))
	for _, gm := range gms {
		if gm == nil {
			continue
		}
		out = append(out, &internGitlab.Milestone{
			ID:          gm.ID,
			IID:         gm.IID,
			Title:       gm.Title,
			Description: gm.Description,
			State:       gm.State,
			DueDate:     gm.DueDate,
			StartDate:   gm.StartDate,
			CreatedAt:   gm.CreatedAt,
			UpdatedAt:   gm.UpdatedAt,
			// GroupMilestone does not expose WebURL; leave empty.
			WebURL: "",
		})
	}
	return out
}

// listAllProjectMilestones paginates project milestones for the given project.
func listAllProjectMilestones(ctx context.Context, client *internGitlab.Client, projectID string) ([]*internGitlab.Milestone, error) {
	popts := &internGitlab.ListMilestonesOptions{
		ListOptions: internGitlab.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var all []*internGitlab.Milestone
	for {
		page, resp, err := client.Milestones.ListMilestones(projectID, popts, internGitlab.WithContext(ctx))
		if respErr := checkResponse(resp); respErr != nil {
			return nil, fmt.Errorf("failed to list milestones for project %s: %w", projectID, respErr)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list milestones for project %s: %w", projectID, err)
		}
		all = append(all, page...)
		if resp.NextPage == 0 {
			break
		}
		popts.Page = resp.NextPage
	}
	return all, nil
}

func (g *gitlab) GetMilestones(ctx context.Context, user *UserInfo, projectID string, token *oauth2.Token) ([]*internGitlab.Milestone, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitLab: %w", err)
	}

	// Resolve project to find its namespace and top-level group
	project, resp, err := client.Projects.GetProject(projectID, nil, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", projectID, respErr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", projectID, err)
	}

	topGroup := topLevelGroupFromProject(project)

	if topGroup != "" {
		opts := &internGitlab.ListGroupMilestonesOptions{
			ListOptions: internGitlab.ListOptions{
				Page:    1,
				PerPage: 100,
			},
		}

		var (
			all  []*internGitlab.Milestone
			page []*internGitlab.GroupMilestone
			resp *internGitlab.Response
		)
		for {
			page, resp, err = client.GroupMilestones.ListGroupMilestones(topGroup, opts, internGitlab.WithContext(ctx))
			if respErr := checkResponse(resp); respErr != nil {
				return nil, fmt.Errorf("failed to list group milestones for %s: %w", topGroup, respErr)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to list group milestones for %s: %w", topGroup, err)
			}
			all = append(all, convertGroupMilestones(page)...)
			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}
		return all, nil
	}

	all, err := listAllProjectMilestones(ctx, client, projectID)
	if err != nil {
		return nil, err
	}
	return all, nil
}

func (g *gitlab) GetProjectMembers(ctx context.Context, user *UserInfo, projectID string, token *oauth2.Token) ([]*internGitlab.ProjectMember, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}
	result, resp, err := client.ProjectMembers.ListAllProjectMembers(
		projectID,
		nil,
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (g *gitlab) CreateIssue(ctx context.Context, user *UserInfo, issue *IssueRequest, token *oauth2.Token) (*internGitlab.Issue, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	result, resp, err := client.Issues.CreateIssue(
		issue.ProjectID,
		&internGitlab.CreateIssueOptions{
			Title:       &issue.Title,
			Description: &issue.Description,
			MilestoneID: &issue.Milestone,
			AssigneeIDs: &issue.Assignees,
			Labels:      &issue.Labels,
		},
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't create issue in GitLab")
	}

	return result, nil
}

func (g *gitlab) AttachCommentToIssue(ctx context.Context, user *UserInfo, issue *IssueRequest, permalink, commentUsername string, token *oauth2.Token) (*internGitlab.Note, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	issueComment := fmt.Sprintf("*@%s attached a* [message](%s) *from @%s*\n\n%s", user.GitlabUsername, permalink, commentUsername, issue.Comment)

	result, resp, err := client.Notes.CreateIssueNote(
		issue.ProjectID,
		issue.IID,
		&internGitlab.CreateIssueNoteOptions{
			Body: &issueComment,
		},
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't create issue comment in GitLab api")
	}

	return result, nil
}

func (g *gitlab) SearchIssues(ctx context.Context, user *UserInfo, search string, token *oauth2.Token) ([]*internGitlab.Issue, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	var issues []*internGitlab.Issue
	if g.gitlabGroup == "" {
		result, resp, err := client.Search.Issues(
			search,
			&internGitlab.SearchOptions{},
			internGitlab.WithContext(ctx),
		)
		if respErr := checkResponse(resp); respErr != nil {
			return nil, respErr
		}
		if err != nil {
			return nil, errors.Wrap(err, "can't search issues in GitLab api")
		}

		issues = append(issues, result...)
	} else {
		result, resp, err := client.Search.IssuesByGroup(
			g.gitlabGroup,
			search,
			&internGitlab.SearchOptions{},
			internGitlab.WithContext(ctx),
		)
		if respErr := checkResponse(resp); respErr != nil {
			return nil, respErr
		}
		if err != nil {
			return nil, errors.Wrap(err, "can't search issues in GitLab api")
		}

		issues = append(issues, result...)
	}

	return issues, nil
}

func (g *gitlab) ResolveNamespaceAndProject(
	ctx context.Context,
	userInfo *UserInfo,
	token *oauth2.Token,
	fullPath string,
	allowPrivate bool,
) (owner string, repo string, err error) {
	// Initialize client
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return "", "", err
	}

	fullPath = strings.TrimPrefix(fullPath, g.gitlabURL)
	fullPath = strings.Trim(fullPath, "/")

	// Search for matching user, group and project concurrently
	//
	// Note: Calls to Users and Groups could be replaced with a single call to Namespaces.
	// However, Namespaces endpoint will not return Group visibility, so we will have to make additional call anyway.
	// Making this extra call here should reduce overall latency.
	var (
		user    *internGitlab.User
		group   *internGitlab.Group
		project *internGitlab.Project
	)

	errGroup, ctx := errgroup.WithContext(ctx)
	if strings.Count(fullPath, "/") == 0 { // This request only makes sense for single path component
		errGroup.Go(func() error {
			users, resp, err := client.Users.ListUsers(&internGitlab.ListUsersOptions{
				Username: &fullPath,
			},
				internGitlab.WithContext(ctx),
			)
			if respErr := checkResponse(resp); respErr != nil {
				return respErr
			}
			if err != nil {
				return errors.Wrap(err, "failed to search users by username")
			}
			if len(users) == 1 {
				user = users[0]
			}
			return nil
		})
	}
	errGroup.Go(func() error {
		gr, resp, err := client.Groups.GetGroup(fullPath, nil, internGitlab.WithContext(ctx))
		if err != nil && resp != nil && resp.StatusCode != http.StatusNotFound {
			return errors.Wrap(err, "failed to retrieve group by path")
		}
		group = gr
		return nil
	})
	errGroup.Go(func() error {
		p, resp, err := client.Projects.GetProject(fullPath, nil, internGitlab.WithContext(ctx))
		if err != nil && resp != nil && resp.StatusCode != http.StatusNotFound {
			return errors.Wrap(err, "failed to retrieve project by path")
		}
		project = p
		return nil
	})
	if err := errGroup.Wait(); err != nil {
		return "", "", err
	}

	// Decide what to return
	if user != nil {
		return user.Username, "", nil
	}

	if group != nil {
		if !allowPrivate && group.Visibility != internGitlab.PublicVisibility {
			return "", "", errors.Wrap(ErrPrivateResource,
				"You can't add a private group on this Mattermost instance. Please enable private repositories in the System Console.",
			)
		}
		return group.FullPath, "", nil
	}

	if project != nil {
		if !allowPrivate && project.Visibility != internGitlab.PublicVisibility {
			return "", "", errors.Wrap(ErrPrivateResource,
				"You can't add a private group on this Mattermost instance. Please enable private repositories in the System Console.",
			)
		}
		return project.Namespace.FullPath, project.Path, nil
	}
	return "", "", ErrNotFound
}

func (g *gitlab) GetIssueByID(ctx context.Context, user *UserInfo, owner, repo string, issueID int, token *oauth2.Token) (*Issue, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}
	projectPath := fmt.Sprintf("%s/%s", owner, repo)
	issue, resp, err := client.Issues.GetIssue(projectPath, issueID)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't get issue in GitLab api")
	}

	gitlabIssue := &Issue{
		Issue: issue,
	}
	if len(issue.Labels) > 0 {
		labelsWithDetails, err := g.GetLabelDetails(client, issue.ProjectID, issue.Labels)
		if err != nil {
			return nil, err
		}
		gitlabIssue.LabelsWithDetails = labelsWithDetails
	}

	return gitlabIssue, nil
}

func (g *gitlab) GetMergeRequestByID(ctx context.Context, user *UserInfo, owner, repo string, mergeRequestID int, token *oauth2.Token) (*MergeRequest, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}
	projectPath := fmt.Sprintf("%s/%s", owner, repo)
	mergeRequest, resp, err := client.MergeRequests.GetMergeRequest(projectPath, mergeRequestID, nil)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't get merge request in GitLab api")
	}

	gitlabMergeRequest := &MergeRequest{
		MergeRequest: mergeRequest,
	}
	if len(mergeRequest.Labels) > 0 {
		labelsWithDetails, err := g.GetLabelDetails(client, mergeRequest.ProjectID, mergeRequest.Labels)
		if err != nil {
			return nil, err
		}
		gitlabMergeRequest.LabelsWithDetails = labelsWithDetails
	}

	return gitlabMergeRequest, nil
}

// TriggerProjectPipeline runs a pipeline in a specific project
func (g *gitlab) TriggerProjectPipeline(userInfo *UserInfo, token *oauth2.Token, projectID string, ref string) (*PipelineInfo, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return &PipelineInfo{}, err
	}
	pipeline, _, err := client.Pipelines.CreatePipeline(projectID, &internGitlab.CreatePipelineOptions{
		Ref: &ref,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to run the pipeline")
	}

	return &PipelineInfo{
		PipelineID: pipeline.ID,
		Status:     pipeline.Status,
		Ref:        pipeline.Ref,
		WebURL:     pipeline.WebURL,
		SHA:        pipeline.SHA,
		User:       pipeline.User.Name,
	}, err
}
