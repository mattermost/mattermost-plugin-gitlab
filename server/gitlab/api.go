package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-plugin-api/experimental/bot/logger"

	"github.com/pkg/errors"
	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/sync/errgroup"
)

const (
	stateOpened = "opened"
	scopeAll    = "all"
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

// NewGroupHook creates a webhook associated with a GitLab group
func (g *gitlab) NewGroupHook(ctx context.Context, user *UserInfo, groupName string, webhookOptions *AddWebhookOptions) (*WebhookInfo, error) {
	client, err := g.gitlabConnect(*user.Token)
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
func (g *gitlab) NewProjectHook(ctx context.Context, user *UserInfo, projectID interface{}, webhookOptions *AddWebhookOptions) (*WebhookInfo, error) {
	client, err := g.gitlabConnect(*user.Token)
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
func (g *gitlab) GetGroupHooks(ctx context.Context, user *UserInfo, owner string) ([]*WebhookInfo, error) {
	client, err := g.gitlabConnect(*user.Token)
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
func (g *gitlab) GetProjectHooks(ctx context.Context, user *UserInfo, owner string, repo string) ([]*WebhookInfo, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	projectPath := fmt.Sprintf("%s/%s", owner, repo)
	projectHooks, resp, err := client.Projects.ListProjectHooks(projectPath, nil, internGitlab.WithContext(ctx))
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}
	var webhooks []*WebhookInfo
	for _, hook := range projectHooks {
		webhooks = append(webhooks, getProjectHookInfo(hook))
	}
	return webhooks, nil
}

func (g *gitlab) GetProject(ctx context.Context, user *UserInfo, owner, repo string) (*internGitlab.Project, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	result, resp, err := client.Projects.GetProject(fmt.Sprintf("%s/%s", owner, repo),
		&internGitlab.GetProjectOptions{},
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

func (g *gitlab) GetReviews(ctx context.Context, user *UserInfo) ([]*MergeRequest, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := stateOpened
	scope := scopeAll

	var result []*internGitlab.MergeRequest

	if g.gitlabGroup == "" {
		result, _, err = client.MergeRequests.ListMergeRequests(&internGitlab.ListMergeRequestsOptions{
			ReviewerID: internGitlab.ReviewerID(user.GitlabUserID),
			State:      &opened,
			Scope:      &scope,
		},
			internGitlab.WithContext(ctx),
		)
	} else {
		result, _, err = client.MergeRequests.ListGroupMergeRequests(g.gitlabGroup, &internGitlab.ListGroupMergeRequestsOptions{
			ReviewerID: internGitlab.ReviewerID(user.GitlabUserID),
			State:      &opened,
			Scope:      &scope,
		},
			internGitlab.WithContext(ctx),
		)
	}

	mergeRequests := []*MergeRequest{}
	for _, res := range result {
		if res.Labels != nil {
			var labelsWithDetails []*internGitlab.Label
			labelsWithDetails, err = g.GetLabelDetails(client, res.ProjectID, res.Labels)
			if err != nil {
				return nil, err
			}
			mergeRequest := &MergeRequest{
				MergeRequest:      res,
				LabelsWithDetails: labelsWithDetails,
			}
			mergeRequests = append(mergeRequests, mergeRequest)
		}
	}

	return mergeRequests, err
}

func (g *gitlab) GetYourPrs(ctx context.Context, user *UserInfo) ([]*MergeRequest, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := stateOpened
	scope := scopeAll

	var result []*internGitlab.MergeRequest
	var resp *internGitlab.Response

	if g.gitlabGroup == "" {
		result, resp, err = client.MergeRequests.ListMergeRequests(&internGitlab.ListMergeRequestsOptions{
			AuthorID: &user.GitlabUserID,
			State:    &opened,
			Scope:    &scope,
		},
			internGitlab.WithContext(ctx),
		)
	} else {
		result, resp, err = client.MergeRequests.ListGroupMergeRequests(g.gitlabGroup, &internGitlab.ListGroupMergeRequestsOptions{
			AuthorID: &user.GitlabUserID,
			State:    &opened,
			Scope:    &scope,
		},
			internGitlab.WithContext(ctx),
		)
	}
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	mergeRequests := []*MergeRequest{}
	for _, res := range result {
		if res.Labels != nil {
			var labelsWithDetails []*internGitlab.Label
			labelsWithDetails, err = g.GetLabelDetails(client, res.ProjectID, res.Labels)
			if err != nil {
				return nil, err
			}
			mergeRequest := &MergeRequest{
				MergeRequest:      res,
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

func (g *gitlab) GetYourPrDetails(ctx context.Context, log logger.Logger, user *UserInfo, prList []*PRDetails) ([]*PRDetails, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	result := []*PRDetails{}
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

func (g *gitlab) GetYourAssignments(ctx context.Context, user *UserInfo) ([]*Issue, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}
	opened := stateOpened
	scope := scopeAll

	var result []*internGitlab.Issue
	var resp *internGitlab.Response

	if g.gitlabGroup == "" {
		result, resp, err = client.Issues.ListIssues(&internGitlab.ListIssuesOptions{
			AssigneeID: &user.GitlabUserID,
			State:      &opened,
			Scope:      &scope,
		},
			internGitlab.WithContext(ctx),
		)
	} else {
		result, resp, err = client.Issues.ListGroupIssues(g.gitlabGroup, &internGitlab.ListGroupIssuesOptions{
			AssigneeID: &user.GitlabUserID,
			State:      &opened,
			Scope:      &scope,
		},
			internGitlab.WithContext(ctx),
		)
	}
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, err
	}

	issues := []*Issue{}
	for _, res := range result {
		if res.Labels != nil {
			var labelsWithDetails []*internGitlab.Label
			labelsWithDetails, err = g.GetLabelDetails(client, res.ProjectID, res.Labels)
			if err != nil {
				return nil, err
			}
			issue := &Issue{
				Issue:             res,
				LabelsWithDetails: labelsWithDetails,
			}
			issues = append(issues, issue)
		}
	}
	return issues, nil
}

func (g *gitlab) GetUnreads(ctx context.Context, user *UserInfo) ([]*internGitlab.Todo, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	result, resp, err := client.Todos.ListTodos(
		&internGitlab.ListTodosOptions{},
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return nil, respErr
	}
	if err != nil {
		return nil, errors.Wrap(err, "can't list todo in GitLab api")
	}
	notifications := make([]*internGitlab.Todo, 0, len(result))
	for _, todo := range result {
		if g.checkGroup(strings.TrimSuffix(todo.Project.PathWithNamespace, "/"+todo.Project.Path)) != nil {
			continue
		}
		notifications = append(notifications, todo)
	}

	return notifications, nil
}

func (g *gitlab) ResolveNamespaceAndProject(
	ctx context.Context,
	userInfo *UserInfo,
	fullPath string,
	allowPrivate bool,
) (owner string, repo string, err error) {
	// Initialize client
	client, err := g.gitlabConnect(*userInfo.Token)
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
