# Architecture

## Repository role

`platformctl` owns the operator-side configuration control plane. It is
separate from:

- `infrastructure-platform`, the public declarative catalog and standards;
- `infra-gitops`, the reusable automation library;
- `provision-*`, service lifecycle repositories;
- service operation CLIs such as `glpictl`.

The MVP does not call or clone any of those repositories. Future integrations
must be explicit adapters around the stable rendered configuration model.
See [Ecosystem](ecosystem.md) for how these repositories hand off to each
other and in what phase each one acts.

## Components

- `cmd/platformctl` contains only process startup.
- `internal/cli` defines commands and terminal I/O.
- `internal/platformhome` resolves portable operator storage.
- `internal/config` owns typed documents, strict parsing, and scaffolding.
- `internal/validation` enforces semantic and cross-document constraints.
- `internal/merge` implements immutable layer composition.
- `internal/render` creates deterministic YAML or JSON.
- `internal/schema` exposes the public schemas embedded from `schemas/`.
- `vagrant` and `container` each own a disposable external Linux validation
  environment (VM-based and container-based, respectively). See
  [Lab environments](lab-environments.md) for how they compare.

Configuration flows through strict parsing, semantic validation, ordered merge,
reference resolution, and deterministic encoding. Rendering stops immediately
when validation fails.

Each disposable lab mounts this checkout as source but keeps its Go
toolchain, caches, binary, status, and rendered artifacts outside the
checkout — under the VM user's home for `vagrant`, under a container-local
volume for `container`. Neither consumes the controller or lifecycle of a
service-provisioning repository.

Service and automation controllers are separate consumers. In lab workflows,
they may mount the operator's live `PLATFORMCTL_HOME` directly at
`/home/vagrant/.config/platformctl`:

```text
operator PLATFORMCTL_HOME
├── <service controller>
├── <automation controller>
└── other automation or provisioning controllers
```

The direct mount is a lab transport, not an ownership transfer. The host
remains authoritative and generated artifacts stay outside the mount. This
live-mount pattern is specific to Vagrant's `synced_folder`; `container` is a
standalone validation environment only and does not currently offer an
equivalent live-mount target. See
[Vagrant controller integration](vagrant-controller-integration.md).

## Security boundary

Operational values live outside Git in the operator's configuration home.
Credentials are references to environment variables or files; platformctl does
not read, resolve, or print the referenced values. On POSIX systems, scaffolded
directories and files use modes `0700` and `0600`.
