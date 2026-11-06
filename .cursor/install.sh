#!/usr/bin/env bash
# Runs on every Build and must stay idempotent. Only disk state (including
# /var/lib/docker) survives, so the stack is booted once here and left stopped:
# agents then skip GitLab's multi-minute first boot.
set -euo pipefail

cd "$(dirname "$0")/.."

image_go="$(GOTOOLCHAIN=local go env GOVERSION)"
repo_go="go$(go mod edit -json | jq -r .Go)"
if [ "$image_go" != "$repo_go" ]; then
    echo "WARNING: image has $image_go but go.mod requires $repo_go; update GO_VERSION in .cursor/Dockerfile." >&2
fi

image_node="$(node --version)"
repo_node="v$(tr -d '[:space:]' < .nvmrc)"
if [ "$image_node" != "$repo_node" ]; then
    echo "WARNING: image has Node $image_node but .nvmrc requires $repo_node; update NODE_VERSION in .cursor/Dockerfile." >&2
fi

.cursor/start.sh

make dist

(
    cd e2e/playwright
    npm ci
    npx playwright install --with-deps chromium
)

if make e2e-env-up; then
    (cd e2e/playwright && docker compose stop)
else
    echo "WARNING: the E2E stack did not become healthy during the build; agents will boot it from scratch." >&2
    (cd e2e/playwright && docker compose logs --no-color --tail 200 && docker compose down -v) || true
fi
