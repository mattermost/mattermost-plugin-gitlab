#!/usr/bin/env bash
# Runs on every Build and must stay idempotent. Only disk state (including
# /var/lib/docker) survives, so the stack is booted once here and left stopped:
# agents then skip GitLab's multi-minute first boot.
set -euo pipefail

cd "$(dirname "$0")/.."

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
