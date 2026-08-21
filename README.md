# platformctl

`platformctl` is a cross-platform CLI for authoring, validating, merging, and
rendering private infrastructure configuration by environment. It gives
operators one layered source of truth that downstream provisioning and
automation repositories can consume as deterministic YAML or JSON.

The current `v1alpha1` foundation is configuration-only. It does **not** invoke
Terraform, Ansible, Packer, Vagrant, `govc`, SemaphoreUI, or any service
provisioner.

## Status

- Development branch: `lab`
- Configuration API: `platformctl.io/v1alpha1`
- Published releases: none; build from source
- Supported development hosts: Linux, macOS, and Windows
- Render formats: YAML (default) and JSON

See [implementation progress](docs/progress.md) for completed capabilities,
validation coverage, and deferred work.

## How it fits

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
the relationships with `infra-gitops`, `provision-platform-environments`,
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

After installing `platformctl`, initialize the operating system's default
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
| Install or update from source | [Installation](docs/installation.md) |
| Create a complete environment | [Usage guide](docs/usage.md) |
| Look up commands and flags | [CLI reference](docs/cli-reference.md) |
| Look up YAML fields and merge behavior | [Configuration reference](docs/configuration.md) |
| Understand internals and boundaries | [Architecture](docs/architecture.md) |
| Understand related repositories | [Ecosystem and integrations](docs/ecosystem.md) |
| Diagnose common failures | [Troubleshooting](docs/troubleshooting.md) |
| Develop and validate changes | [Development](docs/development.md) |
| Choose a disposable validation lab | [Lab environments](docs/lab-environments.md) |
| Mount live configuration into a Vagrant controller | [Vagrant controller integration](docs/vagrant-controller-integration.md) |

## Security boundary

Real operator configuration lives outside Git. A credential may contain an
environment-variable reference, a file reference, or an inline value. Inline
values are redacted by default; `render --reveal-secrets` deliberately exposes
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

Use `lab` as the base branch and create a topic branch. Go commands must run
through `scripts/go.sh` or `scripts/go.ps1`; the wrappers keep toolchains and
caches under ignored project directories. See [Development](docs/development.md)
and the controlled iteration process in [`iterations/`](iterations/README.md).
