#!/bin/sh
# Confirms the host can resolve the names shared by the browser and the containers.
set -eu

resolves() {
    name="$1"
    if command -v python3 >/dev/null 2>&1; then
        python3 -c 'import socket, sys; socket.getaddrinfo(sys.argv[1], None)' "$name" >/dev/null 2>&1
        return $?
    fi
    if command -v dscacheutil >/dev/null 2>&1; then
        dscacheutil -q host -a name "$name" | grep -q 'ip_address:'
        return $?
    fi
    getent hosts "$name" >/dev/null 2>&1
}

missing=""
for name in mattermost gitlab.test; do
    if ! resolves "$name"; then
        missing="$missing $name"
    fi
done

if [ -n "$missing" ]; then
    echo "The E2E hostnames are not resolvable:$missing" >&2
    echo "Add this line to /etc/hosts, then retry:" >&2
    echo "127.0.0.1 mattermost gitlab.test" >&2
    exit 1
fi
