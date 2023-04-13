# Include custom targets and environment variables here

# If there's no MM_RUDDER_PLUGINS_PROD, add DEV data
RUDDER_WRITE_KEY = 1d5bMvdrfWClLxgK1FvV3s4U1tg
ifdef MM_RUDDER_PLUGINS_PROD
  RUDDER_WRITE_KEY = $(MM_RUDDER_PLUGINS_PROD)
endif
GO_BUILD_FLAGS += -ldflags '-X "github.com/mattermost/mattermost-plugin-api/experimental/telemetry.rudderWriteKey=$(RUDDER_WRITE_KEY)"'


## Generates mock golang interfaces for testing
.PHONY: mock
mock:
ifneq ($(HAS_SERVER),)
	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -destination server/mocks/mock_gitlab.go github.com/mattermost/mattermost-plugin-gitlab/server/gitlab Gitlab
	mockgen -destination server/gitlab/mocks/mock_gitlab.go github.com/mattermost/mattermost-plugin-gitlab/server/gitlab Gitlab
endif
