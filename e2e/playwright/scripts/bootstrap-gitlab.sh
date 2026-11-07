#!/bin/sh
# Runs the Rails bootstrap inside the GitLab container and writes credentials for the specs.
set -eu

cd "$(dirname "$0")/.."

output="$(docker compose exec -T gitlab gitlab-rails runner - < scripts/bootstrap-gitlab.rb)"
json="$(printf '%s\n' "$output" | sed -n 's/^E2E_CREDENTIALS_JSON=//p' | tail -n 1)"

if [ -z "$json" ]; then
    echo "GitLab bootstrap did not print credentials." >&2
    printf '%s\n' "$output" >&2
    exit 1
fi

printf '%s\n' "$json" > .gitlab-credentials.json
echo "Wrote e2e/playwright/.gitlab-credentials.json"

runner_token="$(node -e 'process.stdout.write(require("./.gitlab-credentials.json").runner_token)')"
docker compose exec -T gitlab-runner gitlab-runner unregister --all-runners >/dev/null 2>&1 || true
docker compose exec -T gitlab-runner gitlab-runner register \
    --non-interactive \
    --url http://gitlab.test:8929 \
    --token "$runner_token" \
    --executor shell \
    --name e2e-runner
echo "Registered the e2e GitLab runner"
