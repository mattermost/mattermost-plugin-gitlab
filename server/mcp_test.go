// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost-plugin-agents/external/pluginmcp"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	internGitlab "github.com/xanzy/go-gitlab"
	gomock "go.uber.org/mock/gomock"

	mockgitlab "github.com/mattermost/mattermost-plugin-gitlab/server/mocks"
)

// --- MCP lifecycle tests ----------------------------------------------------

func TestServeMCPHTTP_NilServer(t *testing.T) {
	p := &Plugin{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/mcp", nil)

	p.serveMCPHTTP(w, r)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "MCP server not initialized")
}

func TestStartMCP_Idempotent(t *testing.T) {
	api := &plugintest.API{}
	api.On("GetServerVersion").Return("11.3.0").Maybe()
	api.On("LogInfo", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogError", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Maybe()
	api.On("LogDebug", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Maybe()
	// pluginmcp.Register calls PluginHTTP to reach the Agents plugin.
	// Return a nil response so that registration fails gracefully and the
	// plugin continues to operate.
	api.On("PluginHTTP", mock.Anything).Return((*http.Response)(nil)).Maybe()

	p := &Plugin{}
	p.SetAPI(api)

	// Concurrent calls should produce at most one mcpServer instance and must
	// not data-race.
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			p.startMCP()
		})
	}
	wg.Wait()

	p.mcpMu.Lock()
	s := p.mcpServer
	p.mcpMu.Unlock()
	require.NotNil(t, s, "mcpServer should be set after startMCP")

	// Clean up the background registration goroutine.
	_ = s.Unregister()
}

func TestServerSupportsMCP(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"11.3.0", true},
		{"11.4.1", true},
		{"12.0.0", true},
		{"11.2.0", false},
		{"10.7.0", false},
		{"9.11.0", false},
		{"", true},            // unparseable: don't disable on a version quirk
		{"garbage", true},     // unparseable
		{"11.3.0-rc1", false}, // prerelease of 11.3 sorts below 11.3.0
	}
	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			assert.Equal(t, tc.want, serverSupportsMCP(tc.version))
		})
	}
}

func TestStartMCP_SkipsOnOldServer(t *testing.T) {
	api := &plugintest.API{}
	api.On("GetServerVersion").Return("11.2.0")
	api.On("LogWarn", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once()

	p := &Plugin{}
	p.SetAPI(api)
	p.startMCP()

	p.mcpMu.Lock()
	defer p.mcpMu.Unlock()
	assert.Nil(t, p.mcpServer, "MCP server should not be created on an unsupported server")
	api.AssertExpectations(t)
}

func TestStopMCP_NilSafe(t *testing.T) {
	p := &Plugin{}
	require.NotPanics(t, func() {
		p.stopMCP()
	})
}

func TestStopMCP_ClearsServer(t *testing.T) {
	api := &plugintest.API{}
	api.On("LogWarn", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Maybe()
	api.On("LogWarn", mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

	stub := &mcpStub{unregisterErr: nil}
	p := &Plugin{}
	p.SetAPI(api)
	p.mcpServer = stub

	p.stopMCP()

	p.mcpMu.Lock()
	defer p.mcpMu.Unlock()
	assert.Nil(t, p.mcpServer)
	assert.True(t, stub.unregistered)
}

// mcpStub is a minimal mcpServer implementation for unit tests.
type mcpStub struct {
	mu            sync.Mutex
	unregistered  bool
	unregisterErr error
}

func (s *mcpStub) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}

func (s *mcpStub) Register() error { return nil }

func (s *mcpStub) Unregister() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unregistered = true
	return s.unregisterErr
}

// --- End-to-end ServeHTTP tools/list test -----------------------------------

// expectedToolNamePrefix mirrors pluginmcp's sanitization: the plugin ID with
// any character outside [A-Za-z0-9_-] replaced by '_', plus the "__" separator.
func expectedToolNamePrefix() string {
	var b strings.Builder
	for _, r := range manifest.Id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String() + "__"
}

// TestMCP_ToolsListOverServeHTTP drives a real pluginmcp.Server through
// ServeHTTP and a streamable MCP client to verify the namespaced tool names,
// generated schemas, and annotations actually appear on the wire.
func TestMCP_ToolsListOverServeHTTP(t *testing.T) {
	ctx := context.Background()

	p := &Plugin{}
	s := pluginmcp.NewServer(nil, pluginmcp.Config{
		PluginID: manifest.Id,
		Name:     mcpServerName,
		Path:     mcpBasePath,
		Version:  "0.0.1",
	})
	p.registerTools(s)

	// Inject the trusted inter-plugin header the Agents plugin would add.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("Mattermost-Plugin-ID", "mattermost-ai")
		s.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "gitlab-test-client", Version: "0.0.1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	res, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.NotEmpty(t, res.Tools)

	prefix := expectedToolNamePrefix()
	byShortName := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		require.Truef(t, strings.HasPrefix(tool.Name, prefix), "tool %q missing namespace prefix %q", tool.Name, prefix)
		require.NotNilf(t, tool.InputSchema, "tool %q should expose a generated input schema", tool.Name)
		byShortName[strings.TrimPrefix(tool.Name, prefix)] = tool
	}

	// Keep the surface small: every tool's schema is sent on each LLM call.
	assert.LessOrEqual(t, len(res.Tools), 10, "MCP tool count should stay within the pluginmcp budget")

	t.Run("read tool is annotated read-only", func(t *testing.T) {
		getIssue := byShortName["get_issue"]
		require.NotNil(t, getIssue)
		require.NotNil(t, getIssue.Annotations)
		assert.True(t, getIssue.Annotations.ReadOnlyHint)
	})

	t.Run("create_issue is a non-destructive write", func(t *testing.T) {
		createIssue := byShortName["create_issue"]
		require.NotNil(t, createIssue)
		require.NotNil(t, createIssue.Annotations)
		require.NotNil(t, createIssue.Annotations.DestructiveHint)
		assert.False(t, *createIssue.Annotations.DestructiveHint)
	})
}

