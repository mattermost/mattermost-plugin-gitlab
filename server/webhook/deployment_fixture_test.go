// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package webhook

const DeploymentEventRunning = `{
	"object_kind": "deployment",
	"status": "running",
	"project": {
		"id": 24,
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
	"user": {
		"username": "testuser"
	},
	"deployable_url": "http://localhost:3000/myorg/myrepo/deployment/123",
	"status": "running"
}`

const DeploymentEventSuccessful = `{
	"object_kind": "deployment",
	"status": "success",
	"project": {
		"id": 24,
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
	"user": {
		"username": "testuser"
	},
	"deployable_url": "http://localhost:3000/myorg/myrepo/deployment/456",
	"status": "success"
}`

const DeploymentEventFailed = `{
	"object_kind": "deployment",
	"status": "failed",
	"project": {
		"id": 24,
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
	"user": {
		"username": "testuser"
	},
	"deployable_url": "http://localhost:3000/myorg/myrepo/deployment/789",
	"status": "failed"
}`

const DeploymentEventWithoutAction = `{
	"object_kind": "deployment",
	"status": "",
	"project": {
		"id": 24,
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
	"user": {
		"username": "testuser"
	},
	"deployable_url": "http://localhost:3000/myorg/myrepo/deployment/000",
	"status": ""
}`
