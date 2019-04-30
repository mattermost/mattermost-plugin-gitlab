package gitlab

import (
	"context"

	internGitlab "github.com/manland/go-gitlab"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type Gitlab interface {
	GetCurrentUser(userID string, token oauth2.Token) (*GitlabUserInfo, error)
	GetUserDetails(user *GitlabUserInfo) (*internGitlab.User, error)
	GetProject(user *GitlabUserInfo, owner, repo string) (*internGitlab.Project, error)
	GetReviews(user *GitlabUserInfo) ([]*internGitlab.MergeRequest, error)
	GetYourPrs(user *GitlabUserInfo) ([]*internGitlab.MergeRequest, error)
	GetYourAssignments(user *GitlabUserInfo) ([]*internGitlab.Issue, error)
	GetUnreads(user *GitlabUserInfo) ([]*internGitlab.Todo, error)
	Exist(user *GitlabUserInfo, owner string, repo string, enablePrivateRepo bool) (bool, error)
}

type gitlab struct {
	enterpriseBaseURL string
}

func New(enterpriseBaseURL string) Gitlab {
	return &gitlab{enterpriseBaseURL: enterpriseBaseURL}
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
		return nil, errors.Wrap(err, "can't set base url to gitlab client lib")
	}
	return client, nil
}
