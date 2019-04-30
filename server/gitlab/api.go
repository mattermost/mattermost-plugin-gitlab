package gitlab

import (
	"fmt"
	"net/http"

	internGitlab "github.com/manland/go-gitlab"
	"github.com/pkg/errors"
)

func (g *gitlab) GetProject(user *GitlabUserInfo, owner, repo string) (*internGitlab.Project, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	result, _, err := client.Projects.GetProject(fmt.Sprintf("%s/%s", owner, repo), &internGitlab.GetProjectOptions{})
	return result, err
}

func (g *gitlab) GetReviews(user *GitlabUserInfo) ([]*internGitlab.MergeRequest, error) {
	// TODO only for a group ?
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"
	scope := "all"

	//move notifs, merge-request, issues to package gitlab and reuse same for /todo and api http
	//add into client client.Groups.Issues()

	result, _, errRequest := client.MergeRequests.ListMergeRequests(&internGitlab.ListMergeRequestsOptions{
		AssigneeID: &user.GitlabUserId,
		State:      &opened,
		Scope:      &scope,
	})

	return result, errRequest
}

func (g *gitlab) GetYourPrs(user *GitlabUserInfo) ([]*internGitlab.MergeRequest, error) {
	// TODO only for a group ?
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"
	scope := "all"

	result, _, errRequest := client.MergeRequests.ListMergeRequests(&internGitlab.ListMergeRequestsOptions{
		AuthorID: &user.GitlabUserId,
		State:    &opened,
		Scope:    &scope,
	})

	return result, errRequest
}

func (g *gitlab) GetYourAssignments(user *GitlabUserInfo) ([]*internGitlab.Issue, error) {
	// TODO only for a group ?
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	opened := "opened"

	result, _, errRequest := client.Issues.ListIssues(&internGitlab.ListIssuesOptions{
		AssigneeID: &user.GitlabUserId,
		State:      &opened,
	})

	return result, errRequest
}

func (g *gitlab) GetUnreads(user *GitlabUserInfo) ([]*internGitlab.Todo, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return nil, err
	}

	result, _, err := client.Todos.ListTodos(&internGitlab.ListTodosOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "can't list todo in gitlab api")
	}
	notifications := []*internGitlab.Todo{}
	for _, todo := range result {
		// TODO check group
		// if p.checkGroup(todo.Project.NameWithNamespace) != nil {
		// 	continue
		// }
		notifications = append(notifications, todo)
	}

	return notifications, err
}

func (g *gitlab) Exist(user *GitlabUserInfo, owner string, repo string, enablePrivateRepo bool) (bool, error) {
	client, err := g.gitlabConnect(*user.Token)
	if err != nil {
		return false, err
	}

	if repo == "" {
		group, resp, err := client.Groups.GetGroup(owner)
		if group == nil && (err == nil || resp.StatusCode == http.StatusNotFound) {
			users, _, errListUser := client.Users.ListUsers(&internGitlab.ListUsersOptions{Username: &owner})
			if (users == nil || len(users) == 0) && errListUser == nil {
				return false, nil // not an error just not found group, owner
			} else if errListUser != nil {
				return false, errors.Wrapf(errListUser, "can't list user %s", owner)
			}
		} else if err != nil {
			return false, errors.Wrapf(err, "can't call api for group %s", owner)
		}
	} else {
		result, _, err := client.Projects.GetProject(owner+"/"+repo, &internGitlab.GetProjectOptions{})
		if result == nil || err != nil {
			if err != nil {
				return false, errors.Wrapf(err, "can't get project %s/%s", owner, repo)
			}
			return false, nil // not an error just not found project
		}
		if !enablePrivateRepo && result.Visibility != internGitlab.PublicVisibility {
			return false, errors.New("you can't add a private project on this mattermost instance")
		}
	}
	return true, nil
}
