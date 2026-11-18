#!/usr/bin/env bash
# @file vagrant/tests/test-vagrantfile-contract.sh
# @brief Verify Vagrant lab lifecycle and isolation invariants.
# @description
#   Static checks of vagrant/Vagrantfile and vagrant/scripts/provision-test-vm.sh;
#   does not need Vagrant. Prints [FAIL] and exits 1 on the first broken
#   contract.
# @example
#   bash vagrant/tests/test-vagrantfile-contract.sh

set -euo pipefail

vagrant_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
vagrantfile="${vagrant_root}/Vagrantfile"
provisioner="${vagrant_root}/scripts/provision-test-vm.sh"

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

require_text "${vagrantfile}" "config\\.vm\\.define 'platformctl-test', primary: true" \
  'the VM has a stable primary name'
require_text "${vagrantfile}" "machine\\.vm\\.synced_folder '\\.', '/vagrant', disabled: true" \
  'default /vagrant mount is disabled'
require_text "${vagrantfile}" "machine\\.vm\\.synced_folder PROJECT_ROOT, '/home/vagrant/platformctl'" \
  'only the platformctl checkout is mounted explicitly'
require_text "${vagrantfile}" "privileged: false" 'test suite runs as the vagrant operator'
require_text "${vagrantfile}" "host_ip: '127\\.0\\.0\\.1'" 'SSH forwarding is loopback-only'
require_text "${vagrantfile}" 'bento/ubuntu-24\.04' 'Ubuntu 24.04 is the default test box'
require_text "${vagrantfile}" "VAGRANT_DEFAULT_PROVIDER.*DEFAULT_PROVIDER" \
  'the configured provider becomes the Vagrant default'
require_text "${provisioner}" 'PLATFORMCTL_DEV_ROOT' 'VM artifacts live outside the mounted checkout'
require_text "${provisioner}" 'go\.sh vet \./\.\.\.' 'go vet is executed'
require_text "${provisioner}" 'go\.sh test \./\.\.\.' 'go tests are executed'
require_text "${provisioner}" 'render --env lab --format json' 'JSON rendering is exercised'
require_text "${provisioner}" 'write_status ready validation_completed' 'successful state is persisted'

reject_text "${vagrantfile}" 'forward_agent|id_(rsa|ed25519)|PRIVATE KEY' \
  'the test VM must not receive operator SSH identities'
reject_text "${vagrantfile}" 'infra-gitops|platform-modules|provision-glpi|glpictl' \
  'the test VM must not depend on adjacent repositories'

printf '[PASS] Vagrantfile contract tests completed.\n'
