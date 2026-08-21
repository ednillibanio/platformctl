# Development

This guide is for maintainers changing `platformctl`. Operators who only need
to build and install the current CLI should use [Installation](installation.md).

## Development contract

- `main` is the principal/default branch. Base topic work on the current `lab`
  integration branch, not directly on `main`.
- Merge reviewed topic work into `lab`; after integration validation, promote
  `lab` to `main` through a separate reviewed pull request.
- Run Go exclusively through `scripts/go.sh` on Linux/macOS or `scripts/go.ps1`
  on Windows.
- Keep operator configuration outside the checkout.
- Do not commit `.cache/`, `.tools/`, lab-local configuration, renders, or real
  credentials.
- Keep Go code, CLI help, errors, logs, and public documentation in English.
  The operational planning files under `iterations/` use Portuguese.
- Follow the approval gates in [`iterations/README.md`](../iterations/README.md)
  when work is managed as an iteration.

## Toolchain

[`go.mod`](../go.mod) declares Go 1.24. The bootstrap accepts an installed Go
toolchain with minor version 1.24 or newer. If none is available, it downloads
the pinned Go 1.26.5 archive for the supported operating system and architecture
into the ignored `.tools/` directory and verifies the repository-pinned SHA-256
checksum.

The wrappers set default `GOCACHE` and `GOPATH` paths under `.cache/`. Override
the complete development root with `PLATFORMCTL_DEV_ROOT` when the checkout is
read-only, as the disposable labs do.

| Interface | Purpose | Default when unset |
| --- | --- | --- |
| `PLATFORMCTL_DEV_ROOT` | Root for `.tools/` and `.cache/`. | Repository root |
| `GOCACHE` | Go build cache. | `<dev-root>/.cache/go-build` |
| `GOPATH` | Go workspace/module cache. | `<dev-root>/.cache/gopath` |

Bootstrap from the repository root:

```bash
./scripts/bootstrap-dev.sh
```

```powershell
.\scripts\bootstrap-dev.ps1
```

The bootstrap may access `go.dev` and the Go module proxy. Successful output
ends with `[PASS] Development environment is ready.`

## Build and test

Linux and macOS:

```bash
./scripts/go.sh vet ./...
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
```

Windows PowerShell:

```powershell
.\scripts\go.ps1 vet ./...
.\scripts\go.ps1 test ./...
.\scripts\go.ps1 build -o bin\platformctl.exe .\cmd\platformctl
```

Run one package or test while iterating:

```bash
./scripts/go.sh test ./internal/render/...
./scripts/go.sh test ./internal/render/ -run TestResolve
```

`go test ./...` covers the CLI workflow, strict loading, scaffold safety,
platform-home resolution, immutable merge, validation, render/redaction,
schemas, and the committed example.

## Manual CLI smoke test

Build the binary first, then validate the project-owned example without
modifying real operator configuration:

```bash
PLATFORMCTL_HOME="${PWD}/examples/platform-home" \
  ./bin/platformctl validate --env lab

PLATFORMCTL_HOME="${PWD}/examples/platform-home" \
  ./bin/platformctl render --env lab > /tmp/platformctl-lab.yaml

PLATFORMCTL_HOME="${PWD}/examples/platform-home" \
  ./bin/platformctl render --env lab --format json > /tmp/platformctl-lab.json
```

The example deliberately contains credential references but no credential
values. Remove the temporary renders when they are no longer needed.

## Static lab contracts

These checks require Bash but not Vagrant or Docker:

```bash
bash vagrant/tests/test-vagrantfile-contract.sh
bash container/tests/test-compose-contract.sh
```

They verify isolation and lifecycle invariants, including read-only source
mounting in the container, external lab artifacts, non-root execution, and the
absence of dependencies on adjacent repositories.

## Full disposable labs

Both labs call
[`vagrant/scripts/provision-test-vm.sh`](../vagrant/scripts/provision-test-vm.sh)
and run bootstrap → vet → test → build → example validation → YAML/JSON render.
They do not exercise real operator data or provision infrastructure.

The shared script accepts these advanced path overrides; Compose sets all three
explicitly and the Vagrant defaults target the `vagrant` user's home:

| Variable | Purpose | Vagrant-script default |
| --- | --- | --- |
| `PLATFORMCTL_PROJECT_ROOT` | Mounted source checkout. | `/home/vagrant/platformctl` |
| `PLATFORMCTL_DEV_ROOT` | External toolchain, caches, binary, and renders. | `/home/vagrant/.local/share/platformctl-dev` |
| `PLATFORMCTL_LAB_STATE_DIR` | Persisted status directory. | `/home/vagrant/.local/state/platformctl-lab` |

These are development-lab controls, not operator configuration. Do not point
them at `PLATFORMCTL_HOME`.

Container lab:

```bash
cd container
cp .env.example .env
./lab.sh up
./lab.sh status
./lab.sh render
./lab.sh destroy
```

Vagrant lab:

```bash
cd vagrant
cp config.local.example.yaml config.local.yaml
vagrant validate
vagrant up
vagrant ssh -c 'cat ~/.local/state/platformctl-lab/status'
vagrant destroy -f
```

The final destroy commands remove the disposable lab state. Review
[Lab environments](lab-environments.md) and the lab-specific READMEs before
running them.

## CI

[`.github/workflows/ci.yml`](../.github/workflows/ci.yml) runs on pushes and
pull requests:

1. Ubuntu, macOS, and Windows jobs bootstrap, vet, test, and build with Go
   1.26.x.
2. A Linux job runs both static lab contracts.
3. The same Linux job builds and runs the container lab, inspects its persisted
   status, and destroys it even after failure.

CI does not run a Vagrant VM because that requires an external hypervisor and
provider-specific environment.

## Repository layout

| Path | Responsibility |
| --- | --- |
| `cmd/platformctl/` | Process startup only |
| `internal/cli/` | Cobra commands and terminal I/O |
| `internal/config/` | Typed documents, parsing, and scaffolding |
| `internal/validation/` | Semantic and cross-document validation |
| `internal/merge/` | Immutable recursive map merge |
| `internal/render/` | Resolution, redaction, YAML/JSON encoding |
| `internal/platformhome/` | Cross-platform configuration-root policy |
| `internal/schema/`, `schemas/` | Embedded public JSON Schemas |
| `examples/platform-home/` | Committed integration fixture |
| `container/`, `vagrant/` | Disposable Linux validation labs |
| `docs/` | Operator, integrator, and maintainer documentation |
| `iterations/` | Approval-gated operational plans and evidence |

## Documentation checks

Documentation changes should be checked against executable help, source, and
schemas. At minimum:

```bash
./bin/platformctl --help
./bin/platformctl render --help
./scripts/go.sh test ./...
git diff --check
```

Also verify that every relative Markdown link and repository path resolves and
that examples contain no real tokens, passwords, keys, internal production
addresses, or unsupported behavior claims.
