package gitlab

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
)

type addWebHookDataTest struct {
	name     string
	owner    string
	repo     string
	url      string
	token    string
	initMock func(*mockGoGitlabProxy) *mockGoGitlabProxy
	finalize func(t *testing.T, err error)
}

var data = []addWebHookDataTest{
	{
		name:  "error ListProjectHooks",
		owner: "groupname",
		repo:  "projectName",
		url:   "http://mattermost.domain.com/plugins/id/webhook",
		token: "1234ABCD",
		initMock: func(m *mockGoGitlabProxy) *mockGoGitlabProxy {
			m.On("ListProjectHooks", "groupname/projectName", mock.AnythingOfType("*gitlab.ListProjectHooksOptions")).
				Return(nil, nil, errors.New("test"))
			return m
		},
		finalize: func(t *testing.T, err error) {
			if err == nil {
				t.Fatal("should return error!", err.Error())
			}
		},
	}, {
		name:  "group",
		owner: "groupname",
		repo:  "",
		url:   "http://mattermost.domain.com/plugins/id/webhook",
		token: "1234ABCD",
		initMock: func(m *mockGoGitlabProxy) *mockGoGitlabProxy {
			m.On("ListGroupProjects", "groupname", mock.AnythingOfType("*gitlab.ListGroupProjectsOptions")).
				Return(nil, nil, errors.New("test"))
			return m
		},
		finalize: func(t *testing.T, err error) {
			if err == nil {
				t.Fatal("should return error!", err.Error())
			}
		},
	},
}

func TestAddWebHooks(t *testing.T) {
	t.Parallel()
	for _, test := range data {
		t.Run(test.name, func(t *testing.T) {
			g := &myGitlab{
				gitlabGroup: "",
				checkGroup:  func(projectNameWithGroup string) error { return nil },
				connectProxy: func(token oauth2.Token) (goGitlabProxy, error) {
					return test.initMock(&mockGoGitlabProxy{}), nil
				},
			}
			err := g.AddWebHooks(&GitlabUserInfo{Token: &oauth2.Token{}}, test.owner, test.repo, test.url, test.token)
			test.finalize(t, err)
		})
	}
}
