# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Always invoke Go through the wrapper scripts, not `go` directly — they pin
`GOCACHE`/`GOPATH` under the ignored `.cache/` directory and prefer a
bootstrapped toolchain under `.tools/` if one exists. Per the maintainer
workflow, run them inside the `container/` or `vagrant/` lab rather than on
the operator host (see "Disposable lab environments" and "Workflow" below).

```bash
./scripts/bootstrap-dev.sh                                  # one-time setup; downloads pinned Go if none is installed
./scripts/go.sh test ./...                                  # run all tests
./scripts/go.sh test ./internal/render/...                  # test a single package
./scripts/go.sh test ./internal/render/ -run TestResolve     # run a single test
./scripts/go.sh vet ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
```

Windows uses `scripts\bootstrap-dev.ps1` / `scripts\go.ps1` with the same
argument conventions.

Static lab contract checks (Bash only, no Vagrant/Docker needed):

```bash
bash vagrant/tests/test-vagrantfile-contract.sh
bash container/tests/test-compose-contract.sh
```

CI (`.github/workflows/ci.yml`) runs on ubuntu/macos/windows: bootstrap, then
`vet`, `test ./...`, and `build ./cmd/platformctl`. A separate Linux job runs
both static lab contracts and then the full container lab (`container/lab.sh`).

Manual CLI smoke test — use the committed, secret-free fixture so the real
operator config home is never touched:

```bash
PLATFORMCTL_HOME="${PWD}/examples/platform-home" ./bin/platformctl validate --env lab
PLATFORMCTL_HOME="${PWD}/examples/platform-home" ./bin/platformctl render --env lab
PLATFORMCTL_HOME="${PWD}/examples/platform-home" ./bin/platformctl render --env lab --format json
```

`examples/platform-home/` is also exercised by `go test ./...`, so changes to
the config schema must keep it valid. `PLATFORMCTL_HOME` overrides the
configuration root and must be an absolute path; otherwise it resolves to the
OS user config directory (see `internal/platformhome/home.go`). `init` and
`environment init` write to that root, so point them at a scratch directory.

## Architecture

`platformctl` is a configuration-only control plane for infrastructure
environments: it creates, validates, merges, and renders operator
configuration but never invokes Vagrant, Ansible, Terraform, or any
provisioner itself. It's intentionally a separate concern from
`platform-catalog` (declarative catalog and ADRs), `platform-modules` (automation
library; successor to the archived `infra-gitops`), `provision-*` repos, and service CLIs like `glpictl` — those are
future consumers of the rendered model, not dependencies of this repo.

### Pipeline

Commands in `internal/cli` (built on cobra) all funnel through the same
sequence, and rendering stops immediately if any stage fails:

1. **`internal/config`** — strict YAML parsing (`yaml.KnownFields(true)`,
   single-document only) into typed docs (`Defaults`, `Environment`,
   `Profile`, `Service`, `Host`), assembled into a `Bundle`. Filenames must
   match `metadata.name`; `internal/config/scaffold.go` owns `init` /
   `environment init` templating, enforces `0700`/`0600` permissions, and
   refuses to write through symlinks (`ErrUnsafeSymlink`).
2. **`internal/validation`** — semantic and cross-document checks over a
   `Bundle`: apiVersion/kind headers, credential shape (exactly one of
   `env`/`file`/`value`), profile inheritance cycles, port ranges, and every
   cross-reference (host→profile, host→service, service port→source host,
   source host must have addresses). All problems are collected and returned
   together, sorted, rather than failing on the first one.
3. **`internal/merge`** — pure, immutable recursive map merge
   (`merge.Maps(base, overlay)`): maps merge key-by-key, everything else
   (scalars, lists, nil) is replaced wholesale by the overlay. Never mutates
   its inputs.
4. **`internal/render`** — `Resolve(bundle)` computes, per host, the layered
   merge order **defaults → environment → profile chain (root-first, via
   `extends`) → each attached service → host**, for both `values` and
   `credentials` (credentials merge is a shallow key overwrite, not deep).
   Service ports resolve `source_hosts` into deduplicated, sorted address
   lists. `Encode` serializes to YAML or JSON. `Redact` masks inline
   `SecretRef.Value` credentials (`<redacted>`); the CLI applies it by
   default and skips it only with `render --reveal-secrets`. `env`/`file`
   references are never masked since they hold no secret themselves.
5. **`internal/schema`** — exposes the JSON schemas embedded from
   `schemas/v1alpha1/` (via `schemas/embed.go`) for external validation of the
   config and rendered-environment shapes; parsing/validation logic itself
   lives in `config`/`validation`, not here.

### Config home layout

```
<PLATFORMCTL_HOME>/
├── defaults.yaml
└── environments/<env>/
    ├── environment.yaml
    ├── profiles/<name>.yaml
    ├── services/<name>.yaml
    └── hosts/<name>.yaml
```

This tree lives outside the repo and is never committed. Credentials support
`SecretRef{Env|File|Value}`. `Env` and `File` are opaque references that
platformctl never resolves; inline `Value` contains the secret itself and is
redacted by default unless `render --reveal-secrets` is explicit.

### Disposable lab environments

`vagrant/` (VM) and `container/` (Docker Compose) are two equivalent,
self-owned disposable Linux labs for validating platformctl in a real
environment; neither depends on `provision-glpi` or any other service repo.
Both mount the checkout as source but keep their own Go
toolchain/caches/binary/artifacts outside it — the VM user's home for
`vagrant/`, a named Docker volume for `container/` — by invoking the same
shared script, `vagrant/scripts/provision-test-vm.sh` (not duplicated).
`container/` runs natively on `linux/arm64` (Apple Silicon), where the
Vagrant providers (VirtualBox/VMware) have limited or no support. See
`docs/lab-environments.md` for the comparison.

Separately, `docs/vagrant-controller-integration.md` describes mounting an
operator's live `PLATFORMCTL_HOME` directly into a *service* controller VM
(e.g. `provision-glpi`) as a lab-only transport — the host stays
authoritative, and that mount is a different concern from the validation
labs above; it's specific to Vagrant's `synced_folder` and has no container
equivalent yet.

## Workflow

Read [`CONTRIBUTING.md`](CONTRIBUTING.md) before any change that will be
committed. It points to the ecosystem-wide standard
(`platform-catalog/docs/standards/contribution-workflow.md`) and lists
the local rules. In short:

- issue first (template in `.github/ISSUE_TEMPLATE/`, assigned to the
  operator) → topic branch from `lab` → one iteration step at a time, each
  gated on explicit operator confirmation → PR into `lab` with `Refs #N` →
  merge only on explicit confirmation → close the issue manually → separate
  `lab` → `main` promotion PR. Never commit directly to `lab` or `main`.
- iterations live in `iterations/NNN-topic/` (templates in
  `iterations/_templates/`, rules in `iterations/README.md`, proposals in
  `iterations/BACKLOG.md`), written in Portuguese; everything else is English.
- build/test in the `container/` or `vagrant/` labs, not on the operator host.

Cross-repository work (for example the platform configuration alignment with
`platform-modules` and `platform-labs`) is coordinated from
`platform-catalog` (`docs/architecture/platform-config-alignment.md` and
`docs/architecture/decisions/`); see the corresponding entry in
`iterations/BACKLOG.md` before starting it.
