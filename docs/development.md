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

[`go.mod`](../go.mod) declares Go 1.25. The bootstrap accepts an installed Go
toolchain with minor version 1.25 or newer. If none is available, it downloads
the pinned Go 1.26.6 archive for the supported operating system and architecture
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

## Repository checks

The repository checks follow the platform-catalog
`docs/standards/pre-commit-standard.md`. One
[`.pre-commit-config.yaml`](../.pre-commit-config.yaml) runs as the local
hook and as the CI job `higiene`: secrets (gitleaks), merge conflict
markers, large files, private keys, case conflicts, the
[`.editorconfig`](../.editorconfig) format, `.env` files (`sem-env`),
Conventional Commits messages (`commit-msg` stage, local only), Markdown
style and relative links, shell scripts (syntax, ShellCheck, executable bit,
shdoc header), YAML (`yamllint --strict`) and workflows (actionlint). The
shared hooks come from
[`assembleiarr/ci-workflows`](https://github.com/assembleiarr/ci-workflows),
pinned by tag. Go checks are not hooks: they stay in the Go jobs of [CI](#ci).

Install pre-commit once per machine (any version at or above the
`minimum_pre_commit_version` of the configuration; CI pins the one in
[`.github/ci-requirements.txt`](../.github/ci-requirements.txt)), then the
hooks once per clone:

```bash
pipx install pre-commit   # or the pre-commit a platform-modules controller installs
pre-commit install        # pre-commit and commit-msg hooks
pre-commit run --all-files
```

Run `pre-commit run --all-files` before every push. The ShellCheck and
lychee hooks run in containers, so they need a `docker` command that reaches
a running engine (Docker, Podman with `podman-docker`, or Colima on macOS
with the clone under `$HOME`). `git commit --no-verify` skips the hook, not
CI.

Every shell entrypoint (mode `100755`) starts with a shdoc header: `@file`,
`@brief`, `@description` and `@example`. `scripts/release-build.sh` and
`scripts/e2e-shared-root.sh` print their `@description` block as `--help`.

The exceptions, each with its reason in
[#57](https://github.com/ednillibanio/platformctl/issues/57):
`detect-private-key` skips `internal/sharedroot/keys.go` (the key parser),
editorconfig-checker skips `testdata/` (byte-exact fixtures), `.editorconfig`
does not check the indentation of `*.go` (gofmt owns it; a gofmt check in CI
is [#56](https://github.com/ednillibanio/platformctl/issues/56)), and
[`.gitleaksignore`](../.gitleaksignore) lists findings in the history that
are not secrets.

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
schemas, the committed example, and the shared-root commands
(`internal/sharedroot`: names, atomic no-clobber writes, symlink refusal,
parsers, golden skeletons, concurrency, and one test per `doctor` check).
Several shared-root tests check file ownership and modes, so run the suite
both as a regular user and as root before a pull request.

## Shared-root end-to-end check

`scripts/e2e-shared-root.sh` builds the binary (or takes `--bin <path>`) and
runs `bootstrap-config`, `credential` and `doctor` against a scratch root,
never `~/.config/assembleiarr`. It checks that a second `bootstrap-config`
changes nothing, that an invalid declaration writes nothing, that a
divergent file is reported and not overwritten, that `credential` never
overwrites, that `doctor` accepts the filled tree, fails on `$(...)`, an
unquoted `~`, a duplicate key, a missing declared key and a file directly
under `credentials/target-<t>/`, and that its output holds no secret. It
prints one `PASS`/`FAIL` line per check and exits `1` on any failure. It
needs Bash and `python3` or `jq` (to parse the `doctor` JSON); the pinned
`golang` image has `python3`.

It uses only the neutral declaration
`scripts/testdata/e2e/shared-root.yaml` (consumer `provision-example`).
Whether a consumer's own loader accepts a tree that `platformctl` created is
tested in that consumer's repository, not here.

```bash
docker run --rm -v "$PWD":/src:Z -w /src -e HOME=/tmp \
  -e GOPATH=/src/.cache/gopath docker.io/library/golang:1.26.6 \
  ./scripts/e2e-shared-root.sh
```

## Release build

`.github/workflows/release.yml` runs the tests and
`scripts/release-build.sh`, which writes `platformctl-<os>-<arch>` for
`linux`/`darwin` × `amd64`/`arm64` (`CGO_ENABLED=0`, version embedded with
`-ldflags`) and `SHA256SUMS`. On a pull request or a manual run it only
uploads them as a workflow artifact; a `vX.Y.Z` tag also publishes a GitHub
release. Tags are created on `main` after the `lab` → `main` promotion, with
the operator's confirmation. Local run:

```bash
docker run --rm -v "$PWD":/src:Z -w /src -e HOME=/tmp \
  -e GOPATH=/src/.cache/gopath docker.io/library/golang:1.26.6 \
  ./scripts/release-build.sh dev-"$(git rev-parse --short=12 HEAD)" .cache/dist
```

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

1. The `higiene` job runs `pre-commit run --all-files --show-diff-on-failure`
   with the pinned pre-commit (see [Repository checks](#repository-checks)).
2. The `gitleaks` job scans the whole history (`fetch-depth: 0`); the hook
   scans only the commit.
3. Ubuntu, macOS, and Windows jobs bootstrap, vet, test, and build with Go
   1.26.x.
4. A Linux job runs both static lab contracts.
5. The same Linux job builds and runs the container lab, inspects its persisted
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
pre-commit run --all-files
```

`pre-commit` checks Markdown style and relative links. Also verify that every
repository path named in the text exists and that examples contain no real
tokens, passwords, keys, internal production
addresses, or unsupported behavior claims.