// --- resolveCaller tests ----------------------------------------------------

func TestResolveCaller_NoUserID(t *testing.T) {
	p := &Plugin{}
	_, _, err := p.resolveCaller(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Mattermost user ID")
}

// --- splitProjectPath tests -------------------------------------------------

func TestSplitProjectPath(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantOwner  string
		wantRepo   string
		wantErrSub string
	}{
		{
			name:      "simple namespace/project",
			input:     "mygroup/myproject",
			wantOwner: "mygroup",
			wantRepo:  "myproject",
		},
		{
			name:      "nested group",
			input:     "top/sub/myproject",
			wantOwner: "top/sub",
			wantRepo:  "myproject",
		},
		{
			name:       "empty string",
			input:      "",
			wantErrSub: "project_path must be in namespace/project format",
		},
		{
			name:       "no slash — no owner",
			input:      "repoonly",
			wantErrSub: "namespace/project format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := splitProjectPath(tt.input)
			if tt.wantErrSub != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSub)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}

// --- Handler validation tests (mocked GitlabClient) ------------------------

func newPluginWithMockGitlab(t *testing.T) (*Plugin, *mockgitlab.MockGitlab) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockGL := mockgitlab.NewMockGitlab(ctrl)

	api := &plugintest.API{}
	api.On("LogWarn", mock.AnythingOfType("string"), mock.Anything, mock.Anything).Maybe()

	p := &Plugin{
		GitlabClient: mockGL,
	}
	p.SetAPI(api)
	return p, mockGL
}

