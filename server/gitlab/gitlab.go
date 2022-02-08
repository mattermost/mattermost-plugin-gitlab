package gitlab

import (
	"context"
	"errors"
	"strings"

	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

const gitlabdotcom = "https://gitlab.com"

// Errors returned by this package.
var (
	ErrNotFound        = errors.New("not found")
	ErrPrivateResource = errors.New("private resource")
)

// Gitlab is a client to call GitLab api see New() to build one
type Gitlab interface {
	GetCurrentUser(ctx context.Context, userID string, token oauth2.Token) (*UserInfo, error)
	GetUserDetails(ctx context.Context, user *UserInfo) (*internGitlab.User, error)
	GetProject(ctx context.Context, user *UserInfo, owner, repo string) (*internGitlab.Project, error)
	GetReviews(ctx context.Context, user *UserInfo) ([]*internGitlab.MergeRequest, error)
	GetYourPrs(ctx context.Context, user *UserInfo) ([]*internGitlab.MergeRequest, error)
	GetYourAssignments(ctx context.Context, user *UserInfo) ([]*internGitlab.Issue, error)
	GetUnreads(ctx context.Context, user *UserInfo) ([]*internGitlab.Todo, error)
	GetProjectHooks(ctx context.Context, user *UserInfo, owner string, repo string) ([]*WebhookInfo, error)
	GetGroupHooks(ctx context.Context, user *UserInfo, owner string) ([]*WebhookInfo, error)
	NewProjectHook(ctx context.Context, user *UserInfo, projectID interface{}, projectHookOptions *AddWebhookOptions) (*WebhookInfo, error)
	NewGroupHook(ctx context.Context, user *UserInfo, groupName string, groupHookOptions *AddWebhookOptions) (*WebhookInfo, error)
	// ResolveNamespaceAndProject accepts full path to User, Group or namespaced Project and returns corresponding
	// namespace and project name.
	//
	// ErrNotFound will be returned if no resource can be found.
	// If allowPrivate is set to false, and resolved group/project is private, ErrPrivateResource will be returned.
	ResolveNamespaceAndProject(
		ctx context.Context,
		userInfo *UserInfo,
		fullPath string,
		allowPrivate bool,
	) (namespace string, project string, err error)
}

type gitlab struct {
	enterpriseBaseURL string
	gitlabGroup       string
	checkGroup        func(projectNameWithGroup string) error
}

// Scope identifies the scope of a webhook
type Scope int

const (
	// Group is a type for group hooks
	Group Scope = iota
	// Project is a type for project hooks
	Project
)

func (s Scope) String() string {
	return [...]string{"group", "project"}[s]
}

// New return a client to call GitLab API
func New(enterpriseBaseURL string, gitlabGroup string, checkGroup func(projectNameWithGroup string) error) Gitlab {
	return &gitlab{enterpriseBaseURL: enterpriseBaseURL, gitlabGroup: gitlabGroup, checkGroup: checkGroup}
}

func (g *gitlab) gitlabConnect(token oauth2.Token) (*internGitlab.Client, error) {
	if len(g.enterpriseBaseURL) == 0 || strings.EqualFold(g.enterpriseBaseURL, gitlabdotcom) {
		return internGitlab.NewOAuthClient(token.AccessToken)
	}

	return internGitlab.NewOAuthClient(token.AccessToken, internGitlab.WithBaseURL(g.enterpriseBaseURL))
}
