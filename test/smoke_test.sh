#!/usr/bin/env bash
# Smoke E2E against the local dev backend (docs/LOCAL_DEV.md must be up).
set -euo pipefail
export SYSCHECKS_API_URL=http://localhost:8001
export SYSCHECKS_TOKEN="${SYSCHECKS_TOKEN:?set a local PAT}"
go run . whoami
go run . check list -o json | head
go run . probe http https://example.com
go run . verify --url https://example.com --expect-status 200 # exit 0
if go run . verify --url https://example.com --expect-status 599; then
  echo "verify should have failed"
  exit 1
fi
echo "smoke OK"
