package main

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	gitLabAPI "github.com/xanzy/go-gitlab"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
	mocks "github.com/mattermost/mattermost-plugin-gitlab/server/gitlab/mocks"
)

type subscribeCommandTest struct {
	testName       string
	parameters     []string
	want           string
	webhookInfo    []*gitlab.WebhookInfo
	mattermostURL  string
	projectHookErr error
	mockGitlab     bool
}

const subscribeSuccessMessage = "Successfully subscribed to group/project.\nA Webhook is needed, run ```/gitlab webhook add group/project``` to create one now."
const testGitlabToken = `{"access_token":"6328a1014b19f741489b48cdc4291d93aa2957b0cea67335a34dcdadaf212139","token_type":"Bearer","refresh_token":"e6453b621e8979214c9f8a1b0e1e39723df8af11cef5e0d613a2cb2e39bdfeb7","expiry":"3022-10-23T15:14:43.623638795-05:00"}`
const testEncryptionKey = `shD-LC2DElnQzUO50cbvlOvjsNnzfEbk`

var subscribeCommandTests = []subscribeCommandTest{
	{
		testName:   "No Subscriptions",
		parameters: []string{"list"},
		want:       "Currently there are no subscriptions in this channel",
	},
	{
		testName:      "Hook Found",
		parameters:    []string{"add", "group/project"},
		mockGitlab:    true,
		want:          "Successfully subscribed to group/project.",
		webhookInfo:   []*gitlab.WebhookInfo{{URL: "example.com/somewebhookURL"}},
		mattermostURL: "example.com",
	},
	{
		testName:      "No webhooks",
		parameters:    []string{"add", "group/project"},
		mattermostURL: "example.com",
		webhookInfo:   []*gitlab.WebhookInfo{{}},
		mockGitlab:    true,
		want:          subscribeSuccessMessage,
	},
	{
		testName:      "Multiple un-matching hooks",
		parameters:    []string{"add", "group/project"},
		mattermostURL: "example.com",
		mockGitlab:    true,
		webhookInfo:   []*gitlab.WebhookInfo{{URL: "www.anotherhook.io/wrong"}, {URL: "www.213210948239324.edu/notgood"}},
		want:          subscribeSuccessMessage,
	},
	{
		testName:       "Error getting webhooks",
		parameters:     []string{"add", "group"},
		mattermostURL:  "example.com",
		mockGitlab:     true,
		webhookInfo:    []*gitlab.WebhookInfo{{}},
		want:           "Unable to determine status of Webhook. See [setup instructions](https://github.com/mattermost/mattermost-plugin-gitlab#step-3-create-a-gitlab-webhook) to validate.",
		projectHookErr: errors.New("unable to get project hooks"),
	},
}

func TestSubscribeCommand(t *testing.T) {
	for _, test := range subscribeCommandTests {
		t.Run(test.testName, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)

			channelID := "12345"
			userInfo := &gitlab.UserInfo{}

			p := getTestPlugin(t, mockCtrl, test.webhookInfo, test.mattermostURL, test.projectHookErr, test.mockGitlab)
			subscribeMessage := p.subscribeCommand(context.Background(), test.parameters, channelID, &configuration{}, userInfo)

			assert.Equal(t, test.want, subscribeMessage, "Subscribe command message should be the same.")
		})
	}
}

type webhookCommandTest struct {
	testName    string
	parameters  []string
	scope       string
	webhookInfo []*gitlab.WebhookInfo
	want        string
	siteURL     string
	webhook     *gitlab.WebhookInfo
}

var listWebhookCommandTests = []webhookCommandTest{
	{
		testName:   "List Project hooks",
		parameters: []string{"list", "group/project"},
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
		want: "\n\n`http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook`" + `
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
		parameters: []string{"list", "group/project"},
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
		want: "\n\n`http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook`" + `
Triggers:
* Push Events
` + "\n\n`http://anotherURL`" + `
Triggers:
* Push Events
`,
	},
	{
		testName:   "List Group hooks",
		parameters: []string{"list", "group"},
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
		want: "\n\n`http://yourURL/plugins/com.github.manland.mattermost-plugin-gitlab/webhook`" + `
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
		parameters: []string{"invalid", "group"},
		want:       "Unknown webhook command: invalid",
	},
	{
		testName:   "List missing namespace",
		parameters: []string{"list"},
		want:       "Unknown action, please use `/gitlab help` to see all actions available.",
	},
}

func TestListWebhookCommand(t *testing.T) {
	for _, test := range listWebhookCommandTests {
		t.Run(test.testName, func(t *testing.T) {
			p := new(Plugin)

			mockCtrl := gomock.NewController(t)
			mockedClient := mocks.NewMockGitlab(mockCtrl)

			encryptedToken, _ := encrypt([]byte(testEncryptionKey), testGitlabToken)

			p.configuration = &configuration{
				EncryptionKey: testEncryptionKey,
			}

			api := &plugintest.API{}
			api.On("KVGet", "_usertoken").Return([]byte(encryptedToken), nil)
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)

			if test.scope == "project" {
				mockedClient.EXPECT().GetProjectHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(test.webhookInfo, nil)
				mockedClient.EXPECT().ResolveNamespaceAndProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return("group", "project", nil)
				p.GitlabClient = mockedClient
			} else if test.scope == "group" {
				mockedClient.EXPECT().GetGroupHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(test.webhookInfo, nil)
				mockedClient.EXPECT().ResolveNamespaceAndProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return("group", "", nil)
				p.GitlabClient = mockedClient
			}

			got := p.webhookCommand(context.Background(), test.parameters, &gitlab.UserInfo{}, true)
			assert.Equal(t, test.want, got)
		})
	}
}

