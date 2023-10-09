package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/logger"

	"github.com/pkg/errors"
	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"
)

const (
	stateOpened = "opened"
	scopeAll    = "all"

	perPage = 20
)

type PRDetails struct {
	IID          int                           `json:"iid"`
	Status       *internGitlab.BuildStateValue `json:"status"`
	SHA          string                        `json:"sha"`
	NumApprovers int                           `json:"num_approvers"`
	ProjectID    int                           `json:"project_id"`
}

type MergeRequest struct {
	*internGitlab.MergeRequest
	LabelsWithDetails []*internGitlab.Label `json:"labels_with_details,omitempty"`
}

type Issue struct {
	*internGitlab.Issue
	LabelsWithDetails []*internGitlab.Label `json:"labels_with_details,omitempty"`
}

type LHSContent struct {
	PRs         []*MergeRequest      `json:"prs"`
	Reviews     []*MergeRequest      `json:"reviews"`
	Assignments []*Issue             `json:"assignments"`
	Unreads     []*internGitlab.Todo `json:"unreads"`
}

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

	hooks, resp, err := client.Groups.ListGroupHooks(owner, internGitlab.WithContext(ctx))
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

func (g *gitlab) GetLHSData(ctx context.Context, user *UserInfo, token *oauth2.Token) (*LHSContent, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return nil, err
	}

	grp, ctx := errgroup.WithContext(ctx)

	var reviews []*MergeRequest
	grp.Go(func() error {
		reviews, err = g.GetReviews(ctx, user, client)
		return err
	})

	var assignments []*Issue
	grp.Go(func() error {
		assignments, err = g.GetYourAssignments(ctx, user, client)
		return err
	})

	var mergeRequests []*MergeRequest
	grp.Go(func() error {
		mergeRequests, err = g.GetYourPrs(ctx, user, client)
		return err
	})

	var unreads []*internGitlab.Todo
	grp.Go(func() error {
		unreads, err = g.GetUnreads(ctx, user, client)
		return err
	})

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	return &LHSContent{
		Reviews:     reviews,
		PRs:         mergeRequests,
		Assignments: assignments,
		Unreads:     unreads,
	}, nil
}

func (g *gitlab) GetReviews(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*MergeRequest, error) {
	opened := stateOpened
	scope := scopeAll

	var mrs []*internGitlab.MergeRequest
	if g.gitlabGroup == "" {
		opt := &internGitlab.ListMergeRequestsOptions{
			ReviewerID:  internGitlab.ReviewerID(user.GitlabUserID),
			State:       &opened,
			Scope:       &scope,
			ListOptions: internGitlab.ListOptions{Page: 1, PerPage: perPage},
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
			ReviewerID:  internGitlab.ReviewerID(user.GitlabUserID),
			State:       &opened,
			Scope:       &scope,
			ListOptions: internGitlab.ListOptions{Page: 1, PerPage: perPage},
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

	mergeRequests := []*MergeRequest{}
	for _, mr := range mrs {
		if mr.Labels != nil {
			labelsWithDetails, err := g.GetLabelDetails(client, mr.ProjectID, mr.Labels)
			if err != nil {
				return nil, err
			}
			mergeRequest := &MergeRequest{
				MergeRequest:      mr,
				LabelsWithDetails: labelsWithDetails,
			}
			mergeRequests = append(mergeRequests, mergeRequest)
		}
	}

	return mergeRequests, nil
}

func (g *gitlab) GetYourPrs(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*MergeRequest, error) {
	opened := stateOpened
	scope := scopeAll
	var mrs []*internGitlab.MergeRequest
	if g.gitlabGroup == "" {
		opt := &internGitlab.ListMergeRequestsOptions{
			AuthorID:    &user.GitlabUserID,
			State:       &opened,
			Scope:       &scope,
			ListOptions: internGitlab.ListOptions{Page: 1, PerPage: perPage},
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
			AuthorID:    &user.GitlabUserID,
			State:       &opened,
			Scope:       &scope,
			ListOptions: internGitlab.ListOptions{Page: 1, PerPage: perPage},
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

	mergeRequests := []*MergeRequest{}
	for _, mr := range mrs {
		if mr.Labels != nil {
			labelsWithDetails, err := g.GetLabelDetails(client, mr.ProjectID, mr.Labels)
			if err != nil {
				return nil, err
			}
			mergeRequest := &MergeRequest{
				MergeRequest:      mr,
				LabelsWithDetails: labelsWithDetails,
			}
			mergeRequests = append(mergeRequests, mergeRequest)
		}
	}
	return mergeRequests, nil
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

func (g *gitlab) GetYourAssignments(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*Issue, error) {
	opened := stateOpened
	scope := scopeAll
	var issues []*internGitlab.Issue

	if g.gitlabGroup == "" {
		opt := &internGitlab.ListIssuesOptions{
			AssigneeID:  &user.GitlabUserID,
			State:       &opened,
			Scope:       &scope,
			ListOptions: internGitlab.ListOptions{Page: 1, PerPage: perPage},
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
			AssigneeID:  &user.GitlabUserID,
			State:       &opened,
			Scope:       &scope,
			ListOptions: internGitlab.ListOptions{Page: 1, PerPage: perPage},
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

	var result []*Issue
	for _, issue := range issues {
		if issue.Labels != nil {
			labelsWithDetails, err := g.GetLabelDetails(client, issue.ProjectID, issue.Labels)
			if err != nil {
				return nil, err
			}
			issue := &Issue{
				Issue:             issue,
				LabelsWithDetails: labelsWithDetails,
			}
			result = append(result, issue)
		}
	}
	return result, nil
}

func (g *gitlab) GetUnreads(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*internGitlab.Todo, error) {
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
