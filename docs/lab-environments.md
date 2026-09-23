# Lab environments

platformctl ships two disposable, self-owned Linux lab environments for
validating the project outside any service-provisioning repository. Both run
the identical contract — local Go bootstrap, `go vet`, `go test`, a Linux
build, and validate/render against
[`examples/platform-home`](../examples/platform-home) — via the same shared
script, [`vagrant/scripts/provision-test-vm.sh`](../vagrant/scripts/provision-test-vm.sh).
Pick whichever mechanism fits your host.

| | [`vagrant/`](../vagrant/README.md) | [`container/`](../container/README.md) |
| --- | --- | --- |
| Mechanism | VM (VirtualBox or VMware Desktop) | Docker Compose |
| Apple Silicon | Provider support limited or unavailable | Native `linux/arm64`, no emulation |
| Isolation | VM user's home | Named Docker volume |
| Spin-up | `vagrant up` | `docker compose up --build` |
| Teardown | `vagrant destroy -f` | `docker compose down -v` |

Both intentionally validate committed examples rather than private operator
configuration, and neither depends on `platform-modules`, `provision-*`, or any
other adjacent repository.

## Shared validation contract

Both mechanisms execute, in order:

1. development bootstrap with checksum verification;
2. `go vet ./...`;
3. `go test ./...`;
4. Linux binary build;
5. `validate --env lab` against `examples/platform-home`;
6. YAML and JSON renders of that example;
7. persisted `status=ready` only after every stage succeeds.

Source is mounted into the lab, but the toolchain, caches, binary, status, and
renders remain outside the checkout. A failure persists the failing stage and
result code for diagnosis.

## Configuration inventory

Container configuration is read from ignored `container/.env` by Compose:

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `PLATFORMCTL_LAB_BASE_IMAGE` | No | `ubuntu:24.04` | Dockerfile base image. |
| `PLATFORMCTL_LAB_CPUS` | No | `2` | Compose CPU limit. |
| `PLATFORMCTL_LAB_MEMORY` | No | `2g` | Compose memory limit. |

Vagrant configuration is read from ignored `vagrant/config.local.yaml`, with
environment variables taking precedence:

| YAML key | Environment override | Default | Constraint |
| --- | --- | --- | --- |
| `box` | `PLATFORMCTL_VAGRANT_BOX` | `bento/ubuntu-24.04` | Non-empty, no newlines. |
| `memory` | `PLATFORMCTL_VM_MEMORY` | `2048` | Positive integer MiB. |
| `cpus` | `PLATFORMCTL_VM_CPUS` | `2` | Positive integer. |
| `provider` | `PLATFORMCTL_VAGRANT_PROVIDER` | `vmware_desktop` | `virtualbox` or `vmware_desktop`. |
| `ssh_port` | `PLATFORMCTL_TEST_SSH_PORT` | `2266` | Integer `1024..65535`, bound to loopback. |

`PLATFORMCTL_VAGRANT_CONFIG` may select a different Vagrant YAML file path;
the default is `vagrant/config.local.yaml`. Unknown YAML keys are rejected.
Start from the checked-in `.example` files and never add credentials: neither
lab needs them.

## Safety and cleanup

- The container checkout mount is read-only and the lab runs as a non-root
  user without the Docker socket.
- The VM disables the default `/vagrant` mount, mounts only this checkout, does
  not forward the SSH agent, and exposes SSH only on `127.0.0.1`.
- `./lab.sh destroy` removes the container lab and its named `lab-home` volume.
- `vagrant destroy -f` removes the disposable VM.
- Neither cleanup command touches the operator's external configuration home.

Review the mechanism-specific README before executing cleanup:
[container](../container/README.md) or [Vagrant](../vagrant/README.md).

This page is about the disposable validation labs only. It is unrelated to
the live `PLATFORMCTL_HOME` mount feature described in
[Vagrant controller integration](vagrant-controller-integration.md), which
lets an external service or automation controller see an operator's real
configuration tree — that pattern is specific to Vagrant's `synced_folder`
and has no container equivalent today.
