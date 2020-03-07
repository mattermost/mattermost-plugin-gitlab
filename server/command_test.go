package main

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab/mocks"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	mocks "github.com/mattermost/mattermost-plugin-gitlab/server/gitlab/mocks"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gitLabAPI "github.com/xanzy/go-gitlab"
)

//	gitLabAPI ""
type subscribeCommandTest struct {
	testName               string
	paramaters             []string
	want                   string
	projectHooks           []*internGitlab.ProjectHook
	mattermostURL          string
	errorOnGetProjectHooks bool
	mockGitlab             bool
}

var subscribeCommandTests = []subscribeCommandTest{
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
		webhookInfo:   []*gitlab.WebhookInfo{{URL: "example.com/somewebhookURL"}},
		mattermostURL: "example.com",
	},
	{
		testName:      "No webhooks",
		paramaters:    []string{"group/project"},
		mattermostURL: "example.com",
		webhookInfo:   []*gitlab.WebhookInfo{{}},
		mockGitlab:    true,
		want:          "Successfully subscribed to group/project. Please [setup a WebHook](/group/project/hooks) in GitLab to complete integration. See [setup instructions](https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook) for more info.",
	},
	{
		testName:      "Multiple un-matching hooks",
		paramaters:    []string{"group/project"},
		mattermostURL: "example.com",
		mockGitlab:    true,
		webhookInfo:   []*gitlab.WebhookInfo{{URL: "www.anotherhook.io/wrong"}, {URL: "www.213210948239324.edu/notgood"}},
		want:          "Successfully subscribed to group/project. Please [setup a WebHook](/group/project/hooks) in GitLab to complete integration. See [setup instructions](https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook) for more info.",
	},
	{
		testName:       "Error getting webhooks",
		paramaters:     []string{"group"},
		mattermostURL:  "example.com",
		mockGitlab:     true,
		webhookInfo:    []*gitlab.WebhookInfo{{}},
		want:           "Successfully subscribed to group. Unable to determine status of Webhook. See [setup instructions](https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook) to validate.",
		projectHookErr: errors.New("Unable to get project hooks"), //true,
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

type webhookCommandTest struct {
	testName    string
	paramaters  []string
	scope       string
	webhookInfo []*gitlab.WebhookInfo
	want        string
	siteURL     string
	projectHook *gitLabAPI.ProjectHook
	secretToken string
}

var listWebhookCommandTests = []webhookCommandTest{
	{
		testName:   "List Project hooks",
		paramaters: []string{"list", "group/project"},
		scope:      "project",
		webhookInfo: []*gitlab.WebhookInfo{
			{
				URL:                      "http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook",
				PushEvents:               true,
				IssuesEvents:             true,
				ConfidentialIssuesEvents: true,
				MergeRequestsEvents:      true,
				TagPushEvents:            true,
				NoteEvents:               true,
				JobEvents:                false,
				PipelineEvents:           false,
				WikiPageEvents:           false,
			},
		},
		want: `
http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook
Triggers:
* Push Events
* Tag Push Events
* Comments
* Issues Events
* Confidential Issues Events
* Merge Request Events
`,
	},
	{
		testName:   "List multiple project hooks",
		paramaters: []string{"list", "group/project"},
		scope:      "project",
		webhookInfo: []*gitlab.WebhookInfo{
			{
				URL:        "http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook",
				PushEvents: true,
			},
			{
				URL:        "http://anotherURL",
				PushEvents: true,
			},
		},
		want: `
http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook
Triggers:
* Push Events

http://anotherURL
Triggers:
* Push Events
`,
	},
	{
		testName:   "List Group hooks",
		paramaters: []string{"list", "group"},
		scope:      "group",
		webhookInfo: []*gitlab.WebhookInfo{
			{
				URL:                      "http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook",
				PushEvents:               true,
				IssuesEvents:             true,
				ConfidentialIssuesEvents: true,
				MergeRequestsEvents:      true,
				TagPushEvents:            true,
				NoteEvents:               true,
				JobEvents:                false,
				PipelineEvents:           false,
				WikiPageEvents:           false,
			},
		},
		want: `
http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook
Triggers:
* Push Events
* Tag Push Events
* Comments
* Issues Events
* Confidential Issues Events
* Merge Request Events
`,
	},
	{
		testName:   "Unrecognized sub command",
		paramaters: []string{"invalid", "group"},
		want:       "Unknown webhook command: invalid",
	},
	{
		testName:   "List missing namespace",
		paramaters: []string{"list"},
		want:       "Unknown action, please use `/gitlab help` to see all actions available.",
	},
}

func TestListWebhookCommand(t *testing.T) {
	for _, test := range listWebhookCommandTests {
		t.Run(test.testName, func(t *testing.T) {
			p := new(Plugin)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockedClient := mocks.NewMockGitlab(mockCtrl)

			if test.scope == "project" {
				mockedClient.EXPECT().GetProjectHooks(gomock.Any(), gomock.Any(), gomock.Any()).Return(test.webhookInfo, nil)
				p.GitlabClient = mockedClient
			} else if test.scope == "group" {
				mockedClient.EXPECT().GetGroupHooks(gomock.Any(), gomock.Any()).Return(test.webhookInfo, nil)
				p.GitlabClient = mockedClient
			}

			got := p.webhookCommand(test.paramaters, &gitlab.GitlabUserInfo{})
			assert.Equal(t, test.want, got)
		})
	}
}

func getTestPlugin(t *testing.T, mockCtrl *gomock.Controller, hooks []*gitlab.WebhookInfo, mattermostURL string, projectHookErr error, mockGitlab bool) *Plugin {
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

var exampleProjectHookWithAlltriggers = &gitLabAPI.ProjectHook{
	URL:                      "https://example.com",
	PushEvents:               true,
	TagPushEvents:            true,
	NoteEvents:               true,
	ConfidentialNoteEvents:   true,
	IssuesEvents:             true,
	ConfidentialIssuesEvents: true,
	MergeRequestsEvents:      true,
	JobEvents:                true,
	PipelineEvents:           true,
	WikiPageEvents:           true,
	EnableSSLVerification:    true,
}

const allTriggersFormated = `
SSL Verification Enabled
Triggers:
* Push Events
* Tag Push Events
* Comments
* Confidential Comments
* Issues Events
* Confidential Issues Events
* Merge Request Events
* Job Events
* Pipeline Events
* Wiki Page Events
`

var addWebhookCommandTests = []webhookCommandTest{
	{
		testName:    "Create project hook with defaults",
		paramaters:  []string{"add", "group/project"},
		want:        "Webhook Created:\n\nhttps://example.com" + allTriggersFormated,
		siteURL:     "https://example.com",
		projectHook: exampleProjectHookWithAlltriggers,
	},
	{
		testName:    "Create project hook with all trigers",
		paramaters:  []string{"add", "group/project", "*"},
		want:        "Webhook Created:\n\nhttps://example.com" + allTriggersFormated,
		siteURL:     "https://example.com",
		projectHook: exampleProjectHookWithAlltriggers,
	},
	{
		testName:   "Create project hook with explicit trigers",
		paramaters: []string{"add", "group/project", "PushEvents,MergeRequestEvents"},
		want: `Webhook Created:

https://example.com
Triggers:
* Push Events
* Merge Request Events
`,
		siteURL: "https://example.com",
		projectHook: &gitLabAPI.ProjectHook{
			URL:                 "https://example.com",
			PushEvents:          true,
			MergeRequestsEvents: true,
		},
	},
	{
		testName:   "Create project hook with explicit URL",
		paramaters: []string{"add", "group/project", "*", "https://anothersite.com"},
		want:       "Webhook Created:\n\nhttps://anothersite.com" + allTriggersFormated,
		siteURL:    "https://example.com",
		projectHook: &gitLabAPI.ProjectHook{
			URL:                      "https://anothersite.com",
			EnableSSLVerification:    true,
			PushEvents:               true,
			TagPushEvents:            true,
			NoteEvents:               true,
			ConfidentialNoteEvents:   true,
			IssuesEvents:             true,
			ConfidentialIssuesEvents: true,
			MergeRequestsEvents:      true,
			JobEvents:                true,
			PipelineEvents:           true,
			WikiPageEvents:           true,
		},
	},
	{
		testName:    "Create project hook with explicit token",
		paramaters:  []string{"add", "group/project", "*", "https://example.com", "1234abcd"},
		want:        "Webhook Created:\n\nhttps://example.com" + allTriggersFormated,
		siteURL:     "https://example.com",
		projectHook: exampleProjectHookWithAlltriggers,
	},
	// TODOS
	//SSL validation
	//refact to use stingify consistently
	//implement group scoped procts
	//update instructions
	//update subscribe response

	//wont do
	//push events pattern would require upstream implimentation
}

func TestAddProjectHookCommand(t *testing.T) {

	for _, test := range addWebhookCommandTests {
		t.Run(test.testName, func(t *testing.T) {
			p := new(Plugin)

			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockedClient := mocks.NewMockGitlab(mockCtrl)

			project := &gitLabAPI.Project{ID: 4}
			mockedClient.EXPECT().GetProject(gomock.Any(), gomock.Any(), gomock.Any()).Return(project, nil)
			mockedClient.EXPECT().NewProjectHook(gomock.Any(), gomock.Any(), gomock.Any()).Return(test.projectHook, nil)
			p.GitlabClient = mockedClient

			api := &plugintest.API{}
			conf := &model.Config{}
			conf.ServiceSettings.SiteURL = &test.siteURL
			api.On("GetConfig", mock.Anything).Return(conf)
			p.SetAPI(api)

			got := p.webhookCommand(test.paramaters, &gitlab.GitlabUserInfo{})

			assert.Equal(t, test.want, got)
		})
	}
}
