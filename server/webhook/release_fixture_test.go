// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package webhook

const ReleaseEventCreate = `{
	"object_kind": "release",
	"project": {
		"id": 1,
		"name": "myrepo",
		"namespace": "myorg",
		"web_url": "http://localhost:3000/myorg/myrepo",
		"avatar_url": null,
		"git_ssh_url": "ssh://user@localhost:2222/myorg/myrepo.git",
		"git_http_url": "http://localhost:3000/myorg/myrepo.git",
		"namespace": "myorg",
		"visibility_level": 20,
		"path_with_namespace": "myorg/myrepo",
		"default_branch": "main",
		"ci_config_path": null,
		"homepage": "http://localhost:3000/myorg/myrepo"
	},
	"name": "v1.0.0",
	"tag": "v1.0.0",
	"message": "Initial release",
	"created_at": "2024-08-08T00:00:00Z",
	"updated_at": "2024-08-08T00:00:00Z",
	"url": "http://localhost:3000/myorg/myrepo/releases/v1.0.0",
	"action": "create"
}`

const ReleaseEventUpdate = `{
	"object_kind": "release",
	"project": {
		"id": 1,
		"name": "myrepo",
		"namespace": "myorg",
		"web_url": "http://localhost:3000/myorg/myrepo",
		"avatar_url": null,
		"git_ssh_url": "ssh://user@localhost:2222/myorg/myrepo.git",
		"git_http_url": "http://localhost:3000/myorg/myrepo.git",
		"namespace": "myorg",
		"visibility_level": 20,
		"path_with_namespace": "myorg/myrepo",
		"default_branch": "main",
		"ci_config_path": null,
		"homepage": "http://localhost:3000/myorg/myrepo"
	},
	"name": "v1.1.0",
	"tag": "v1.1.0",
	"message": "Updated release",
	"created_at": "2024-08-08T00:00:00Z",
	"updated_at": "2024-08-08T00:00:00Z",
	"url": "http://localhost:3000/myorg/myrepo/releases/v1.1.0",
	"action": "update"
}`

const ReleaseEventDelete = `{
	"object_kind": "release",
	"project": {
		"id": 1,
		"name": "myrepo",
		"namespace": "myorg",
		"web_url": "http://localhost:3000/myorg/myrepo",
		"avatar_url": null,
		"git_ssh_url": "ssh://user@localhost:2222/myorg/myrepo.git",
		"git_http_url": "http://localhost:3000/myorg/myrepo.git",
		"namespace": "myorg",
		"visibility_level": 20,
		"path_with_namespace": "myorg/myrepo",
		"default_branch": "main",
		"ci_config_path": null,
		"homepage": "http://localhost:3000/myorg/myrepo"
	},
	"name": "v1.2.0",
	"tag": "v1.2.0",
	"message": "Release to be deleted",
	"created_at": "2024-08-08T00:00:00Z",
	"updated_at": "2024-08-08T00:00:00Z",
	"url": "http://localhost:3000/myorg/myrepo/releases/v1.2.0",
	"action": "delete"
}`

const ReleaseEventWithoutAction = `{
	"object_kind": "release",
	"project": {
		"id": 1,
		"name": "myrepo",
		"namespace": "myorg",
		"web_url": "http://localhost:3000/myorg/myrepo",
		"avatar_url": null,
		"git_ssh_url": "ssh://user@localhost:2222/myorg/myrepo.git",
		"git_http_url": "http://localhost:3000/myorg/myrepo.git",
		"namespace": "myorg",
		"visibility_level": 20,
		"path_with_namespace": "myorg/myrepo",
		"default_branch": "main",
		"ci_config_path": null,
		"homepage": "http://localhost:3000/myorg/myrepo"
	},
	"name": "v1.3.0",
	"tag": "v1.3.0",
	"message": "Release with no action",
	"created_at": "2024-08-08T00:00:00Z",
	"updated_at": "2024-08-08T00:00:00Z",
	"url": "http://localhost:3000/myorg/myrepo/releases/v1.3.0",
	"action": ""
}`
