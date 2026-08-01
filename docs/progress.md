# Implementation progress

## Completed

- Cross-platform Go CLI foundation.
- Portable operator home and idempotent scaffolding.
- Strict YAML parsing, typed schemas, semantic validation, and reference checks.
- Ordered layer resolution, profile inheritance, and host-specific overrides.
- Deterministic YAML and JSON rendering without credential expansion.
- Linux and Windows development bootstrap with pinned Go checksums.
- Unit, integration, security, example, and cross-build coverage.
- Standalone disposable Vagrant test VM contract.
- macOS development bootstrap for Intel and Apple Silicon.
- Live host configuration mount guide for Vagrant controllers.
- Standalone disposable Docker container lab contract (native `linux/arm64`,
  works on Apple Silicon without emulation).

## Current validation

Local Linux validation passes for formatting, vet, tests, example rendering,
bootstrap from a machine without Go, and Linux/Windows builds. The Vagrant
contract passes without requiring Vagrant. The container lab contract passes
without requiring Docker.

## Pending validation

- Run the GitHub Actions Linux and Windows matrix.
- Run the PowerShell bootstrap on Windows.
- Execute `vagrant validate`, create `platformctl-test` with both supported
  providers where available, and inspect the persisted ready status.
- Build and run the container lab (`docker compose up --build`), and inspect
  the persisted ready status and rendered artifacts, mirroring the Vagrant
  contract.

## Deferred product work

- Runtime orchestration of `infra-gitops` and `provision-*`.
- Real provisioning through Ansible, Terraform, govc, or glpictl, and
  orchestration of Vagrant- or container-based environments as managed
  provisioning targets — distinct from the `vagrant` and `container`
  development labs, which already exist and are not provisioning.
- Catalog import, Vault integration, daemon/API, and graphical interfaces.
- Release packaging, signed checksums, upgrade policy, and stable exit codes.
