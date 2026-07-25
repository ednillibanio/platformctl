#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEV_ROOT="${PLATFORMCTL_DEV_ROOT:-${PROJECT_ROOT}}"
LOCAL_GO="${DEV_ROOT}/.tools/go/bin/go"

if [[ -x "${LOCAL_GO}" ]]; then
  GO_BINARY="${LOCAL_GO}"
elif command -v go >/dev/null 2>&1; then
  GO_BINARY="$(command -v go)"
else
  echo "[ERROR] Go is not available. Run ./scripts/bootstrap-dev.sh first." >&2
  exit 1
fi

export GOCACHE="${GOCACHE:-${DEV_ROOT}/.cache/go-build}"
export GOPATH="${GOPATH:-${DEV_ROOT}/.cache/gopath}"
mkdir -p "${GOCACHE}" "${GOPATH}"

exec "${GO_BINARY}" "$@"
