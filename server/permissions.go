// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"

	"github.com/pkg/errors"
	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

type permissionAction string

const actionManageWebhook permissionAction = "manage_webhook"

// insufficientPermissionsMessage is the generic, client-facing response when a
// user lacks the permissions to run a command. Details are logged server-side.
const insufficientPermissionsMessage = "You don't have the required permissions to run this command."

// minimumAccessLevels maps an action to the minimum GitLab access level required
// per scope, following GitLab's documented prerequisites. Add an entry here to
// gate a new action.
// https://docs.gitlab.com/user/project/integrations/webhooks/
var minimumAccessLevels = map[permissionAction]map[gitlab.Scope]internGitlab.AccessLevelValue{
	actionManageWebhook: {
		gitlab.Project: internGitlab.MaintainerPermissions,
		gitlab.Group:   internGitlab.OwnerPermissions,
	},
}

// getGitlabAccessLevel resolves the connected user's effective access level for a
// project (when project is non-empty) or group.
func (p *Plugin) getGitlabAccessLevel(ctx context.Context, info *gitlab.UserInfo, namespace, project string) (internGitlab.AccessLevelValue, error) {
	var level internGitlab.AccessLevelValue
	err := p.useGitlabClient(info, func(info *gitlab.UserInfo, token *oauth2.Token) error {
		var resp internGitlab.AccessLevelValue
		var err error
		if project != "" {
			resp, err = p.GitlabClient.GetProjectAccessLevel(ctx, info, token, namespace, project)
		} else {
			resp, err = p.GitlabClient.GetGroupAccessLevel(ctx, info, token, namespace)
		}
		if err != nil {
			return err
		}
		level = resp
		return nil
	})
	return level, err
}

// hasGitlabPermission reports whether the connected user may perform the given
// action on the resource (project when project is non-empty, otherwise group).
// It compares the user's effective GitLab access level against the minimum
// required for the action.
//
// When the access level cannot be determined (API failure), it falls back to the
// Mattermost System Admin check so behavior is never more permissive than before
// on error.
func (p *Plugin) hasGitlabPermission(ctx context.Context, info *gitlab.UserInfo, namespace, project string, action permissionAction) (bool, error) {
	scope := gitlab.Group
	if project != "" {
		scope = gitlab.Project
	}

	level, err := p.getGitlabAccessLevel(ctx, info, namespace, project)
	if err != nil {
		if errors.Is(err, gitlab.ErrNotFound) {
			return false, nil
		}
		p.client.Log.Warn("Falling back to System Admin check after GitLab permission lookup failed", "err", err.Error())
		return p.isAuthorizedSysAdmin(info.UserID)
	}

	required, ok := minimumAccessLevels[action][scope]
	if !ok {
		return false, errors.Errorf("no minimum access level configured for action %q scope %q", action, scope)
	}

	return level >= required, nil
}

// requireWebhookPermission checks whether the user has sufficient GitLab
// permissions to manage webhooks. Returns an empty string on success or the
// user-facing denial message.
func (p *Plugin) requireWebhookPermission(ctx context.Context, info *gitlab.UserInfo, group, project string) string {
	allowed, err := p.hasGitlabPermission(ctx, info, group, project, actionManageWebhook)
	if err != nil {
		p.client.Log.Warn("Failed to check GitLab permission for webhook", "err", err.Error())
	}
	if !allowed {
		return insufficientPermissionsMessage
	}
	return ""
}
