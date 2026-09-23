#!/usr/bin/env bash

# Verify container lab lifecycle and isolation invariants without requiring Docker.

set -euo pipefail

container_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${container_root}/docker-compose.yml"
dockerfile="${container_root}/Dockerfile"

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

require_text() {
  local file="$1" pattern="$2" description="$3"
  grep -Eq "${pattern}" "${file}" || fail "Missing contract: ${description}"
}

reject_text() {
  local file="$1" pattern="$2" description="$3"
  if grep -Eq "${pattern}" "${file}"; then
    fail "Forbidden contract: ${description}"
  fi
}

require_text "${compose_file}" '^\s*platformctl-lab:' 'the service has a stable name'
require_text "${compose_file}" '\.\.:/workspace:ro' 'the checkout is mounted read-only at /workspace'
require_text "${compose_file}" 'lab-home:/home/lab' 'the lab home is a named, disposable volume'
require_text "${compose_file}" '^volumes:' 'a top-level volumes section declares the named volume'
require_text "${dockerfile}" 'ENTRYPOINT.*vagrant/scripts/provision-test-vm\.sh' \
  'provisioning reuses the shared Vagrant lab script instead of duplicating it'
require_text "${dockerfile}" '^USER lab' 'the lab runs as a non-root user'

reject_text "${compose_file}" 'privileged:[[:space:]]*true' \
  'the lab container must not run privileged'
reject_text "${compose_file}" '/var/run/docker\.sock' \
  'the lab container must not receive the Docker socket'
reject_text "${compose_file}" 'infra-gitops|platform-modules|provision-glpi|glpictl' \
  'the lab must not depend on adjacent repositories'
reject_text "${dockerfile}" 'infra-gitops|platform-modules|provision-glpi|glpictl' \
  'the lab must not depend on adjacent repositories'

printf '[PASS] Container lab contract tests completed.\n'
