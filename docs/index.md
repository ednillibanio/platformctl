# Documentation

This index routes operators, integrators, and maintainers to the authoritative
`platformctl` documentation. The repository is in the `v1alpha1` foundation
phase; releases are published from `vX.Y.Z` tags of `main`. `main` is the
principal/default branch; `lab` is the integration branch used to validate
changes before promotion.

## Choose a starting point

| Reader goal | Start here |
| --- | --- |
| Evaluate what `platformctl` does | [README](../README.md) and [Architecture](architecture.md) |
| Build and install the CLI | [Installation](installation.md) |
| Understand the `provision-*` shared root | [Shared root concepts](shared-root-concepts.md) |
| Prepare and check the shared root on a controller | [Usage guide](usage.md#shared-root-of-the-provision--repositories) |
| Write a repository's `shared-root.yaml` | [Writing a key declaration](shared-root-declaration.md) |
| Create an environment from scratch (paused YAML model) | [Usage guide](usage.md#yaml-model-paused) |
| Look up a command or flag | [CLI reference](cli-reference.md) |
| Look up YAML fields, validation, or merge precedence | [Configuration reference](configuration.md) |
| Integrate rendered output with another repository | [Ecosystem and integrations](ecosystem.md) |
| Diagnose an error | [Troubleshooting](troubleshooting.md) |
| Develop or validate a change | [Development](development.md) |
| Run an isolated Linux validation | [Lab environments](lab-environments.md) |
| Share live host configuration with a Vagrant controller | [Vagrant controller integration](vagrant-controller-integration.md) |
| Review current capabilities and deferred work | [Implementation progress](progress.md) |

## Documentation by purpose

### Learn and perform tasks

- [Installation](installation.md) installs a release on Linux and macOS, or
  builds the source on Linux, macOS, and Windows, and explains where the
  binary and configuration live.
- [Usage guide](usage.md) prepares and checks the `provision-*` shared root
  on a controller, and walks through the paused YAML model: initialization,
  manual YAML authoring, validation, and rendering.
- [Writing a key declaration](shared-root-declaration.md) guides a
  `provision-*` maintainer through `shared-root.yaml` and how to check it.
- [Lab environments](lab-environments.md) compares the disposable container
  and Vagrant validation environments.
- [Vagrant controller integration](vagrant-controller-integration.md) mounts an
  operator-owned `PLATFORMCTL_HOME` into an external controller VM.

### Look up exact behavior

- [CLI reference](cli-reference.md) documents commands, flags, configuration
  home and shared root selection, output, re-execution, and exit behavior.
- [Configuration reference](configuration.md) defines the document model,
  identifiers, credentials, validation rules, merge order, and rendered shape.
- [`schemas/v1alpha1/`](../schemas/v1alpha1/) contains the public JSON Schemas
  embedded into the binary.

### Understand design and boundaries

- [Shared root concepts](shared-root-concepts.md) explains why the shared
  root exists, its layout, who writes and reads each file, identities, and
  several controllers.
- [Architecture](architecture.md) explains the internal pipeline and component
  responsibilities.
- [Ecosystem and integrations](ecosystem.md) distinguishes current repository
  ownership from planned consumer adapters.
- [Implementation progress](progress.md) records completed, validated, and
  deferred areas.

### Maintain the repository

- [Development](development.md) covers the Go wrappers, test suites, the
  shared-root end-to-end check, CI, labs, project layout, and branch
  conventions.
- [`AGENTS.md`](../AGENTS.md) lists the rules for automation agents working in
  this repository.
- [`iterations/README.md`](../iterations/README.md) defines the controlled,
  approval-based iteration process used by this repository.
- [Pull request template](../.github/PULL_REQUEST_TEMPLATE.md) lists the
  expected validation evidence for a change.

## Authoritative sources

When documentation and implementation differ, the current implementation is
authoritative. The main evidence locations are:

| Subject | Source |
| --- | --- |
| Commands and flags | [`internal/cli/cli.go`](../internal/cli/cli.go) |
| Configuration types | [`internal/config/types.go`](../internal/config/types.go) |
| Loading and strict YAML parsing | [`internal/config/load.go`](../internal/config/load.go) |
| Scaffold behavior and permissions | [`internal/config/scaffold.go`](../internal/config/scaffold.go) |
| Semantic validation | [`internal/validation/validation.go`](../internal/validation/validation.go) |
| Merge behavior | [`internal/merge/merge.go`](../internal/merge/merge.go) |
| Render, redaction, and encoding | [`internal/render/render.go`](../internal/render/render.go) |
| Platform configuration home | [`internal/platformhome/home.go`](../internal/platformhome/home.go) |
| CI checks | [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) |

## Related repositories

`platformctl` is intentionally independent at runtime, but its rendered model
is designed for downstream automation. The integration map links to the
authoritative documentation in:

- [`al-rr/platform-modules`](https://github.com/al-rr/platform-modules)
- [`al-rr/provision-platform-environments`](https://github.com/al-rr/provision-platform-environments)
- [`al-rr/provision-semaphoreui`](https://github.com/al-rr/provision-semaphoreui)
- [`al-rr/provision-nexus3`](https://github.com/al-rr/provision-nexus3)

See [Ecosystem and integrations](ecosystem.md) before asserting that any of
these repositories consumes `platformctl`: the intended handoff is documented,
but consumer adapters are still deferred.
