#!/usr/bin/env bash

# Provision and verify the disposable platformctl test VM.

set -Eeuo pipefail

project_root="${PLATFORMCTL_PROJECT_ROOT:-/home/vagrant/platformctl}"
dev_root="${PLATFORMCTL_DEV_ROOT:-/home/vagrant/.local/share/platformctl-dev}"
state_dir="${PLATFORMCTL_LAB_STATE_DIR:-/home/vagrant/.local/state/platformctl-lab}"
status_file="${state_dir}/status"
render_dir="${dev_root}/rendered"
current_step="initialization"

log() {
  printf '[platformctl-lab] %s\n' "$*"
}

write_status() {
  local status="$1" reason="$2" result_code="${3:-0}"
  local status_tmp="${status_file}.tmp"

  mkdir -p "${state_dir}"
  {
    printf 'status=%s\n' "${status}"
    printf 'reason=%s\n' "${reason}"
    printf 'result_code=%s\n' "${result_code}"
    printf 'updated_at=%s\n' "$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
    printf 'project_root=%s\n' "${project_root}"
    printf 'dev_root=%s\n' "${dev_root}"
  } >"${status_tmp}"
  chmod 0600 "${status_tmp}"
  mv -f "${status_tmp}" "${status_file}"
}

handle_error() {
  local result_code=$?

  trap - ERR
  log "Provisioning failed during ${current_step} with code ${result_code}."
  write_status failed "${current_step}" "${result_code}"
  exit "${result_code}"
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    log "Required base-box command is unavailable: $1"
    return 1
  }
}

main() {
  trap handle_error ERR

  current_step="checkout_validation"
  [[ -f "${project_root}/go.mod" ]] || {
    log "Mounted platformctl checkout is unavailable at ${project_root}."
    return 1
  }
  require_command bash
  require_command curl
  require_command sha256sum
  require_command tar

  export PLATFORMCTL_DEV_ROOT="${dev_root}"
  mkdir -p "${render_dir}"
  cd "${project_root}"

  current_step="development_bootstrap"
  ./scripts/bootstrap-dev.sh

  current_step="go_vet"
  ./scripts/go.sh vet ./...

  current_step="go_test"
  ./scripts/go.sh test ./...

  current_step="linux_build"
  ./scripts/go.sh build -o "${dev_root}/bin/platformctl" ./cmd/platformctl

  current_step="example_validation"
  PLATFORMCTL_HOME="${project_root}/examples/platform-home" \
    "${dev_root}/bin/platformctl" validate --env lab

  current_step="yaml_render"
  PLATFORMCTL_HOME="${project_root}/examples/platform-home" \
    "${dev_root}/bin/platformctl" render --env lab >"${render_dir}/lab.yaml"

  current_step="json_render"
  PLATFORMCTL_HOME="${project_root}/examples/platform-home" \
    "${dev_root}/bin/platformctl" render --env lab --format json >"${render_dir}/lab.json"

  trap - ERR
  write_status ready validation_completed
  log "Validation completed. Status: ${status_file}"
  log "Rendered examples: ${render_dir}"
}

main "$@"