func TestHandleGetIssue_Validation(t *testing.T) {
	t.Run("empty project_path", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleGetIssue(context.Background(), nil, GetIssueInput{IssueIID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_path is required")
	})

	t.Run("zero iid", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleGetIssue(context.Background(), nil, GetIssueInput{ProjectPath: "g/p"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issue_iid must be a positive integer")
	})

	t.Run("no caller in context", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleGetIssue(context.Background(), nil, GetIssueInput{ProjectPath: "g/p", IssueIID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no Mattermost user ID")
	})
}

func TestHandleGetMergeRequest_Validation(t *testing.T) {
	t.Run("empty project_path", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleGetMergeRequest(context.Background(), nil, GetMergeRequestInput{MergeRequestID: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_path is required")
	})

	t.Run("zero merge_request_iid", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleGetMergeRequest(context.Background(), nil, GetMergeRequestInput{ProjectPath: "g/p"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "merge_request_iid must be a positive integer")
	})
}

func TestHandleCreateIssue_Validation(t *testing.T) {
	t.Run("empty project_path", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleCreateIssue(context.Background(), nil, CreateIssueInput{Title: "T"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_path is required")
	})

	t.Run("empty title", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleCreateIssue(context.Background(), nil, CreateIssueInput{ProjectPath: "g/p"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})
}

func TestHandleGetProjectMetadata_Validation(t *testing.T) {
	t.Run("empty project_path", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleGetProjectMetadata(context.Background(), nil, GetProjectMetadataInput{Kind: "labels"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_path is required")
	})

	t.Run("invalid kind", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleGetProjectMetadata(context.Background(), nil, GetProjectMetadataInput{ProjectPath: "g/p", Kind: "bogus"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kind must be")
	})
}

func TestHandleAddComment_Validation(t *testing.T) {
	cases := []struct {
		name   string
		input  AddCommentInput
		errSub string
	}{
		{"empty project_path", AddCommentInput{TargetType: "issue", TargetIID: 1, Body: "hi"}, "project_path is required"},
		{"zero target_iid", AddCommentInput{TargetType: "issue", ProjectPath: "g/p", Body: "hi"}, "target_iid must be a positive integer"},
		{"empty body", AddCommentInput{TargetType: "issue", ProjectPath: "g/p", TargetIID: 1}, "body is required"},
		{"invalid target_type", AddCommentInput{TargetType: "epic", ProjectPath: "g/p", TargetIID: 1, Body: "hi"}, "target_type must be"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _ := newPluginWithMockGitlab(t)
			_, _, err := p.handleAddComment(context.Background(), nil, tc.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errSub)
		})
	}
}

// --- Conversion helper tests ------------------------------------------------

func TestIssueToSummary(t *testing.T) {
	t.Run("full issue", func(t *testing.T) {
		ts := time.Date(2026, 5, 8, 9, 0, 0, 0, time.UTC)
		issue := &internGitlab.Issue{
			ID:          100,
			IID:         42,
			ProjectID:   7,
			Title:       "Fix the bug",
			State:       "opened",
			Description: "A nasty bug",
			Labels:      internGitlab.Labels{"bug", "priority::high"},
			Assignees:   []*internGitlab.IssueAssignee{{Username: "alice"}, {Username: "bob"}},
			Milestone:   &internGitlab.Milestone{Title: "v2.0"},
			WebURL:      "https://gitlab.com/g/p/-/issues/42",
			CreatedAt:   &ts,
			UpdatedAt:   &ts,
		}

		s := issueToSummary(issue)

		assert.Equal(t, 100, s.ID)
		assert.Equal(t, 42, s.IID)
		assert.Equal(t, "Fix the bug", s.Title)
		assert.Equal(t, "opened", s.State)
		assert.Equal(t, []string{"bug", "priority::high"}, s.Labels)
		assert.Equal(t, []string{"alice", "bob"}, s.Assignees)
		assert.Equal(t, "v2.0", s.Milestone)
		assert.NotEmpty(t, s.CreatedAt)
	})

	t.Run("nil issue returns zero value", func(t *testing.T) {
		s := issueToSummary(nil)
		assert.Empty(t, s.Title)
	})

	t.Run("nil optional fields", func(t *testing.T) {
		s := issueToSummary(&internGitlab.Issue{ID: 1, Title: "Min"})
		assert.Empty(t, s.Assignees)
		assert.Empty(t, s.Milestone)
		assert.Empty(t, s.CreatedAt)
	})
}

func TestMrToSummary(t *testing.T) {
	t.Run("full MR", func(t *testing.T) {
		ts := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
		mr := &internGitlab.MergeRequest{
			ID:           200,
			IID:          15,
			ProjectID:    7,
			Title:        "Add feature X",
			State:        "opened",
			SourceBranch: "feature/x",
			TargetBranch: "main",
			Author:       &internGitlab.BasicUser{Username: "carol"},
			Assignees:    []*internGitlab.BasicUser{{Username: "dave"}},
			Reviewers:    []*internGitlab.BasicUser{{Username: "eve"}},
			Labels:       internGitlab.Labels{"feature"},
			Milestone:    &internGitlab.Milestone{Title: "v3.0"},
			WebURL:       "https://gitlab.com/g/p/-/merge_requests/15",
			CreatedAt:    &ts,
			UpdatedAt:    &ts,
		}

		s := mrToSummary(mr)

		assert.Equal(t, 200, s.ID)
		assert.Equal(t, 15, s.IID)
		assert.Equal(t, "carol", s.Author)
		assert.Equal(t, []string{"dave"}, s.Assignees)
		assert.Equal(t, []string{"eve"}, s.Reviewers)
		assert.Equal(t, "v3.0", s.Milestone)
		assert.Equal(t, "feature/x", s.SourceBranch)
	})

	t.Run("nil MR returns zero value", func(t *testing.T) {
		s := mrToSummary(nil)
		assert.Empty(t, s.Title)
	})
}

func TestIssuesToSummaries_Order(t *testing.T) {
	issues := []*internGitlab.Issue{
		{ID: 1, Title: "First"},
		{ID: 2, Title: "Second"},
		nil, // should be skipped
	}
	out := issuesToSummaries(issues)
	require.Len(t, out, 2)
	assert.Equal(t, 1, out[0].ID)
	assert.Equal(t, 2, out[1].ID)
}

func TestMrsToSummaries_SkipsNil(t *testing.T) {
	mrs := []*internGitlab.MergeRequest{nil, {ID: 5, Title: "OK"}, nil}
	out := mrsToSummaries(mrs)
	require.Len(t, out, 1)
	assert.Equal(t, 5, out[0].ID)
}

func TestNoteWebURL(t *testing.T) {
	t.Run("issue note", func(t *testing.T) {
		got := noteWebURL("https://gitlab.com", "g/p", "issues", 42, 7)
		assert.Equal(t, "https://gitlab.com/g/p/-/issues/42#note_7", got)
	})

	t.Run("merge request note", func(t *testing.T) {
		got := noteWebURL("https://gitlab.example.com", "g/sub/p", "merge_requests", 15, 99)
		assert.Equal(t, "https://gitlab.example.com/g/sub/p/-/merge_requests/15#note_99", got)
	})

	t.Run("trims trailing slash on base URL", func(t *testing.T) {
		got := noteWebURL("https://gitlab.com/", "g/p", "issues", 42, 7)
		assert.Equal(t, "https://gitlab.com/g/p/-/issues/42#note_7", got)
	})

	t.Run("missing base URL returns empty", func(t *testing.T) {
		assert.Empty(t, noteWebURL("", "g/p", "issues", 1, 1))
	})

	t.Run("missing project path returns empty", func(t *testing.T) {
		assert.Empty(t, noteWebURL("https://gitlab.com", "", "issues", 1, 1))
	})
}

func TestSplitProjectPathParts(t *testing.T) {
	owner, repo := splitProjectPathParts("group/sub/project")
	assert.Equal(t, "group/sub", owner)
	assert.Equal(t, "project", repo)

	owner2, repo2 := splitProjectPathParts("simple/repo")
	assert.Equal(t, "simple", owner2)
	assert.Equal(t, "repo", repo2)

	owner3, repo3 := splitProjectPathParts("noslash")
	assert.Equal(t, "", owner3)
	assert.Equal(t, "noslash", repo3)
}

func TestProjectToSummary(t *testing.T) {
	t.Run("nil project returns zero value", func(t *testing.T) {
		assert.Empty(t, projectToSummary(nil).Name)
	})

	t.Run("populates fields", func(t *testing.T) {
		s := projectToSummary(&internGitlab.Project{
			ID:                7,
			Name:              "my-project",
			PathWithNamespace: "g/my-project",
			Description:       "Test",
			WebURL:            "https://gitlab.com/g/my-project",
			Visibility:        internGitlab.PublicVisibility,
			DefaultBranch:     "main",
		})
		assert.Equal(t, 7, s.ID)
		assert.Equal(t, "my-project", s.Name)
		assert.Equal(t, "g/my-project", s.PathWithNamespace)
		assert.Equal(t, "main", s.DefaultBranch)
		assert.Equal(t, "public", s.Visibility)
	})
}