func getTestPlugin(t *testing.T, mockCtrl *gomock.Controller, hooks []*gitlab.WebhookInfo, mattermostURL string, projectHookErr error, mockGitlab bool) *Plugin {
	p := new(Plugin)

	mockedClient := mocks.NewMockGitlab(mockCtrl)
	if mockGitlab {
		mockedClient.EXPECT().ResolveNamespaceAndProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("group", "project", nil)
		mockedClient.EXPECT().GetProjectHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(hooks, projectHookErr)
		if projectHookErr == nil {
			mockedClient.EXPECT().GetGroupHooks(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(hooks, projectHookErr)
		}
	}

	p.GitlabClient = mockedClient

	conf := &model.Config{}
	conf.ServiceSettings.SiteURL = &mattermostURL

	encryptedToken, _ := encrypt([]byte(testEncryptionKey), testGitlabToken)

	p.configuration = &configuration{
		EncryptionKey: testEncryptionKey,
	}

	var subVal []byte

	api := &plugintest.API{}
	api.On("GetConfig", mock.Anything).Return(conf)
	api.On("KVGet", "_usertoken").Return([]byte(encryptedToken), nil)
	api.On("KVGet", mock.Anything).Return(subVal, nil)
	api.On("KVSet", mock.Anything, mock.Anything).Return(nil)
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), mock.Anything, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil)
	api.On("PublishWebSocketEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)

	return p
}

var exampleWebhookWithAlltriggers = &gitlab.WebhookInfo{
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
		testName:   "Create project hook with defaults",
		parameters: []string{"add", "group/project"},
		want:       "Webhook Created:\n\n\n`https://example.com`" + allTriggersFormated,
		siteURL:    "https://example.com",
		webhook:    exampleWebhookWithAlltriggers,
	},
	{
		testName:   "Create project hook with all trigers",
		parameters: []string{"add", "group/project", "*"},
		want:       "Webhook Created:\n\n\n`https://example.com`" + allTriggersFormated,
		siteURL:    "https://example.com",
		webhook:    exampleWebhookWithAlltriggers,
	},
	{
		testName:   "Create project hook with explicit trigers",
		parameters: []string{"add", "group/project", "PushEvents,MergeRequestEvents"},
		want: "Webhook Created:\n\n\n`https://example.com`" + `
Triggers:
* Push Events
* Merge Request Events
`,
		siteURL: "https://example.com",
		webhook: &gitlab.WebhookInfo{
			URL:                 "https://example.com",
			PushEvents:          true,
			MergeRequestsEvents: true,
		},
	},
	{
		testName:   "Create project hook with explicit URL",
		parameters: []string{"add", "group/project", "*", "https://anothersite.com"},
		want:       "Webhook Created:\n\n\n`https://anothersite.com`" + allTriggersFormated,
		siteURL:    "https://example.com",
		webhook: &gitlab.WebhookInfo{
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
		testName:   "Create project hook with explicit token",
		parameters: []string{"add", "group/project", "*", "https://example.com", "1234abcd"},
		want:       "Webhook Created:\n\n\n`https://example.com`" + allTriggersFormated,
		siteURL:    "https://example.com",
		webhook:    exampleWebhookWithAlltriggers,
	},
	{
		testName:   "Create Group hook with defaults",
		parameters: []string{"add", "group"},
		want:       "Webhook Created:\n\n\n`https://example.com`" + allTriggersFormated,
		siteURL:    "https://example.com",
		scope:      "group",
		webhook:    exampleWebhookWithAlltriggers,
	},
}

func TestAddWebhookCommand(t *testing.T) {
	for _, test := range addWebhookCommandTests {
		t.Run(test.testName, func(t *testing.T) {
			p := new(Plugin)

			mockCtrl := gomock.NewController(t)
			mockedClient := mocks.NewMockGitlab(mockCtrl)

			if test.scope == "group" {
				mockedClient.EXPECT().NewGroupHook(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(test.webhook, nil)
				mockedClient.EXPECT().ResolveNamespaceAndProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return("group", "", nil)
			} else {
				project := &gitLabAPI.Project{ID: 4}
				mockedClient.EXPECT().ResolveNamespaceAndProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).Return("group", "project", nil)
				mockedClient.EXPECT().GetProject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(project, nil)
				mockedClient.EXPECT().NewProjectHook(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(test.webhook, nil)
			}
			p.GitlabClient = mockedClient

			conf := &model.Config{}
			conf.ServiceSettings.SiteURL = model.NewString(test.siteURL)

			encryptedToken, _ := encrypt([]byte(testEncryptionKey), testGitlabToken)

			p.configuration = &configuration{
				EncryptionKey: testEncryptionKey,
			}

			api := &plugintest.API{}
			api.On("GetConfig", mock.Anything).Return(conf)
			api.On("KVGet", "_usertoken").Return([]byte(encryptedToken), nil)
			p.SetAPI(api)
			p.client = pluginapi.NewClient(api, p.Driver)

			got := p.webhookCommand(context.Background(), test.parameters, &gitlab.UserInfo{}, true)

			assert.Equal(t, test.want, got)
		})
	}
}
