# Include custom targets and environment variables here

E2E_DIR = e2e/playwright

$(E2E_DIR)/node_modules: $(E2E_DIR)/package.json $(E2E_DIR)/package-lock.json
	cd $(E2E_DIR) && npm ci && npx playwright install chromium
	touch $@

## Starts the disposable Mattermost and GitLab servers used by the E2E suite.
.PHONY: e2e-env-up
e2e-env-up:
	$(E2E_DIR)/scripts/ensure-hosts.sh
	cd $(E2E_DIR) && docker compose up -d --wait

## Stops the E2E servers and wipes their data.
.PHONY: e2e-env-down
e2e-env-down:
	cd $(E2E_DIR) && docker compose down -v

## Builds the plugin and runs the Playwright E2E suite against PW_BASE_URL (default http://mattermost:8065).
.PHONY: e2e
e2e: dist $(E2E_DIR)/node_modules
	cd $(E2E_DIR) && npm run tsc && npm test

## Generates mock golang interfaces for testing
.PHONY: mock
mock:
ifneq ($(HAS_SERVER),)
	go get go.uber.org/mock/mockgen
	mockgen -destination server/mocks/mock_gitlab.go github.com/mattermost/mattermost-plugin-gitlab/server/gitlab Gitlab
	mockgen -destination server/gitlab/mocks/mock_gitlab.go github.com/mattermost/mattermost-plugin-gitlab/server/gitlab Gitlab
endif
