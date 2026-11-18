#!/usr/bin/env bash
# @file container/lab.sh
# @brief Vagrant-like lifecycle commands for the platformctl container lab.
# @description
#   Every command is repeatable and idempotent: `up` re-runs the same
#   provisioning script each time (fast on a warm lab-home volume), and
#   `destroy` always returns to a clean slate. Run without arguments for the
#   list of commands.
# @example
#   cd container && ./lab.sh up && ./lab.sh status && ./lab.sh destroy
# @arg $1 string Command: up, status, render, ssh or destroy.

set -Eeuo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${script_dir}"

status_path="/home/lab/.local/state/platformctl-lab/status"
rendered_dir="/home/lab/.local/share/platformctl-dev/rendered"

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}

usage() {
  cat <<'EOF'
Usage: ./lab.sh <command>

Commands:
  up        Build the image and run the full validation pipeline (like `vagrant up`)
  status    Print the persisted validation status
  render    Print the rendered YAML output
  ssh       Open an interactive shell inside the lab (like `vagrant ssh`)
  destroy   Tear down the lab and its volume (like `vagrant destroy -f`)
EOF
}

main() {
  local cmd="${1:-}"
  case "${cmd}" in
    up)
      compose up --build --exit-code-from platformctl-lab
      ;;
    status)
      compose run --rm --entrypoint cat platformctl-lab "${status_path}"
      ;;
    render)
      compose run --rm --entrypoint cat platformctl-lab "${rendered_dir}/lab.yaml"
      ;;
    ssh)
      compose run --rm --entrypoint bash platformctl-lab
      ;;
    destroy)
      compose down -v
      ;;
    *)
      usage
      exit 1
      ;;
  esac
}

main "$@"
