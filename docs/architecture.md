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

## Components

- `cmd/platformctl` contains only process startup.
- `internal/cli` defines commands and terminal I/O.
- `internal/platformhome` resolves portable operator storage.
- `internal/config` owns typed documents, strict parsing, and scaffolding.
- `internal/validation` enforces semantic and cross-document constraints.
- `internal/merge` implements immutable layer composition.
- `internal/render` creates deterministic YAML or JSON.
- `internal/schema` exposes the public schemas embedded from `schemas/`.
- `vagrant` owns the disposable external Linux validation environment.

Configuration flows through strict parsing, semantic validation, ordered merge,
reference resolution, and deterministic encoding. Rendering stops immediately
when validation fails.

The Vagrant VM mounts this checkout as source but places its Go toolchain,
caches, binary, status, and rendered artifacts in the VM user's home. It does
not consume the controller or lifecycle of a service-provisioning repository.

## Security boundary

Operational values live outside Git in the operator's configuration home.
Credentials are references to environment variables or files; platformctl does
not read, resolve, or print the referenced values. On POSIX systems, scaffolded
directories and files use modes `0700` and `0600`.
