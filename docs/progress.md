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

## Current validation

Local Linux validation passes for formatting, vet, tests, example rendering,
bootstrap from a machine without Go, and Linux/Windows builds. The Vagrant
contract passes without requiring Vagrant.

## Pending validation

- Run the GitHub Actions Linux and Windows matrix.
- Run the PowerShell bootstrap on Windows.
- Execute `vagrant validate`, create `platformctl-test` with both supported
  providers where available, and inspect the persisted ready status.

## Deferred product work

- Runtime orchestration of `infra-gitops` and `provision-*`.
- Real provisioning through Ansible, Terraform, govc, Vagrant, or glpictl.
- Catalog import, Vault integration, daemon/API, and graphical interfaces.
- Release packaging, signed checksums, upgrade policy, and stable exit codes.
