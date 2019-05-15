package gitlab

import (
	"context"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

//Gitlab is a client to call GitLab api see New() to build one
type Gitlab interface {
	GetCurrentUser(userID string, token oauth2.Token) (*GitlabUserInfo, error)
	GetUserDetails(user *GitlabUserInfo) (*gitlab.User, error)
	GetProject(user *GitlabUserInfo, owner, repo string) (*gitlab.Project, error)
	GetReviews(user *GitlabUserInfo) ([]*gitlab.MergeRequest, error)
	GetYourPrs(user *GitlabUserInfo) ([]*gitlab.MergeRequest, error)
	GetYourAssignments(user *GitlabUserInfo) ([]*gitlab.Issue, error)
	GetUnreads(user *GitlabUserInfo) ([]*gitlab.Todo, error)
	Exist(user *GitlabUserInfo, owner string, repo string, enablePrivateRepo bool) (bool, error)
	AddWebHooks(user *GitlabUserInfo, owner string, repo string, url string, token string) error
}

type myGitlab struct {
	gitlabGroup  string
	checkGroup   func(projectNameWithGroup string) error
	connect      func(token oauth2.Token) (*gitlab.Client, error)
	connectProxy func(token oauth2.Token) (goGitlabProxy, error)
}

//New return a client to call GitLab API
func New(gitlabURL string, gitlabGroup string, checkGroup func(projectNameWithGroup string) error) Gitlab {
	return &myGitlab{
		gitlabGroup: gitlabGroup,
		checkGroup:  checkGroup,
		connect: func(token oauth2.Token) (*gitlab.Client, error) {
			ctx := context.Background()
			ts := oauth2.StaticTokenSource(&token)
			tc := oauth2.NewClient(ctx, ts)

			client := gitlab.NewOAuthClient(tc, token.AccessToken)
			if err := client.SetBaseURL(gitlabURL); err != nil {
				return nil, errors.Wrap(err, "can't set base url to GitLab client lib")
			}
			return client, nil
		},
		connectProxy: func(token oauth2.Token) (goGitlabProxy, error) {
			ctx := context.Background()
			ts := oauth2.StaticTokenSource(&token)
			tc := oauth2.NewClient(ctx, ts)

			client := gitlab.NewOAuthClient(tc, token.AccessToken)
			if err := client.SetBaseURL(gitlabURL); err != nil {
				return nil, errors.Wrap(err, "can't set base url to GitLab client lib")
			}
			return &goGitlab{client: client}, nil
		},
	}
}

type goGitlabProxy interface {
	ListGroupProjects(owner string, options *gitlab.ListGroupProjectsOptions) ([]*gitlab.Project, *gitlab.Response, error)
	ListProjectHooks(projectNameWithNamespace string, options *gitlab.ListProjectHooksOptions) ([]*gitlab.ProjectHook, *gitlab.Response, error)
	AddProjectHook(projectNameWithNamespace string, options *gitlab.AddProjectHookOptions) error
}

type goGitlab struct {
	client *gitlab.Client
}

func (g *goGitlab) ListGroupProjects(owner string, options *gitlab.ListGroupProjectsOptions) ([]*gitlab.Project, *gitlab.Response, error) {
	return g.client.Groups.ListGroupProjects(owner, options)
}

func (g *goGitlab) ListProjectHooks(projectNameWithNamespace string, options *gitlab.ListProjectHooksOptions) ([]*gitlab.ProjectHook, *gitlab.Response, error) {
	return g.client.Projects.ListProjectHooks(projectNameWithNamespace, options)
}

func (g *goGitlab) AddProjectHook(projectNameWithNamespace string, options *gitlab.AddProjectHookOptions) error {
	_, _, err := g.client.Projects.AddProjectHook(projectNameWithNamespace, options)
	return err
}
