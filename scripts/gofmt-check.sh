#!/usr/bin/env bash
# @file scripts/gofmt-check.sh
# @brief Fail when a Go file is not gofmt-formatted.
# @description
#   Runs gofmt -l from the toolchain that scripts/go.sh selects over cmd/,
#   internal/ and schemas/, prints every file that needs formatting and
#   exits 1 if there is one. It never rewrites a file: run
#   gofmt -w <file> with the same toolchain to fix it. CI runs it in the
#   Go job on Linux (#56).
# @example
#   ./scripts/gofmt-check.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

GOROOT_DIR="$("${SCRIPT_DIR}/go.sh" env GOROOT)"
GOFMT="${GOROOT_DIR}/bin/gofmt"
if [[ ! -x "${GOFMT}" ]]; then
  echo "[ERROR] gofmt not found at ${GOFMT}." >&2
  exit 1
fi

cd "${PROJECT_ROOT}"
unformatted="$("${GOFMT}" -l cmd internal schemas)"
if [[ -n "${unformatted}" ]]; then
  echo "[FAIL] Not gofmt-formatted (fix with: ${GOFMT} -w <file>):" >&2
  printf '%s\n' "${unformatted}" >&2
  exit 1
fi
echo "[PASS] gofmt: all Go files are formatted."
