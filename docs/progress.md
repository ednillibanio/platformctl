# Implementation progress

This page separates implemented capability, repository-owned automated
validation, external/manual validation, and deferred product work. It is not a
release guarantee; the public configuration API remains `v1alpha1`.

## Implemented

- Cross-platform Go CLI foundation with Cobra.
- Portable operator configuration-home resolution and absolute
  `PLATFORMCTL_HOME` overrides.
- Idempotent scaffolding with divergent-file protection and POSIX `0700`/`0600`
  permissions.
- Strict typed YAML parsing, filename identity checks, and single-document
  enforcement.
- Semantic validation and sorted, aggregated cross-document errors.
- Ordered host resolution through defaults, environment, profile inheritance,
  services, and host overrides.
- Immutable recursive map merge and shallow credential replacement.
- Deterministic YAML and JSON rendering.
- `export --format env` shell projection (ADR-0002): service or host
  selection, prefixed names, all-or-nothing credential resolution by the
  evaluating shell.
- Credential forms `env`, `file`, and inline `value`; inline values are
  redacted by default and revealed only with `render --reveal-secrets`.
- Embedded JSON Schemas for configuration and rendered environments.
- Linux/macOS/Windows development wrappers and pinned fallback Go bootstrap.
- Project-owned Vagrant and Docker Compose validation labs sharing one
  provisioning/test script.
- Live host configuration mount guidance for external Vagrant controllers.
- Operator, integrator, and maintainer documentation under `docs/`.
- Shared root of the `provision-*` repositories (iteration 004, catalog
  provision shared root standard): `bootstrap-config` (skeleton from
  `shared-root.yaml` declarations, automation key pair generated or
  imported), `credential` (random, literal, or standard-input values;
  replaces `gen-credential`), and read-only `doctor` (text and JSON, no
  secrets), on Linux and macOS; identities per ADR-0007.
- `scripts/e2e-shared-root.sh`: end-to-end check against a scratch root
  with a neutral declaration.

## Automated repository validation

The Go suite covers CLI workflow, loading, scaffolding, platform-home policy,
merge, validation, rendering/redaction, embedded schemas, and the committed
example.

The workflow in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) is
configured to run on pushes and pull requests:

- bootstrap, vet, test, and build on Ubuntu, macOS, and Windows with Go 1.26.x;
- static Vagrant and container contract tests on Ubuntu;
- full Docker Compose lab build/run/status/cleanup on Ubuntu.

The static lab tests do not require Docker, Vagrant, Ruby, or a hypervisor.

## Manual or environment-dependent validation

These checks depend on external environments and are not fully represented by
the ordinary local unit suite:

- PowerShell bootstrap behavior on a real Windows control host;
- Vagrant validation and VM creation with each available provider;
- inspection of provider-specific synced-folder behavior;
- live `PLATFORMCTL_HOME` mounts in external controller repositories;
- downstream consumer adapters and provisioning, which are not implemented;
- `scripts/e2e-shared-root.sh`, which is not run by CI yet;
- each consumer's loader against a `platformctl` tree, tested in that
  consumer's repository.

See [Lab environments](lab-environments.md) for supported lab mechanisms and
[Vagrant controller integration](vagrant-controller-integration.md) for the
separate live-mount procedure.

## Deferred product work

- A tagged release with checksums (iteration 004 step 07) and adoption of
  the shared-root commands by each `provision-*` repository (their
  adaptation issues).
- Consumer adapters for `provision-platform-environments`,
  `provision-semaphoreui`, `provision-nexus3`, and other `provision-*`
  repositories.
- Runtime orchestration of `platform-modules` or provisioning repositories.
- Adoption of the `export` projection by `platform-modules`: not planned;
  the [platform configuration alignment program](https://github.com/ednillibanio/platform-catalog/blob/main/docs/architecture/platform-config-alignment.md)
  was paused on 2026-09-23.
- Real provisioning through Ansible, Terraform, Packer, `govc`, or service
  CLIs.
- Catalog import, Vault/secrets-manager integration, daemon/API, and graphical
  interfaces.
- Release packaging, signed checksums, upgrade/migration policy, and stable
  category-specific exit codes.
- Managed Vagrant/container infrastructure targets; the existing labs validate
  this repository and are not product provisioning targets.

See [Ecosystem and integrations](ecosystem.md) for the difference between a
documented future handoff and a currently implemented integration.
