package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/sync/errgroup"
)

// NewProjectHook creates a webhook associated with a GitLab project
func (g *gitlab) NewProjectHook(user *GitlabUserInfo, projectID interface{}, projectHookOptions *internGitlab.AddProjectHookOptions) (*internGitlab.ProjectHook, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	///	url := "http://www.example.com"
	//	pushEvents := true
	//projectHookOptions := &internGitlab.AddProjectHookOptions{
	//URL:        &url,
	//PushEvents: &pushEvents,
	//}
	projectHook, _, err := client.Projects.AddProjectHook(projectID, projectHookOptions)
	if err != nil {
		return nil, err
	}

	return projectHook, nil
}

// GetGroupHooks gathers all the group level hooks for a GitLab group.
// group hooks are not available on Comunity Edition.
func (g *gitlab) GetGroupHooks(user *GitlabUserInfo, owner string) ([]*WebhookInfo, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	hooks, _, err := client.Groups.ListGroupHooks(owner)
	if err != nil {
		return nil, err
	}

	var webhooks []*WebhookInfo
	for _, hook := range hooks {
		webhooks = append(webhooks,
			&WebhookInfo{
				Scope:                    group,
				URL:                      hook.URL,
				ID:                       hook.ID,
				PushEvents:               hook.PushEvents,
				IssuesEvents:             hook.IssuesEvents,
				ConfidentialIssuesEvents: hook.ConfidentialIssuesEvents,
				MergeRequestsEvents:      hook.MergeRequestsEvents,
				TagPushEvents:            hook.TagPushEvents,
				NoteEvents:               hook.NoteEvents,
				JobEvents:                hook.JobEvents,
				PipelineEvents:           hook.PipelineEvents,
				WikiPageEvents:           hook.WikiPageEvents,
				EnableSslVerification:    hook.EnableSslVerification,
				CreatedAt:                hook.CreatedAt,
			},
		)
	}

	return webhooks, nil
}

