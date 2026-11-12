# platformctl

`platformctl` is the operator-side platform tooling of the platform
repositories (platform-catalog type "Platform tooling"). It creates and
checks, on every controller, the shared configuration root that the
`provision-*` repositories read, `~/.config/assembleiarr/<env>/`, as defined
by the platform-catalog
[provision shared root standard](https://github.com/ednillibanio/platform-catalog/blob/main/docs/standards/provision-shared-root-standard.md):

| Command | Purpose |
| --- | --- |
| `platformctl bootstrap-config` | Create the skeleton of one environment from the repositories' `shared-root.yaml` declarations, and the automation key pair. |
| `platformctl credential` | Create credential files (random, literal, or from standard input). |
| `platformctl doctor` | Check the tree, read-only, in text or JSON, without printing a secret. |

None of them overwrites or deletes an existing file. They run on Linux and
macOS.

It also holds the YAML model (`init`, `environment init`, `validate`,
`render`, `export`) for authoring, validating, merging, and rendering layered
configuration by environment. That model is **paused**: the commands keep
working, but no consumer adopts it now.

`platformctl` is configuration-only. It does **not** invoke Terraform,
Ansible, Packer, Vagrant, `govc`, SemaphoreUI, or any service provisioner.

## Status

- Primary/default branch: `main`
- Integration and pre-promotion branch: `lab`
- Configuration API: `platformctl.io/v1alpha1`
- Published releases: none yet; `vX.Y.Z` tags of `main` publish Linux and macOS binaries ([Installation](docs/installation.md)); until then build from source
- Supported development hosts: Linux, macOS, and Windows (the shared-root
  commands: Linux and macOS)
- Render formats: YAML (default) and JSON
- Shell projection: `export --format env` (for example `PM_*` for
  `platform-modules`)

See [implementation progress](docs/progress.md) for completed capabilities,
validation coverage, and deferred work.

## How it fits

```text
shared-root.yaml ──> platformctl bootstrap-config ──> ~/.config/assembleiarr/<env>/
 per provision-*      credential, doctor               read by each provision-* loader
```

The paused YAML model:

```text
operator configuration                     downstream execution

PLATFORMCTL_HOME ──> platformctl render ──> provision-* / automation adapter
 values + credential      YAML or JSON        Terraform, Ansible, APIs, etc.
 references/values
```

`platformctl` owns configuration composition, not infrastructure execution.
The adapters that translate rendered data into repository-specific manifests,
inventories, or secret files are separate integration work. See
[Ecosystem and integrations](docs/ecosystem.md) for the current boundaries and
the relationships with `platform-modules`, `provision-platform-environments`,
`provision-semaphoreui`, and `provision-nexus3`.

## Quick verification from source

Prerequisites and platform-specific installation steps are in
[Installation](docs/installation.md). From the repository root:

```bash
./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
```

Validate and render the committed, secret-free example without creating or
changing an operator configuration home:

```bash
PLATFORMCTL_HOME="${PWD}/examples/platform-home" \
  ./bin/platformctl validate --env lab

PLATFORMCTL_HOME="${PWD}/examples/platform-home" \
  ./bin/platformctl render --env lab
```

Successful validation prints `Environment "lab" is valid`. The render contains
two hosts and preserves credential references without reading their values.

## First operator workflow

On a controller, create and check the shared root of one environment:

```bash
platformctl bootstrap-config --env lab --target vsphere-lab \
  --declaration ~/src/provision-example/shared-root.yaml --automation-key
# set the values in targets/vsphere-lab/vars.sh and hosts/<consumer>.sh
platformctl credential --env lab --consumer provision-example \
  EXAMPLE_DB_USER=example EXAMPLE_DB_PASS
platformctl doctor --env lab \
  --declaration ~/src/provision-example/shared-root.yaml
```

Follow [Usage: Shared root](docs/usage.md#shared-root-of-the-provision--repositories)
for the whole workflow, including more controllers of one environment.

For the paused YAML model, initialize the operating system's default
configuration location and one environment:

```bash
platformctl init
platformctl environment init lab
platformctl validate --env lab
platformctl render --env lab
```

The first two commands create private directories and managed YAML files.
Profiles, services, and hosts are then authored manually. Follow the complete
[Usage guide](docs/usage.md) before building a real environment.

`PLATFORMCTL_HOME` can select a different configuration root, but it must be an
absolute path. On POSIX systems, generated directories use mode `0700` and
generated files use `0600`.

## Documentation

Start with the [documentation index](docs/index.md), or go directly to:

| Need | Document |
| --- | --- |
| Install a release or build from source | [Installation](docs/installation.md) |
| Understand the `provision-*` shared root | [Shared root concepts](docs/shared-root-concepts.md) |
| Prepare and check the shared root on a controller | [Usage guide](docs/usage.md#shared-root-of-the-provision--repositories) |
| Write a repository's `shared-root.yaml` | [Writing a key declaration](docs/shared-root-declaration.md) |
| Create a complete environment (paused YAML model) | [Usage guide](docs/usage.md#yaml-model-paused) |
| Look up commands and flags | [CLI reference](docs/cli-reference.md) |
| Look up YAML fields and merge behavior | [Configuration reference](docs/configuration.md) |
| Understand internals and boundaries | [Architecture](docs/architecture.md) |
| Understand related repositories | [Ecosystem and integrations](docs/ecosystem.md) |
| Diagnose common failures | [Troubleshooting](docs/troubleshooting.md) |
| Develop and validate changes | [Development](docs/development.md) |
| Choose a disposable validation lab | [Lab environments](docs/lab-environments.md) |
| Mount live configuration into a Vagrant controller | [Vagrant controller integration](docs/vagrant-controller-integration.md) |

## Security boundary

Real operator configuration lives outside Git. The shared-root commands
create credentials with mode `0600`, never print a secret, and never follow
a symbolic link inside the tree. In the YAML model, a credential may contain
an environment-variable reference, a file reference, or an inline value.
Inline values are redacted by default; `render --reveal-secrets` deliberately exposes
them. `platformctl` never resolves `env` or `file` references itself.

Read [Configuration reference: Credentials](docs/configuration.md#credentials)
before adding secrets or sending rendered output to another process.

## Disposable validation labs

The repository owns equivalent Vagrant and Docker Compose labs. Both bootstrap
Go, run `vet` and tests, build the CLI, validate the committed example, and
persist YAML/JSON renders outside the source checkout.

```bash
cd container
cp .env.example .env
./lab.sh up
./lab.sh status
```

Use [Lab environments](docs/lab-environments.md) to choose between the
container and Vagrant workflows and to understand their isolation boundaries.

## Contributing

`main` is the principal branch. Create topic branches from the current `lab`
integration branch, merge validated topic work into `lab`, then promote `lab`
to `main` through a reviewed pull request. Go commands must run through
`scripts/go.sh` or `scripts/go.ps1`; the wrappers keep toolchains and caches
under ignored project directories. See [Development](docs/development.md) and
the controlled iteration process in [`iterations/`](iterations/README.md).
