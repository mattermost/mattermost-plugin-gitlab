package gitlab

import (
	"context"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/pluginapi/experimental/bot/logger"

	"github.com/pkg/errors"
	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

const Gitlabdotcom = "https://gitlab.com"

// Errors returned by this package.
var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("access forbidden")
	ErrPrivateResource = errors.New("private resource")
)

// Gitlab is a client to call GitLab api see New() to build one
type Gitlab interface {
	GitlabConnect(token oauth2.Token) (*internGitlab.Client, error)
	GetCurrentUser(ctx context.Context, userID string, token oauth2.Token) (*UserInfo, error)
	CreateIssue(ctx context.Context, user *UserInfo, issue *IssueRequest, token *oauth2.Token) (*internGitlab.Issue, error)
	AttachCommentToIssue(ctx context.Context, user *UserInfo, issue *IssueRequest, permalink, commentUsername string, token *oauth2.Token) (*internGitlab.Note, error)
	SearchIssues(ctx context.Context, user *UserInfo, search string, token *oauth2.Token) ([]*internGitlab.Issue, error)
	GetYourProjects(ctx context.Context, user *UserInfo, token *oauth2.Token) ([]*internGitlab.Project, error)
	GetLabels(ctx context.Context, user *UserInfo, projectID string, token *oauth2.Token) ([]*internGitlab.Label, error)
	GetProjectMembers(ctx context.Context, user *UserInfo, projectID string, token *oauth2.Token) ([]*internGitlab.ProjectMember, error)
	GetMilestones(ctx context.Context, user *UserInfo, projectID string, token *oauth2.Token) ([]*internGitlab.Milestone, error)
	GetIssueByID(ctx context.Context, user *UserInfo, owner, repo string, issueID int, token *oauth2.Token) (*Issue, error)
	GetMergeRequestByID(ctx context.Context, user *UserInfo, owner, repo string, mergeRequestID int, token *oauth2.Token) (*MergeRequest, error)
	GetUserDetails(ctx context.Context, user *UserInfo, token *oauth2.Token) (*internGitlab.User, error)
	GetProject(ctx context.Context, user *UserInfo, token *oauth2.Token, owner, repo string) (*internGitlab.Project, error)
	GetYourPrDetails(ctx context.Context, log logger.Logger, user *UserInfo, token *oauth2.Token, prList []*PRDetails) ([]*PRDetails, error)
	GetReviews(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*MergeRequest, error)
	GetYourAssignedPrs(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*MergeRequest, error)
	GetLHSData(ctx context.Context, user *UserInfo, token *oauth2.Token) (*LHSContent, error)
	GetYourAssignedIssues(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*Issue, error)
	GetToDoList(ctx context.Context, user *UserInfo, client *internGitlab.Client) ([]*internGitlab.Todo, error)
	GetProjectHooks(ctx context.Context, user *UserInfo, token *oauth2.Token, owner string, repo string) ([]*WebhookInfo, error)
	GetGroupHooks(ctx context.Context, user *UserInfo, token *oauth2.Token, owner string) ([]*WebhookInfo, error)
	NewProjectHook(ctx context.Context, user *UserInfo, token *oauth2.Token, projectID interface{}, projectHookOptions *AddWebhookOptions) (*WebhookInfo, error)
	NewGroupHook(ctx context.Context, user *UserInfo, token *oauth2.Token, groupName string, groupHookOptions *AddWebhookOptions) (*WebhookInfo, error)
	TriggerProjectPipeline(userInfo *UserInfo, token *oauth2.Token, projectID string, ref string) (*PipelineInfo, error)
	// ResolveNamespaceAndProject accepts full path to User, Group or namespaced Project and returns corresponding
	// namespace and project name.
	//
	// ErrNotFound will be returned if no resource can be found.
	// If allowPrivate is set to false, and resolved group/project is private, ErrPrivateResource will be returned.
	ResolveNamespaceAndProject(
		ctx context.Context,
		userInfo *UserInfo,
		token *oauth2.Token,
		fullPath string,
		allowPrivate bool,
	) (namespace string, project string, err error)
}

type gitlab struct {
	gitlabURL   string
	gitlabGroup string
	checkGroup  func(projectNameWithGroup string) error
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
func New(gitlabURL string, gitlabGroup string, checkGroup func(projectNameWithGroup string) error) Gitlab {
	if gitlabURL == "" {
		gitlabURL = Gitlabdotcom
	}
	return &gitlab{gitlabURL: gitlabURL, gitlabGroup: gitlabGroup, checkGroup: checkGroup}
}

func (g *gitlab) GitlabConnect(token oauth2.Token) (*internGitlab.Client, error) {
	if g.gitlabURL == "" || strings.EqualFold(g.gitlabURL, Gitlabdotcom) {
		return internGitlab.NewOAuthClient(token.AccessToken)
	}

	return internGitlab.NewOAuthClient(token.AccessToken, internGitlab.WithBaseURL(g.gitlabURL))
}

// checkResponse returns known errors based on the http status code.
func checkResponse(resp *internGitlab.Response) error {
	if resp == nil {
		return nil
	}

	switch resp.StatusCode {
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return nil
	}
}

// PrettyError returns an err in a better readable way.
func PrettyError(err error) error {
	var errResp *internGitlab.ErrorResponse
	if errors.As(err, &errResp) {
		return errors.New(errResp.Message)
	}

	return err
}
