package gitlab

import (
	"context"
	"errors"
	"fmt"
	"time"

	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

// DefaultRequestTimeout specifies default value for request timeouts.
const DefaultRequestTimeout = 5 * time.Second

// Errors returned by this package.
var (
	ErrNotFound        = errors.New("not found")
	ErrPrivateResource = errors.New("private resource")
)

// Gitlab is a client to call GitLab api see New() to build one
type Gitlab interface {
	GetCurrentUser(userID string, token oauth2.Token) (*GitlabUserInfo, error)
	GetUserDetails(user *GitlabUserInfo) (*internGitlab.User, error)
	GetProject(user *GitlabUserInfo, owner, repo string) (*internGitlab.Project, error)
	GetReviews(user *GitlabUserInfo) ([]*internGitlab.MergeRequest, error)
	GetYourPrs(user *GitlabUserInfo) ([]*internGitlab.MergeRequest, error)
	GetYourAssignments(user *GitlabUserInfo) ([]*internGitlab.Issue, error)
	GetUnreads(user *GitlabUserInfo) ([]*internGitlab.Todo, error)
	GetProjectHooks(user *GitlabUserInfo, owner string, repo string) ([]*internGitlab.ProjectHook, error)
	// ResolveNamespaceAndProject accepts full path to User, Group or namespaced Project and returns corresponding
	// namespace and project name.
	//
	// ErrNotFound will be returned if no resource can be found.
	// If allowPrivate is set to false, and resolved group/project is private, ErrPrivateResource will be returned.
	ResolveNamespaceAndProject(
		userInfo *GitlabUserInfo,
		fullPath string,
		allowPrivate bool,
	) (namespace string, project string, err error)
}

type gitlab struct {
	enterpriseBaseURL string
	gitlabGroup       string
	checkGroup        func(projectNameWithGroup string) error
}

// New return a client to call GitLab API
func New(enterpriseBaseURL string, gitlabGroup string, checkGroup func(projectNameWithGroup string) error) Gitlab {
	return &gitlab{enterpriseBaseURL: enterpriseBaseURL, gitlabGroup: gitlabGroup, checkGroup: checkGroup}
}

func (g *gitlab) gitlabConnect(token oauth2.Token) (*internGitlab.Client, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&token)
	tc := oauth2.NewClient(ctx, ts)

	if len(g.enterpriseBaseURL) == 0 {
		return internGitlab.NewOAuthClient(tc, token.AccessToken), nil
	}

	client := internGitlab.NewOAuthClient(tc, token.AccessToken)
	if err := client.SetBaseURL(g.enterpriseBaseURL); err != nil {
		return nil, fmt.Errorf("can't set base url to GitLab client lib: %w", err)
	}
	return client, nil
}
