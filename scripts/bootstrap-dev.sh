#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GO_VERSION="1.26.5"
DEV_ROOT="${PLATFORMCTL_DEV_ROOT:-${PROJECT_ROOT}}"
TOOLS_DIR="${DEV_ROOT}/.tools"
LOCAL_GO="${TOOLS_DIR}/go/bin/go"

system_go_is_compatible() {
  local version minor
  command -v go >/dev/null 2>&1 || return 1
  version="$(go env GOVERSION 2>/dev/null || true)"
  [[ "${version}" =~ ^go1\.([0-9]+) ]] || return 1
  minor="${BASH_REMATCH[1]}"
  ((minor >= 24))
}

download_go() {
  local arch archive checksum temp_dir
  case "$(uname -m)" in
    x86_64|amd64)
      arch="amd64"
      checksum="5c2c3b16caefa1d968a94c1daca04a7ca301a496d9b086e17ad77bb81393f053"
      ;;
    aarch64|arm64)
      arch="arm64"
      checksum="fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49"
      ;;
    *)
      echo "[ERROR] Unsupported Linux architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac

  command -v curl >/dev/null 2>&1 || {
    echo "[ERROR] curl is required to bootstrap Go." >&2
    exit 1
  }
  command -v sha256sum >/dev/null 2>&1 || {
    echo "[ERROR] sha256sum is required to verify Go." >&2
    exit 1
  }

  archive="go${GO_VERSION}.linux-${arch}.tar.gz"
  temp_dir="$(mktemp -d)"
  trap 'rm -rf "${temp_dir}"' RETURN
  curl --fail --location --silent --show-error \
    "https://go.dev/dl/${archive}" \
    --output "${temp_dir}/${archive}"
  printf '%s  %s\n' "${checksum}" "${temp_dir}/${archive}" | sha256sum --check -
  mkdir -p "${TOOLS_DIR}"
  rm -rf "${TOOLS_DIR}/go"
  tar -xzf "${temp_dir}/${archive}" -C "${TOOLS_DIR}"
}

if [[ ! -x "${LOCAL_GO}" ]] && ! system_go_is_compatible; then
  echo "[INFO] Installing Go ${GO_VERSION} under ${TOOLS_DIR}"
  download_go
fi

"${SCRIPT_DIR}/go.sh" version
"${SCRIPT_DIR}/go.sh" mod download
echo "[PASS] Development environment is ready."
