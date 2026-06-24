// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Masterminds/semver/v3"
	"github.com/mattermost/mattermost-plugin-agents/external/pluginmcp"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

const (
	mcpBasePath   = "/mcp"
	mcpServerName = "GitLab"

	// minServerVersionForMCP is the minimum Mattermost server version that
	// stamps the trusted Mattermost-Plugin-ID header on inter-plugin RPC and
	// supports PluginHTTPStream, both required by the Agents MCP bridge. On
	// older servers registration is rejected with a 401, so we skip it.
	minServerVersionForMCP = "11.3.0"
)

// mcpServer is an interface over *pluginmcp.Server so we can swap it with a
// nil-safe stub in tests without importing the real package.
type mcpServer interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Register() error
	Unregister() error
}

// startMCP initialises the pluginmcp server and registers it with the Agents
// plugin.  Panics from pluginmcp are caught so the GitLab plugin continues to
// start normally even when the Agents plugin is absent.
func (p *Plugin) startMCP() {
	defer func() {
		if rec := recover(); rec != nil {
			p.API.LogWarn("MCP server initialization panicked; GitLab plugin continues without MCP",
				"panic", fmt.Sprintf("%v", rec))
		}
	}()

	p.mcpMu.Lock()
	defer p.mcpMu.Unlock()
	if p.mcpServer != nil {
		return
	}

	if serverVersion := p.API.GetServerVersion(); !serverSupportsMCP(serverVersion) {
		p.API.LogWarn("Skipping GitLab MCP registration: Mattermost server is older than the version required by the Agents MCP bridge. Upgrade the server to expose GitLab tools to Agents.",
			"server_version", serverVersion,
			"required_version", minServerVersionForMCP,
		)
		return
	}

	s := pluginmcp.NewServer(p.API, pluginmcp.Config{
		PluginID: manifest.Id,
		Name:     mcpServerName,
		Path:     mcpBasePath,
		Version:  manifest.Version,
	})

	p.registerTools(s)
	p.mcpServer = s

	// Register returns nil immediately and retries with the Agents plugin in a
	// background goroutine, so there is no synchronous error to handle here.
	// pluginmcp emits its own (stdlib) logs on terminal failure; log the attempt
	// here so the plugin logs show that our side ran and what it pushed.
	p.API.LogInfo("Registering GitLab MCP server with the Agents plugin",
		"plugin_id", manifest.Id,
		"path", mcpBasePath,
	)
	_ = s.Register()
}

// serverSupportsMCP reports whether the running Mattermost server is new enough
// for the Agents MCP bridge. An unparseable/empty version is treated as
// supported so a version-string quirk never silently disables the feature.
func serverSupportsMCP(serverVersion string) bool {
	current, err := semver.NewVersion(serverVersion)
	if err != nil {
		return true
	}
	return !current.LessThan(semver.MustParse(minServerVersionForMCP))
}

func (p *Plugin) stopMCP() {
	p.mcpMu.Lock()
	s := p.mcpServer
	p.mcpServer = nil
	p.mcpMu.Unlock()

	if s == nil {
		return
	}
	if err := s.Unregister(); err != nil {
		p.API.LogWarn("MCP unregister failed", "err", err.Error())
	}
}

func (p *Plugin) serveMCPHTTP(w http.ResponseWriter, r *http.Request) {
	p.mcpMu.Lock()
	s := p.mcpServer
	p.mcpMu.Unlock()

	if s == nil {
		http.Error(w, "MCP server not initialized", http.StatusServiceUnavailable)
		return
	}
	s.ServeHTTP(w, r)
}

// resolveCaller extracts the Mattermost user ID injected by the Agents plugin,
// then retrieves the user's GitLab UserInfo and a valid (possibly refreshed)
// OAuth token.  It returns an error if the user has not connected their GitLab
// account.
func (p *Plugin) resolveCaller(ctx context.Context) (*gitlab.UserInfo, *oauth2.Token, error) {
	userID := pluginmcp.GetUserID(ctx)
	if userID == "" {
		return nil, nil, fmt.Errorf("no Mattermost user ID in context (request did not arrive through the Agents plugin)")
	}

	info, apiErr := p.getGitlabUserInfoByMattermostID(userID)
	if apiErr != nil {
		return nil, nil, fmt.Errorf("GitLab account not connected: %s", apiErr.Message)
	}

	// Don't DM the user on a revoked token here: an LLM may retry a failed tool
	// call several times, and we'd spam the same disconnect notice each time.
	token, err := p.getOrRefreshToken(info, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get GitLab token: %w", err)
	}

	return info, token, nil
}