func (w *WebhookInfo) Stringify() string {

	var formatedTriggers string
	if w.EnableSslVerification {
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

	return "\n" + w.URL + "\n" + formatedTriggers
}

func GetProjectHookInfo(hook *internGitlab.ProjectHook) *WebhookInfo {
	webhook := &WebhookInfo{
		Scope:                    project,
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
		EnableSslVerification:    hook.EnableSSLVerification,
		CreatedAt:                hook.CreatedAt,
	}
	return webhook
}

// GetProjectHooks gathers all the project level hooks from a single GitLab project.
func (g *gitlab) GetProjectHooks(user *GitlabUserInfo, owner string, repo string) ([]*internGitlab.ProjectHook, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	projectPath := fmt.Sprintf("%s/%s", owner, repo)
	projectHooks, _, err := client.Projects.ListProjectHooks(projectPath, nil)
	if err != nil {
		return nil, err
	}
	var webhooks []*WebhookInfo
	for _, hook := range projectHooks {
		webhooks = append(webhooks, GetProjectHookInfo(hook))
	}
	return webhooks, err
}

func (g *gitlab) GetProject(user *GitlabUserInfo, owner, repo string) (*internGitlab.Project, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	result, _, err := client.Projects.GetProject(fmt.Sprintf("%s/%s", owner, repo), &internGitlab.GetProjectOptions{})
	return result, err
}

func (g *gitlab) GetReviews(user *GitlabUserInfo) ([]*internGitlab.MergeRequest, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"
	scope := "all"

	var result []*internGitlab.MergeRequest
	var errRequest error

	if g.gitlabGroup == "" {
		result, _, errRequest = client.MergeRequests.ListMergeRequests(&internGitlab.ListMergeRequestsOptions{
			AssigneeID: &user.GitlabUserId,
			State:      &opened,
			Scope:      &scope,
		})
	} else {
		result, _, errRequest = client.MergeRequests.ListGroupMergeRequests(g.gitlabGroup, &internGitlab.ListGroupMergeRequestsOptions{
			AssigneeID: &user.GitlabUserId,
			State:      &opened,
			Scope:      &scope,
		})
	}

	return result, errRequest
}

func (g *gitlab) GetYourPrs(user *GitlabUserInfo) ([]*internGitlab.MergeRequest, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"
	scope := "all"

	var result []*internGitlab.MergeRequest
	var errRequest error

	if g.gitlabGroup == "" {
		result, _, errRequest = client.MergeRequests.ListMergeRequests(&internGitlab.ListMergeRequestsOptions{
			AuthorID: &user.GitlabUserId,
			State:    &opened,
			Scope:    &scope,
		})
	} else {
		result, _, errRequest = client.MergeRequests.ListGroupMergeRequests(g.gitlabGroup, &internGitlab.ListGroupMergeRequestsOptions{
			AuthorID: &user.GitlabUserId,
			State:    &opened,
			Scope:    &scope,
		})
	}

	return result, errRequest
}

func (g *gitlab) GetYourAssignments(user *GitlabUserInfo) ([]*internGitlab.Issue, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"

	var result []*internGitlab.Issue
	var errRequest error

	if g.gitlabGroup == "" {
		result, _, errRequest = client.Issues.ListIssues(&internGitlab.ListIssuesOptions{
			AssigneeID: &user.GitlabUserId,
			State:      &opened,
		})
	} else {
		result, _, errRequest = client.Issues.ListGroupIssues(g.gitlabGroup, &internGitlab.ListGroupIssuesOptions{
			AssigneeID: &user.GitlabUserId,
			State:      &opened,
		})
	}

	return result, errRequest
}

func (g *gitlab) GetUnreads(user *GitlabUserInfo) ([]*internGitlab.Todo, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	result, _, err := client.Todos.ListTodos(&internGitlab.ListTodosOptions{})
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

	return notifications, err
}

func (g *gitlab) ResolveNamespaceAndProject(
	userInfo *GitlabUserInfo,
	fullPath string,
	allowPrivate bool,
) (owner string, repo string, err error) {

	// Initialize client
	client, err := g.gitlabConnect(*userInfo.Token)
	if err != nil {
		return "", "", err
	}

	// Search for matching user, group and project concurrently
	//
	// Note: Calls to Users and Groups could be replaced with a single call to Namespaces.
	// However, Namespaces endpoint will not return Group visibility, so we will have to make additional call anyway.
	// Making this extra call here should reduce overall latency.
	var (
		user           *internGitlab.User
		group          *internGitlab.Group
		project        *internGitlab.Project
		ctx, ctxCancel = context.WithTimeout(context.Background(), DefaultRequestTimeout)
	)
	defer ctxCancel()
	errGroup, _ := errgroup.WithContext(ctx)
	if strings.Count(fullPath, "/") == 0 { // This request only makes sense for single path component
		errGroup.Go(func() error {
			users, _, err := client.Users.ListUsers(&internGitlab.ListUsersOptions{
				Username: &fullPath,
			})
			if err != nil {
				return fmt.Errorf("failed to search users by username: %w", err)
			}
			if len(users) == 1 {
				user = users[0]
			}
			return nil
		})
	}
	errGroup.Go(func() error {
		gr, response, err := client.Groups.GetGroup(fullPath)
		if err != nil && response != nil && response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("failed to retrieve group by path: %w", err)
		}
		group = gr
		return nil
	})
	errGroup.Go(func() error {
		p, response, err := client.Projects.GetProject(fullPath, nil, nil)
		if err != nil && response != nil && response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("failed to retrieve project by path: %w", err)
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
	} else if group != nil {
		if !allowPrivate && group.Visibility != nil && *group.Visibility != internGitlab.PublicVisibility {
			return "", "", fmt.Errorf(
				"you can't add a private group on this Mattermost instance: %w",
				ErrPrivateResource,
			)
		}
		return group.FullPath, "", nil
	} else if project != nil {
		if !allowPrivate && project.Visibility != internGitlab.PublicVisibility {
			return "", "", fmt.Errorf(
				"you can't add a private project on this Mattermost instance: %w",
				ErrPrivateResource,
			)
		}
		return project.Namespace.FullPath, project.Path, nil
	}
	return "", "", ErrNotFound
}
