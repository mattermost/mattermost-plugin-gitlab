# Include custom targets and environment variables here

## Generates mock golang interfaces for testing
.PHONY: mock
mock:
ifneq ($(HAS_SERVER),)
	go get go.uber.org/mock/mockgen
	mockgen -destination server/mocks/mock_gitlab.go github.com/mattermost/mattermost-plugin-gitlab/server/gitlab Gitlab
	mockgen -destination server/gitlab/mocks/mock_gitlab.go github.com/mattermost/mattermost-plugin-gitlab/server/gitlab Gitlab
endif
