# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Always invoke Go through the wrapper scripts, not `go` directly — they pin
`GOCACHE`/`GOPATH` under the ignored `.cache/` directory and prefer a
bootstrapped toolchain under `.tools/` if one exists.

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

CI (`.github/workflows/ci.yml`) runs on ubuntu/macos/windows: bootstrap, then
`vet`, `test ./...`, and `build ./cmd/platformctl`.

Manual CLI smoke test:

```bash
platformctl init
platformctl environment init lab
platformctl validate --env lab
platformctl render --env lab            # yaml
platformctl render --env lab --format json
```

`PLATFORMCTL_HOME` overrides the configuration root used by `init`/`validate`/
`render`; otherwise it resolves to the OS user config directory (see
`internal/platformhome/home.go`).

## Architecture

`platformctl` is a configuration-only control plane for infrastructure
environments: it creates, validates, merges, and renders operator
configuration but never invokes Vagrant, Ansible, Terraform, or any
provisioner itself. It's intentionally a separate concern from
`infrastructure-platform` (declarative catalog), `infra-gitops` (automation
library), `provision-*` repos, and service CLIs like `glpictl` — those are
future consumers of the rendered model, not dependencies of this repo.

### Pipeline

Commands in `internal/cli` (built on cobra) all funnel through the same
sequence, and rendering stops immediately if any stage fails:

1. **`internal/config`** — strict YAML parsing (`yaml.KnownFields(true)`,
   single-document only) into typed docs (`Defaults`, `Environment`,
   `Profile`, `Service`, `Host`), assembled into a `Bundle`. Filenames must
   match `metadata.name`; `internal/config/scaffold.go` owns `init` /
   `environment init` templating and enforces `0700`/`0600` permissions.
2. **`internal/validation`** — semantic and cross-document checks over a
   `Bundle`: apiVersion/kind headers, credential shape (exactly one of
   `env`/`file`), profile inheritance cycles, port ranges, and every
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
   lists. `Encode` serializes to YAML or JSON.
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

This tree lives outside the repo and is never committed. Credentials are only
ever *references* (`SecretRef{Env|File}`) — platformctl never reads or prints
the underlying secret values.

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
