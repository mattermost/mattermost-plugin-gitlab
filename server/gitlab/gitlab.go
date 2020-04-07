package gitlab

import (
	"errors"
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
	GetProjectHooks(user *GitlabUserInfo, owner string, repo string) ([]*WebhookInfo, error)
	GetGroupHooks(user *GitlabUserInfo, owner string) ([]*WebhookInfo, error)
	NewProjectHook(user *GitlabUserInfo, projectID interface{}, projectHookOptions *AddWebhookOptions) (*WebhookInfo, error)
	NewGroupHook(user *GitlabUserInfo, groupName string, groupHookOptions *AddWebhookOptions) (*WebhookInfo, error)
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

// Scope identifies the scope of a webhook
type Scope int

const (
	//Group is a type for group hooks
	Group Scope = iota
	//Project is a type for project hooks
	Project
)

func (s Scope) String() string {
	return [...]string{"group", "project"}[s]
}

// WebhookInfo Provides information about group or project hooks.
type WebhookInfo struct {
	ID                       int
	URL                      string
	ConfidentialNoteEvents   bool
	PushEvents               bool
	IssuesEvents             bool
	ConfidentialIssuesEvents bool
	MergeRequestsEvents      bool
	TagPushEvents            bool
	NoteEvents               bool
	JobEvents                bool
	PipelineEvents           bool
	WikiPageEvents           bool
	EnableSSLVerification    bool
	CreatedAt                *time.Time
	Scope                    Scope
}

//AddWebhookOptions is a paramater object with options for creating a project or group hook.
type AddWebhookOptions struct {
	URL                      string
	ConfidentialNoteEvents   bool
	PushEvents               bool
	IssuesEvents             bool
	ConfidentialIssuesEvents bool
	MergeRequestsEvents      bool
	TagPushEvents            bool
	NoteEvents               bool
	JobEvents                bool
	PipelineEvents           bool
	WikiPageEvents           bool
	EnableSSLVerification    bool
	Token                    string
}

// New return a client to call GitLab API
func New(enterpriseBaseURL string, gitlabGroup string, checkGroup func(projectNameWithGroup string) error) Gitlab {
	return &gitlab{enterpriseBaseURL: enterpriseBaseURL, gitlabGroup: gitlabGroup, checkGroup: checkGroup}
}

func (g *gitlab) gitlabConnect(token oauth2.Token) (*internGitlab.Client, error) {
	if len(g.enterpriseBaseURL) == 0 {
		return internGitlab.NewOAuthClient(token.AccessToken)
	}

	client, err := internGitlab.NewOAuthClient(token.AccessToken)
	if err != nil {
		return nil, err
	}

	return client, nil
}
