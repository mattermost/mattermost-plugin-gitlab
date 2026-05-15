// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
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

func TestHandleCreateMergeRequest_Validation(t *testing.T) {
	cases := []struct {
		name   string
		input  CreateMergeRequestInput
		errSub string
	}{
		{"empty project_path", CreateMergeRequestInput{Title: "T", SourceBranch: "feat", TargetBranch: "main"}, "project_path is required"},
		{"empty title", CreateMergeRequestInput{ProjectPath: "g/p", SourceBranch: "feat", TargetBranch: "main"}, "title is required"},
		{"empty source_branch", CreateMergeRequestInput{ProjectPath: "g/p", Title: "T", TargetBranch: "main"}, "source_branch is required"},
		{"empty target_branch", CreateMergeRequestInput{ProjectPath: "g/p", Title: "T", SourceBranch: "feat"}, "target_branch is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _ := newPluginWithMockGitlab(t)
			_, _, err := p.handleCreateMergeRequest(context.Background(), nil, tc.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errSub)
		})
	}
}

func TestHandleSearchIssues_Validation(t *testing.T) {
	p, _ := newPluginWithMockGitlab(t)
	_, _, err := p.handleSearchIssues(context.Background(), nil, SearchIssuesInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search term is required")
}

func TestHandleSearchMergeRequests_Validation(t *testing.T) {
	p, _ := newPluginWithMockGitlab(t)
	_, _, err := p.handleSearchMergeRequests(context.Background(), nil, SearchMergeRequestsInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search term is required")
}

func TestHandleRunPipeline_Validation(t *testing.T) {
	t.Run("empty project_path", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleRunPipeline(context.Background(), nil, RunPipelineInput{Ref: "main"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_path is required")
	})

	t.Run("empty ref", func(t *testing.T) {
		p, _ := newPluginWithMockGitlab(t)
		_, _, err := p.handleRunPipeline(context.Background(), nil, RunPipelineInput{ProjectPath: "g/p"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ref is required")
	})
}

func TestHandleAddIssueComment_Validation(t *testing.T) {
	cases := []struct {
		name   string
		input  AddIssueCommentInput
		errSub string
	}{
		{"empty project_path", AddIssueCommentInput{IssueIID: 1, Body: "hi"}, "project_path is required"},
		{"zero issue_iid", AddIssueCommentInput{ProjectPath: "g/p", Body: "hi"}, "issue_iid must be a positive integer"},
		{"empty body", AddIssueCommentInput{ProjectPath: "g/p", IssueIID: 1}, "body is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _ := newPluginWithMockGitlab(t)
			_, _, err := p.handleAddIssueComment(context.Background(), nil, tc.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errSub)
		})
	}
}

func TestHandleAddMergeRequestComment_Validation(t *testing.T) {
	cases := []struct {
		name   string
		input  AddMergeRequestCommentInput
		errSub string
	}{
		{"empty project_path", AddMergeRequestCommentInput{MergeRequestID: 1, Body: "hi"}, "project_path is required"},
		{"zero merge_request_iid", AddMergeRequestCommentInput{ProjectPath: "g/p", Body: "hi"}, "merge_request_iid must be a positive integer"},
		{"empty body", AddMergeRequestCommentInput{ProjectPath: "g/p", MergeRequestID: 1}, "body is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, _ := newPluginWithMockGitlab(t)
			_, _, err := p.handleAddMergeRequestComment(context.Background(), nil, tc.input)
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

func TestPipelineInfoToSummary(t *testing.T) {
	t.Run("nil returns zero value", func(t *testing.T) {
		assert.Equal(t, 0, pipelineInfoToSummary(nil).ID)
	})

	t.Run("populates fields", func(t *testing.T) {
		ts := time.Date(2026, 5, 8, 9, 0, 0, 0, time.UTC)
		s := pipelineInfoToSummary(&internGitlab.PipelineInfo{
			ID:        123,
			Status:    "success",
			Ref:       "main",
			SHA:       "abc123",
			WebURL:    "https://gitlab.com/g/p/-/pipelines/123",
			CreatedAt: &ts,
			UpdatedAt: &ts,
		})
		assert.Equal(t, 123, s.ID)
		assert.Equal(t, "success", s.Status)
		assert.Equal(t, "main", s.Ref)
		assert.NotEmpty(t, s.CreatedAt)
	})
}

func TestTodoToSummary(t *testing.T) {
	t.Run("nil returns zero value", func(t *testing.T) {
		assert.Equal(t, 0, todoToSummary(nil).ID)
	})

	t.Run("populates fields", func(t *testing.T) {
		s := todoToSummary(&internGitlab.Todo{
			ID:         99,
			ActionName: internGitlab.TodoAction("assigned"),
			TargetType: internGitlab.TodoTargetType("Issue"),
			Target: &internGitlab.TodoTarget{
				Title:  "Fix the thing",
				WebURL: "https://gitlab.com/g/p/-/issues/1",
			},
			Project: &internGitlab.BasicProject{PathWithNamespace: "g/p"},
		})
		assert.Equal(t, 99, s.ID)
		assert.Equal(t, "assigned", s.ActionName)
		assert.Equal(t, "Issue", s.TargetType)
		assert.Equal(t, "Fix the thing", s.TargetTitle)
		assert.Equal(t, "g/p", s.ProjectPath)
		assert.Equal(t, "https://gitlab.com/g/p/-/issues/1", s.WebURL)
	})

	t.Run("nil target and project safe", func(t *testing.T) {
		s := todoToSummary(&internGitlab.Todo{ID: 1})
		assert.Empty(t, s.TargetTitle)
		assert.Empty(t, s.ProjectPath)
	})
}
