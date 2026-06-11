// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gitlab

import (
	"context"
	"fmt"
	"net/http"

	internGitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

// GetProjectAccessLevel returns the effective access level (including memberships
// inherited from ancestor groups) of the connected user for the given project.
//
// It calls GET /projects/:id/members/all/:user_id. ErrNotFound is returned when
// the user has no membership at all, which callers should treat as "no access".
func (g *gitlab) GetProjectAccessLevel(ctx context.Context, user *UserInfo, token *oauth2.Token, namespace, project string) (internGitlab.AccessLevelValue, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return internGitlab.NoPermissions, err
	}

	member, resp, err := client.ProjectMembers.GetInheritedProjectMember(
		fmt.Sprintf("%s/%s", namespace, project),
		user.GitlabUserID,
		internGitlab.WithContext(ctx),
	)
	if respErr := checkResponse(resp); respErr != nil {
		return internGitlab.NoPermissions, respErr
	}
	if err != nil {
		return internGitlab.NoPermissions, err
	}

	return member.AccessLevel, nil
}

// GetGroupAccessLevel returns the effective access level (including memberships
// inherited from ancestor groups) of the connected user for the given group.
//
// The go-gitlab client does not expose a single-user inherited group lookup, so
// this issues a raw request to GET /groups/:id/members/all/:user_id. ErrNotFound
// is returned when the user has no membership at all.
func (g *gitlab) GetGroupAccessLevel(ctx context.Context, user *UserInfo, token *oauth2.Token, group string) (internGitlab.AccessLevelValue, error) {
	client, err := g.GitlabConnect(*token)
	if err != nil {
		return internGitlab.NoPermissions, err
	}

	endpoint := fmt.Sprintf("groups/%s/members/all/%d", internGitlab.PathEscape(group), user.GitlabUserID)
	req, err := client.NewRequest(http.MethodGet, endpoint, nil, []internGitlab.RequestOptionFunc{internGitlab.WithContext(ctx)})
	if err != nil {
		return internGitlab.NoPermissions, err
	}

	member := new(internGitlab.GroupMember)
	resp, err := client.Do(req, member)
	if respErr := checkResponse(resp); respErr != nil {
		return internGitlab.NoPermissions, respErr
	}
	if err != nil {
		return internGitlab.NoPermissions, err
	}

	return member.AccessLevel, nil
}
