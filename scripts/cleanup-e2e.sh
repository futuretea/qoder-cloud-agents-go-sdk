#!/usr/bin/env bash
set -euo pipefail

# cleanup-e2e.sh
#
# Reads e2e/.e2e-resources.jsonl and deletes/archives/cancels each recorded
# resource. The actual implementation lives in scripts/cleanup-e2e.go and is
# executed via `go run` so it can use the SDK and run on any platform supported
# by Go.
#
# Usage:
#   ./scripts/cleanup-e2e.sh [--dry-run]

if [[ -z "${QODER_PAT:-}" ]]; then
  echo "Error: QODER_PAT is not set" >&2
  exit 1
fi

if [[ "${QODER_E2E_ACK:-}" != "1" ]]; then
  echo "Error: QODER_E2E_ACK must be set to 1" >&2
  exit 1
fi

exec go run scripts/cleanup-e2e.go "$@"
