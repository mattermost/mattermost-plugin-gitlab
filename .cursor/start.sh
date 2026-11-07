#!/usr/bin/env bash
# Starts Docker and maps the hostnames shared by the browser and the E2E containers.
set -euo pipefail

if ! grep -qE '^127\.0\.0\.1[[:space:]]+mattermost gitlab\.test$' /etc/hosts; then
    echo "127.0.0.1 mattermost gitlab.test" | sudo tee -a /etc/hosts >/dev/null
fi

if ! sudo docker info >/dev/null 2>&1; then
    sudo service docker start
fi

for _ in $(seq 1 30); do
    docker info >/dev/null 2>&1 && exit 0
    sleep 1
done

echo "Docker did not become ready" >&2
exit 1
