package gitlab

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
)

func (g *myGitlab) GetProject(user *GitlabUserInfo, owner, repo string) (*gitlab.Project, error) {
	client, err := g.connect(*user.Token)
	if err != nil {
		return nil, err
	}

	result, _, err := client.Projects.GetProject(fmt.Sprintf("%s/%s", owner, repo), &gitlab.GetProjectOptions{})
	return result, err
}

func (g *myGitlab) GetReviews(user *GitlabUserInfo) ([]*gitlab.MergeRequest, error) {
	client, err := g.connect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"
	scope := "all"

	var result []*gitlab.MergeRequest
	var errRequest error

	if g.gitlabGroup == "" {
		result, _, errRequest = client.MergeRequests.ListMergeRequests(&gitlab.ListMergeRequestsOptions{
			AssigneeID: &user.GitlabUserId,
			State:      &opened,
			Scope:      &scope,
		})
	} else {
		result, _, errRequest = client.MergeRequests.ListGroupMergeRequests(g.gitlabGroup, &gitlab.ListGroupMergeRequestsOptions{
			AssigneeID: &user.GitlabUserId,
			State:      &opened,
			Scope:      &scope,
		})
	}

	return result, errRequest
}

func (g *myGitlab) GetYourPrs(user *GitlabUserInfo) ([]*gitlab.MergeRequest, error) {
	client, err := g.connect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"
	scope := "all"

	var result []*gitlab.MergeRequest
	var errRequest error

	if g.gitlabGroup == "" {
		result, _, errRequest = client.MergeRequests.ListMergeRequests(&gitlab.ListMergeRequestsOptions{
			AuthorID: &user.GitlabUserId,
			State:    &opened,
			Scope:    &scope,
		})
	} else {
		result, _, errRequest = client.MergeRequests.ListGroupMergeRequests(g.gitlabGroup, &gitlab.ListGroupMergeRequestsOptions{
			AuthorID: &user.GitlabUserId,
			State:    &opened,
			Scope:    &scope,
		})
	}

	return result, errRequest
}

func (g *myGitlab) GetYourAssignments(user *GitlabUserInfo) ([]*gitlab.Issue, error) {
	client, err := g.connect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"

	var result []*gitlab.Issue
	var errRequest error

	if g.gitlabGroup == "" {
		result, _, errRequest = client.Issues.ListIssues(&gitlab.ListIssuesOptions{
			AssigneeID: &user.GitlabUserId,
			State:      &opened,
		})
	} else {
		result, _, errRequest = client.Issues.ListGroupIssues(g.gitlabGroup, &gitlab.ListGroupIssuesOptions{
			AssigneeID: &user.GitlabUserId,
			State:      &opened,
		})
	}

	return result, errRequest
}

func (g *myGitlab) GetUnreads(user *GitlabUserInfo) ([]*gitlab.Todo, error) {
	client, err := g.connect(*user.Token)
	if err != nil {
		return nil, err
	}

	result, _, err := client.Todos.ListTodos(&gitlab.ListTodosOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "can't list todo in GitLab api")
	}
	notifications := []*gitlab.Todo{}
	for _, todo := range result {
		if g.checkGroup(todo.Project.NameWithNamespace) != nil {
			continue
		}
		notifications = append(notifications, todo)
	}

	return notifications, err
}

func (g *myGitlab) Exist(user *GitlabUserInfo, owner string, repo string, enablePrivateRepo bool) (bool, error) {
	client, err := g.connect(*user.Token)
	if err != nil {
		return false, err
	}

	publicVisibility := gitlab.PublicVisibility

	if repo == "" {
		group, resp, err := client.Groups.GetGroup(owner)
		if group == nil && (err == nil || resp.StatusCode == http.StatusNotFound) {
			users, _, errListUser := client.Users.ListUsers(&gitlab.ListUsersOptions{Username: &owner})
			if (users == nil || len(users) == 0) && errListUser == nil {
				return false, nil // not an error just not found group, owner
			} else if errListUser != nil {
				return false, errors.Wrapf(errListUser, "can't list user %s", owner)
			}
		} else if err != nil {
			return false, errors.Wrapf(err, "can't call api for group %s", owner)
		} else if !enablePrivateRepo && group.Visibility != &publicVisibility {
			return false, errors.New("you can't add a private group on this mattermost instance")
		}
	} else {
		result, _, err := client.Projects.GetProject(owner+"/"+repo, &gitlab.GetProjectOptions{})
		if result == nil || err != nil {
			if err != nil {
				return false, errors.Wrapf(err, "can't get project %s/%s", owner, repo)
			}
			return false, nil // not an error just not found project
		}
		if !enablePrivateRepo && result.Visibility != publicVisibility {
			return false, errors.New("you can't add a private project on this mattermost instance")
		}
	}
	return true, nil
}

//TODO test me
func (g *myGitlab) AddWebHooks(user *GitlabUserInfo, owner string, repo string, url string, token string) error {
	client, err := g.connectProxy(*user.Token)
	if err != nil {
		return err
	}

	if repo == "" {
		page := 0
		for {
			projects, resp, err := client.ListGroupProjects(owner, &gitlab.ListGroupProjectsOptions{
				ListOptions: gitlab.ListOptions{Page: page},
			})
			if err != nil {
				return errors.Wrap(err, "can't list projects of group "+owner)
			}

			for _, p := range projects {
				if err := g.addWebHook(client, p.NameWithNamespace, url, token); err != nil {
					//don't handle error and go to next project
				}
			}

			if resp.NextPage == 0 {
				return nil
			}

			page = resp.NextPage
		}
	}

	return g.addWebHook(client, fmt.Sprintf("%s/%s", owner, repo), url, token)
}

func (g *myGitlab) addWebHook(client goGitlabProxy, projectNameWithNamespace, url, token string) error {
	hooks, _, err := client.ListProjectHooks(projectNameWithNamespace, &gitlab.ListProjectHooksOptions{})
	if err != nil {
		return errors.Wrap(err, "can't list project hooks of "+projectNameWithNamespace)
	}

	//TODO all pages from response http
	for _, h := range hooks {
		if h.URL == url {
			return nil
		}
	}

	t := true
	f := false
	err = client.AddProjectHook(projectNameWithNamespace, &gitlab.AddProjectHookOptions{
		URL:                      &url,
		PushEvents:               &t,
		IssuesEvents:             &t,
		ConfidentialIssuesEvents: &t,
		MergeRequestsEvents:      &t,
		TagPushEvents:            &t,
		NoteEvents:               &t,
		JobEvents:                &t,
		PipelineEvents:           &t,
		WikiPageEvents:           &t,
		EnableSSLVerification:    &f,
		Token:                    &token,
	})
	if err != nil {
		return errors.Wrap(err, "can't add web hook for "+projectNameWithNamespace)
	}

	return nil
}
