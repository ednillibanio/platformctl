# AGENTS.md

Rules for automation agents working in this repository. People start with
[`README.md`](README.md).

## Repository role

- Classification: `platform-tooling`.
- `platformctl` creates and checks the shared root of the `provision-*`
  repositories (`bootstrap-config`, `credential`, `doctor`) and holds the
  paused YAML model (`init`, `environment init`, `validate`, `render`,
  `export`).

## Scope boundaries

Belongs here:

- the Go CLI (`cmd/`, `internal/`), its embedded schemas (`schemas/`), tests
  and test fixtures, the development wrappers (`scripts/`), the disposable
  validation labs (`container/`, `vagrant/`), documentation (`docs/`) and
  iterations (`iterations/`).

Keep elsewhere:

- the shared-root contract (layout, file format, names, identities,
  commands): platform-catalog `docs/standards/provision-shared-root-standard.md`
  and ADR-0005/ADR-0007. Change the standard first; this repository follows
  it.
- each consumer's key declaration (`shared-root.yaml`) and loader: the
  `provision-*` repository that reads the values.
- real operator values and secrets: the operator's trees
  (`~/.config/assembleiarr/`, `PLATFORMCTL_HOME`), never Git.
- running infrastructure (Terraform, Ansible, Packer, `govc`, service APIs):
  `platform-modules` and the `provision-*` repositories. `platformctl` never
  runs them.

## Normative documents

- [`CONTRIBUTING.md`](CONTRIBUTING.md) and the platform-catalog
  `docs/standards/contribution-workflow.md`: issue, branch, pull request and
  merge rules.
- [`iterations/README.md`](iterations/README.md): iteration steps and their
  approval gates.
- [`docs/cli-reference.md`](docs/cli-reference.md): command behavior.
- [`docs/development.md`](docs/development.md): build, test, end-to-end check
  and CI.
- platform-catalog `docs/standards/provision-shared-root-standard.md`: the
  shared-root contract; it wins over any local document.

## Language policy

- Code, CLI help, errors, logs, `docs/`, `README.md` and this file: English.
- Files under `iterations/`: Portuguese.
- One primary language per document; never mix languages in a file.

## Editing rules

- Branch from `lab`; never commit or push to `lab` or `main` directly. Each
  iteration step, commit, pull request and merge needs explicit operator
  confirmation.
- Run Go only through `scripts/go.sh` or `scripts/go.ps1`, in the pinned Go
  container or a lab, never on the operator host. Run the tests as a regular
  user and as root, and `vet` with `GOOS=linux`, `darwin` and `windows`.
- Shared-root commands must never overwrite, truncate, move or delete an
  existing file, never follow a symbolic link inside the tree, and never
  print or log a secret value or a hash of one.
- Tests and examples use a scratch root (`--config-dir`,
  `ASSEMBLEIARR_CONFIG_DIR`, `PLATFORMCTL_HOME`), never the operator's real
  trees.
- Install the repository checks once per clone with `pre-commit install`
  (hooks for the `pre-commit` and `commit-msg` stages), and run
  `pre-commit run --all-files` before every push; CI runs the same
  configuration in the `higiene` job. Never use `git commit --no-verify` to
  get past a check. See
  [Development: Repository checks](docs/development.md#repository-checks).
- Do not commit secrets, generated binaries (`bin/`), caches (`.cache/`,
  `.tools/`) or agent records (`.agent-runs/`).
- Use `kebab-case` for Markdown file names.
- When the catalog standard changes (for example an identity or an
  environment), update the mirrored lists in `internal/sharedroot/names.go`
  and their tests in one pull request.
