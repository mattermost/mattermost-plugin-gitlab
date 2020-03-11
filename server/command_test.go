package main

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab/mocks"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	internGitlab "github.com/xanzy/go-gitlab"
)

type subscribeCommandTest struct {
	testName               string
	paramaters             []string
	want                   string
	projectHooks           []*internGitlab.ProjectHook
	mattermostURL          string
	errorOnGetProjectHooks bool
	mockGitlab             bool
}

func TestSubscribeCommand(t *testing.T) {

	subscribeCommandTests := []subscribeCommandTest{
		{
			testName:   "No Subscriptions",
			paramaters: []string{"list"},
			want:       "Currently there are no subscriptions in this channel",
		},
		{
			testName:      "Hook Found",
			paramaters:    []string{"group/project"},
			mockGitlab:    true,
			want:          "Successfully subscribed to group/project.",
			projectHooks:  []*internGitlab.ProjectHook{{URL: "example.com/somewebhookURL"}},
			mattermostURL: "example.com",
		},
		{
			testName:      "No webhooks",
			paramaters:    []string{"group/project"},
			mattermostURL: "example.com",
			projectHooks:  []*internGitlab.ProjectHook{{}},
			mockGitlab:    true,
			want:          "Successfully subscribed to group/project. Please [setup a WebHook](/group/project/hooks) in GitLab to complete integration. See [setup instructions](https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook) for more info.",
		},
		{
			testName:      "Multiple un-matching hooks",
			paramaters:    []string{"group/project"},
			mattermostURL: "example.com",
			mockGitlab:    true,
			projectHooks:  []*internGitlab.ProjectHook{{URL: "www.anotherhook.io/wrong"}, {URL: "www.213210948239324.edu/notgood"}},
			want:          "Successfully subscribed to group/project. Please [setup a WebHook](/group/project/hooks) in GitLab to complete integration. See [setup instructions](https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook) for more info.",
		},
		{
			testName:               "Error getting webhooks",
			paramaters:             []string{"group"},
			mattermostURL:          "example.com",
			mockGitlab:             true,
			projectHooks:           []*internGitlab.ProjectHook{{}},
			want:                   "Successfully subscribed to group. Unable to determine status of Webhook. See [setup instructions](https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook) to validate.",
			errorOnGetProjectHooks: true,
		},
	}

	for _, test := range subscribeCommandTests {
		t.Run(test.testName, func(t *testing.T) {

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			channelID := "12345"
			userInfo := &gitlab.GitlabUserInfo{}

			p := getTestPlugin(t, mockCtrl, test.projectHooks, test.mattermostURL, test.errorOnGetProjectHooks, test.mockGitlab)
			subscribeMessage := p.subscribeCommand(test.paramaters, channelID, &configuration{}, userInfo)

			assert.Equal(t, subscribeMessage, test.want, "Subscribe command message should be the same.")
		})

	}

}

func getTestPlugin(t *testing.T, mockCtrl *gomock.Controller, hooks []*internGitlab.ProjectHook, mattermostURL string, errorOnGetProjectHooks bool, mockGitlab bool) *Plugin {

	p := new(Plugin)

	mockedClient := mocks.NewMockGitlab(mockCtrl)
	if mockGitlab {
		mockedClient.EXPECT().ResolveNamespaceAndProject(gomock.Any(), gomock.Any(), gomock.Any()).Return("group", "project", nil)
		var projectHookError error
		if errorOnGetProjectHooks {
			projectHookError = errors.New("Unable to get project hooks")
		}
		mockedClient.EXPECT().GetProjectHooks(gomock.Any(), gomock.Any(), gomock.Any()).Return(hooks, projectHookError)
	}

	p.GitlabClient = mockedClient

	api := &plugintest.API{}

	conf := &model.Config{}
	conf.ServiceSettings.SiteURL = &mattermostURL
	api.On("GetConfig", mock.Anything).Return(conf)

	var subVal []byte
	api.On("KVGet", mock.Anything).Return(subVal, nil)
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil)
	p.SetAPI(api)
	return p
}
