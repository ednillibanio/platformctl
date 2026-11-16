# Troubleshooting

Start with the exact failing command and preserve its standard error. Parsing
errors stop before semantic validation; semantic validation collects and sorts
all problems it can find. The YAML model commands exit `1` for every error
category, so the message is the useful diagnostic. The shared-root commands
exit `1` for problems and `2` for usage errors; see
[Shared root](#shared-root).

## Confirm the active binary and configuration home

```bash
command -v platformctl
platformctl --help
printf '%s\n' "${PLATFORMCTL_HOME:-<operating-system default>}"
```

If `PLATFORMCTL_HOME` is unset, see the platform defaults in the
[CLI reference](cli-reference.md#configuration-home-selection). The CLI does
not print the resolved home directly except after successful initialization.

## Initialization

### `PLATFORMCTL_HOME must be an absolute path`

Relative overrides are rejected. Use an absolute operator-owned path:

```bash
export PLATFORMCTL_HOME="${HOME}/.config/platformctl"
```

On macOS, the native default contains a space; quote it when setting an
explicit override:

```bash
export PLATFORMCTL_HOME="${HOME}/Library/Application Support/platformctl"
```

### `platform home is not initialized`

`environment init` requires `defaults.yaml` to exist. Run:

```bash
platformctl init
platformctl environment init lab
```

### `refusing to overwrite an existing managed file without --force`

The managed `defaults.yaml` or `environment.yaml` differs from the exact
scaffold. This protects operator content. Compare the file with the templates
in [`internal/config/scaffold.go`](../internal/config/scaffold.go), back it up,
and use `--force` only when replacing that one managed file is intended.
Additional files are preserved.

### Permission differs from `0700` or `0600`

On POSIX, re-running `init` or `environment init` reapplies the documented
permissions to managed directories/files when the scaffold is unchanged.
Windows permission behavior follows the filesystem and does not emulate POSIX
modes.

## Loading and parsing

### `environment "<name>" does not exist`

The selected directory lacks `environment.yaml` under the active home. Confirm
the environment name and root:

```bash
platformctl environment init <name>
platformctl validate --env <name>
```

Do not initialize inside a controller that mounts a host-owned configuration
tree; initialize on the authoritative host instead.

### `metadata.name ... must match filename`

For profiles, services, and hosts, the filename stem and `metadata.name` must
match exactly:

```text
profiles/application.yaml  -> metadata.name: application
```

Only direct children ending in `.yaml` are loaded. `.yml`, nested files,
directories, and other extensions are ignored.

### `field <name> not found`

Structural YAML is strict. Correct the field name or move consumer-specific
data under `spec.values`, the intentionally extensible map. Do not add arbitrary
fields beside `apiVersion`, `kind`, `metadata`, or `spec`.

### `multiple YAML documents are not allowed`

Each file must contain exactly one document. Split `---`-separated documents
into the appropriate individual files.

### YAML duplicate-key error

Keep every mapping key unique. A later duplicate is not an override mechanism;
put overrides in a later configuration layer instead.

## Semantic validation

### `configuration is invalid`

Read every bullet before editing: the validator aggregates independent
problems. Common causes include:

- unsupported `apiVersion` or wrong `kind`;
- invalid lowercase identifier;
- missing profile or service reference;
- profile inheritance cycle;
- invalid or duplicate CIDR;
- port outside `1..65535` or protocol other than `tcp`/`udp`;
- missing, duplicate, or address-less `source_host`;
- credential with zero or multiple forms;
- invalid environment-variable name in a credential.

The complete rule inventory is in
[Configuration reference: Validation contract](configuration.md#validation-contract).

### A file credential is accepted even though the file does not exist

This is expected. `platformctl` preserves `file` as an opaque reference and
never checks, opens, or resolves it. The downstream consumer defines the base
directory, expected contents, permissions, and existence requirements. Document
that contract in the consuming repository.

### An environment credential is accepted but unset

The CLI validates only the variable name syntax. It does not read the process
environment or require the referenced variable to be set. Resolution belongs
to the downstream consumer.

## Rendering

### Inline secret appears as `<redacted>`

This is the secure default for `credentials.<name>.value`. Use
`--reveal-secrets` only when a trusted consumer explicitly requires the real
value:

```bash
umask 077
platformctl render --env lab --reveal-secrets > /secure/output/lab.yaml
```

The shell, not `platformctl`, creates redirected files. Verify permissions and
remove the artifact when it is no longer needed.

### `unsupported format ... expected yaml or json`

Only `yaml` and `json` are implemented:

```bash
platformctl render --env lab --format json
```

### Render output is empty or contains no hosts

An environment with no valid files directly under `hosts/` renders a valid
document with an empty host list. `environment init` creates empty directories;
it does not scaffold a host. Follow the [Usage guide](usage.md) to add profiles,
services, and hosts.

### A value did not merge as expected

Check the layer order:

```text
defaults -> environment -> profile chain -> services in declared order -> host
```

Maps merge recursively. Scalars, lists, and `null` replace the earlier value.
Credentials replace the complete reference object per credential name. See the
[worked merge example](configuration.md#worked-merge-example).

## Shared root

Run `platformctl doctor --env <env>` first: it lists every problem with its
file and line. No command fixes a file; edit or remove it by hand. The rules
come from the [provision shared root standard](https://github.com/ednillibanio/platform-catalog/blob/main/docs/standards/provision-shared-root-standard.md).

### `unknown environment "<env>": not an entry of platform-catalog catalog/environments/`

`--env` accepts only the catalog environments (`lab`, `prod`). A new
environment is added to the catalog first, then to
`internal/sharedroot/names.go`.

### `identity "<i>" is reserved and cannot be used yet` or `is deprecated`

Only an active identity of the standard may be declared, used with
`credential --identity`, or have a directory under
`credentials/target-<t>/`. Today that is `vcenter-access-admin`. A reserved
identity becomes usable when the catalog activates it; `terraform-provisioner` is deprecated (ADR-0007). `doctor`
reports an existing directory of either kind as a problem.

### `a file directly under credentials/target-<t>/`

Target secrets belong in `credentials/target-<t>/<identity>/`. A consumer
that still reads its target account directly from `credentials/target-<t>/`
needs those files until it follows the standard (its adaptation issue); the
problem stays until then.

### `required key is still a placeholder`

The line is still `# KEY=`. Write the value after the `=` and remove the
leading `# `.

### `only KEY=value lines with a literal value are allowed`

The shared files take only literal assignments: no `$(...)`, backquotes,
variables, `declare`, functions, or an unquoted `~`. Write the literal value,
for example the absolute path instead of `~/data`.

### `a comment goes on its own line, not after the value`

A line has a comment after its value, such as `GOVC_URL=https://vc/sdk # lab`.
The standard allows only the assignment on that line. Move the comment to
the line above:

```bash
# lab
GOVC_URL=https://vc/sdk
```

### `the process umask removes owner permissions`

The commands create directories `0700` and files `0600`/`0644` and need a
umask that keeps the owner bits. Run them with `umask 077` or `umask 022`.

### `refusing to follow a symbolic link in the shared root`

No path inside the root may be a symbolic link, and the root itself
(`--config-dir`, `ASSEMBLEIARR_CONFIG_DIR` or `~/.config/assembleiarr`)
must not be one either. Replace the link with the real directory or file, or
point the root at the real directory. Links in the parent directories of the
root are followed.

### `incomplete automation key pair`

`credentials/automation/` exists without both `ssh_ed25519` and
`ssh_ed25519.pub`. `bootstrap-config` creates nothing in it. Restore the pair
from another controller of the environment, or remove the incomplete
directory and import the pair with `--automation-key-import -`.

### `--automation-key-import -` rejects the key

Standard input must hold exactly one unencrypted ed25519 OpenSSH private key
(`BEGIN OPENSSH PRIVATE KEY`). A key with a passphrase, another key type, or
extra text is refused. Pipe the file itself, for example
`ssh first-controller cat .config/assembleiarr/lab/credentials/automation/ssh_ed25519`.

### `shared-root commands are not supported on this operating system`

The shared-root commands run on Linux and macOS only.

## Development bootstrap

### `Go is not available`

Run the bootstrap wrapper first:

```bash
./scripts/bootstrap-dev.sh
```

It requires network access, `curl`, archive tools, and a SHA-256 utility when a
compatible Go installation is unavailable. See [Development](development.md).

### Checksum mismatch while downloading Go

Do not bypass checksum verification. Remove the incomplete ignored toolchain if
necessary, confirm the requested Go version and platform in the bootstrap
script, and retry from a trusted network. If upstream artifacts changed or the
pinned checksum is wrong, treat that as a repository maintenance issue.

## Disposable labs

### Container lab cannot find Compose

`container/lab.sh` first tries `docker compose` and then `docker-compose`.
Install one supported Compose interface and confirm the Docker daemon/context
is reachable before retrying.

### Container or VM reports `status=failed`

Inspect the persisted `reason` and `result_code`:

```bash
cd container
./lab.sh status
```

```bash
cd vagrant
vagrant ssh -c 'cat ~/.local/state/platformctl-lab/status'
```

The `reason` names the failed pipeline stage, such as `development_bootstrap`,
`go_vet`, `go_test`, or `example_validation`.

### Vagrant provider or Apple Silicon limitation

The Vagrant lab supports `virtualbox` and `vmware_desktop`, but availability
depends on host architecture and installed provider support. Use the container
lab for native `linux/arm64` validation on Apple Silicon. See
[Lab environments](lab-environments.md).

## Getting more evidence

For a reproducible report, include:

- operating system and architecture;
- `platformctl --help` and build/commit identity;
- the command and complete redacted error;
- whether `PLATFORMCTL_HOME` is set, without sharing its secret contents;
- `./scripts/go.sh test ./...` results for development failures;
- lab status fields for disposable-lab failures.

Never attach a render produced with `--reveal-secrets`, credential files, raw
tokens, private keys, or a real operator configuration tree.
