#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GO_VERSION="1.26.6"
DEV_ROOT="${PLATFORMCTL_DEV_ROOT:-${PROJECT_ROOT}}"
TOOLS_DIR="${DEV_ROOT}/.tools"
LOCAL_GO="${TOOLS_DIR}/go/bin/go"

system_go_is_compatible() {
  local version minor
  command -v go >/dev/null 2>&1 || return 1
  version="$(go env GOVERSION 2>/dev/null || true)"
  [[ "${version}" =~ ^go1\.([0-9]+) ]] || return 1
  minor="${BASH_REMATCH[1]}"
  ((minor >= 25))
}

download_go() {
  local arch archive checksum os_name temp_dir

  case "$(uname -s)" in
    Linux) os_name="linux" ;;
    Darwin) os_name="darwin" ;;
    *)
      echo "[ERROR] Unsupported operating system: $(uname -s)" >&2
      exit 1
      ;;
  esac

  case "${os_name}:$(uname -m)" in
    linux:x86_64|linux:amd64)
      arch="amd64"
      checksum="708effb774be8237570d0add163225abbdfaf4fca28b2611df167beba4feef89"
      ;;
    linux:aarch64|linux:arm64)
      arch="arm64"
      checksum="d0507e9e9d7fe012aae570108cbd76c15de879e17130ab8cb90d4d7445cb1f2e"
      ;;
    darwin:x86_64|darwin:amd64)
      arch="amd64"
      checksum="08b65a63f244115121ced6c3b55ad38d801a7442acad5c949a17aad84ae6d684"
      ;;
    darwin:arm64)
      arch="arm64"
      checksum="2dc95ce4675829f2df0e86b28bcef3283635902062a5f0580ca659bf570f3204"
      ;;
    *)
      echo "[ERROR] Unsupported ${os_name} architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac

  command -v curl >/dev/null 2>&1 || {
    echo "[ERROR] curl is required to bootstrap Go." >&2
    exit 1
  }
  if ! command -v sha256sum >/dev/null 2>&1 &&
    ! command -v shasum >/dev/null 2>&1; then
    echo "[ERROR] sha256sum or shasum is required to verify Go." >&2
    exit 1
  fi

  archive="go${GO_VERSION}.${os_name}-${arch}.tar.gz"
  temp_dir="$(mktemp -d)"
  trap 'rm -rf "${temp_dir}"' RETURN
  curl --fail --location --silent --show-error \
    "https://go.dev/dl/${archive}" \
    --output "${temp_dir}/${archive}"
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s  %s\n' "${checksum}" "${temp_dir}/${archive}" | sha256sum --check -
  else
    printf '%s  %s\n' "${checksum}" "${temp_dir}/${archive}" | shasum -a 256 --check
  fi
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
