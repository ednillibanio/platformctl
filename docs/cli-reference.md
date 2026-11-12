# CLI reference

This reference describes the command-line interface implemented in
[`internal/cli/cli.go`](../internal/cli/cli.go) and
[`internal/cli/sharedroot.go`](../internal/cli/sharedroot.go). For a guided
workflow, use the [Usage guide](usage.md). For YAML fields and merge semantics
of the paused YAML model, use the [Configuration reference](configuration.md).

The commands fall into two groups:

- **Shared-root commands** (`bootstrap-config`, `credential`, `doctor`)
  create and check the configuration tree that every `provision-*`
  repository reads on a controller. Their contract is the platform-catalog
  [provision shared root standard](https://github.com/ednillibanio/platform-catalog/blob/main/docs/standards/provision-shared-root-standard.md); when this page and the standard
  differ, the standard wins. They run on Linux and macOS only.
- **YAML model commands** (`init`, `environment init`, `validate`, `render`,
  `export`) author and render the `PLATFORMCTL_HOME` model. That model is
  paused: the commands keep working and do not change, but no consumer
  adopts them now.

## Synopsis

```text
platformctl <command> [flags]
```

| Command | Purpose | Changes state? |
| --- | --- | --- |
| `bootstrap-config --env <env>` | Create the shared-root skeleton of one environment. | Yes, local files only; never changes an existing file. |
| `credential --env <env> ... NAME...` | Create credential files in the shared root. | Yes, local files only; never changes an existing file. |
| `doctor --env <env>` | Check the shared root of one environment. | No. |
| `init` | Initialize the platform configuration home. | Yes, local files only. |
| `environment init <env>` | Initialize one environment directory. | Yes, local files only. |
| `validate --env <env>` | Parse and validate one environment. | No. |
| `render --env <env>` | Validate and render one effective environment. | No; writes only to standard output. |
| `export --env <env>` | Validate and project resolved values as shell `export` statements. | No; writes only to standard output. |
| `completion <shell>` | Generate Cobra shell-completion output. | No; installation is left to the caller. |
| `help [command]` | Show command help. | No. |

Run `platformctl <command> --help` for the executable's current synopsis.

## Configuration home selection

Every YAML model command resolves the same configuration root before reading
or writing. The shared-root commands use their own root; see
[Shared root selection](#shared-root-selection).

| Platform | Default |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/platformctl` when set; otherwise normally `~/.config/platformctl` |
| macOS | `~/Library/Application Support/platformctl` |
| Windows | `%AppData%\platformctl` |

Set `PLATFORMCTL_HOME` to override the default:

```bash
PLATFORMCTL_HOME=/srv/platform-config platformctl validate --env lab
```

```powershell
$env:PLATFORMCTL_HOME = "D:\platform-config"
platformctl validate --env lab
```

The override is required to be an absolute path. `platformctl` does not search
the current directory or a repository checkout for configuration.

## `platformctl init`

```text
platformctl init [--force]
```

Creates:

```text
<PLATFORMCTL_HOME>/
├── defaults.yaml
└── environments/
```

On POSIX systems, directories are set to `0700` and `defaults.yaml` to `0600`.

| Flag | Default | Effect |
| --- | --- | --- |
| `--force` | `false` | Replace a divergent managed `defaults.yaml`. It does not remove unrelated files. |

Re-running the command is safe when the managed file still contains the exact
scaffold. If it differs, the command refuses to overwrite it unless `--force`
is supplied. Review and back up operator changes before forcing replacement.

Expected success output:

```text
Initialized platform home at <absolute-path>
```

## `platformctl environment init`

```text
platformctl environment init <env> [--force]
```

`<env>` must match `^[a-z][a-z0-9-]*$`. The platform home must already contain
`defaults.yaml`; run `platformctl init` first.

Creates:

```text
<PLATFORMCTL_HOME>/environments/<env>/
├── environment.yaml
├── hosts/
├── profiles/
└── services/
```

| Flag | Default | Effect |
| --- | --- | --- |
| `--force` | `false` | Replace only a divergent managed `environment.yaml`. Additional files are preserved. |

The same idempotency and POSIX permission rules as `init` apply. The command
does not create profiles, services, or hosts.

## `platformctl validate`

```text
platformctl validate --env <env>
```

The command performs strict YAML loading followed by semantic and
cross-document validation. It reports all semantic problems it can collect,
sorted into one error, instead of stopping at the first one. Parsing and file
loading errors necessarily stop earlier.

| Flag | Required | Expected value |
| --- | --- | --- |
| `--env` | Yes | Identifier matching `^[a-z][a-z0-9-]*$`. |

Expected success output:

```text
Environment "lab" is valid
```

Validation is read-only and safe to repeat. See
[Configuration validation](configuration.md#validation-contract) for the
complete rule inventory.

## `platformctl render`

```text
platformctl render --env <env> [--format yaml|json] [--reveal-secrets]
```

Rendering always validates first. Invalid configuration produces no rendered
document.

| Flag | Required | Default | Effect |
| --- | --- | --- | --- |
| `--env` | Yes | — | Selects one environment. |
| `--format` | No | `yaml` | Selects `yaml` or `json`; other values are rejected. |
| `--reveal-secrets` | No | `false` | Includes real inline credential `value` fields instead of `<redacted>`. |

The command writes the document to standard output, which makes redirection and
pipes possible:

```bash
platformctl render --env lab --format json > /secure/output/lab.json
```

Treat output generated with `--reveal-secrets` as secret material. Avoid
terminal scrollback, shared logs, CI artifacts, and world-readable files.
`platformctl` does not create or permission redirected output; the shell does.

The render preserves `env` and `file` references exactly and never reads their
targets. Host order and resolved source addresses are deterministic. Declared
service order and address order are preserved where order is meaningful.

## `platformctl export`

```text
platformctl export --env <env> [--service <name>]... [--host <name>]
                   [--format env] [--prefix <PREFIX>] [--reveal-secrets]
```

Export validates first and prints shell code that exports one variable per
resolved leaf value and per credential. It serves consumers that read
environment variables, such as `platform-modules` (`PM_*`). The contract is
[ADR-0002](https://github.com/ednillibanio/platform-catalog/blob/main/docs/architecture/decisions/0002-configuration-contract.md).

| Flag | Required | Default | Effect |
| --- | --- | --- | --- |
| `--env` | Yes | — | Selects one environment. |
| `--service` | No | none | Service layer to apply after defaults and the environment. Repeatable; applied in the order given. |
| `--host` | No | — | Exports one host's resolved configuration instead. Mutually exclusive with `--service`. |
| `--format` | No | `env` | Only `env` is supported. |
| `--prefix` | No | empty | Prepended to every variable name, for example `PM_`. |
| `--reveal-secrets` | No | `false` | Allows inline credential `value` fields, emitted as literals. Without it they are refused. |

Evaluate the output; never store it:

```bash
eval "$(platformctl export --env lab \
  --service vsphere-esxi --service packer --prefix PM_)"
```

- Names flatten the value path with `_` (`-` and `.` also become `_`) and are
  upper-cased: `vsphere.endpoint` becomes `PM_VSPHERE_ENDPOINT`.
- Strings, numbers, and booleans are exported. Lists, `null`, empty maps,
  invalid names, and two sources producing the same name are errors.
- `platformctl` never reads secrets. An `env` credential is copied from the
  named variable by the evaluating shell. A `file` credential is read with
  `cat` by that shell; it must be absolute or start with `~/`, which resolves
  against `HOME` on the evaluating host.
- All secrets are read before anything is exported. If a referenced variable
  is unset or empty, or a file is unreadable, the evaluation prints the cause
  to standard error, exports nothing, and returns a non-zero status. It does
  not terminate an interactive shell; under `set -e` it stops the script.
- The output targets bash, zsh, and dash.

Evaluating the output runs `cat` on paths taken from `PLATFORMCTL_HOME`, so
that tree must be as trusted as a shell profile.

## Shared root selection

The shared-root commands resolve the root in this order and then work under
`<root>/<env>/`:

1. `--config-dir <path>`
2. `ASSEMBLEIARR_CONFIG_DIR`
3. `~/.config/assembleiarr`

`--env` is required and must be an environment of the platform-catalog
`catalog/environments/` (`lab`, `prod`). Target, consumer, identity, key and
credential names must match the patterns of the standard's "Names" section;
a name outside them is a usage error, so no path leaves `<root>/<env>/`.

Common behavior of the three commands:

- They never overwrite, truncate, move or delete an existing file, and they
  are idempotent: a second run with the same arguments changes nothing.
- Each new file is written atomically under a temporary name and published
  only if the final name does not exist, so no file is ever left partly
  written, and of two concurrent runs only one creates it. Atomicity is per
  file: a run that fails part-way may already have created some of its
  files and directories. Running the same command again creates the rest and
  reports the others as `exists`.
- They refuse to follow a symbolic link inside the tree.
- Directories are created with mode `0700`; shared files (`vars.sh`,
  `hosts/<consumer>.sh`) with `0644`; credentials and the private key with
  `0600`. A umask that removes owner bits is an error.
- They never print a secret value.
- On Windows they exit `1` with "not supported".

The tree layout and the file format are defined in the standard ("Paths"
and "File format").

## `platformctl bootstrap-config`

```text
platformctl bootstrap-config --env <env> [--target <t>]... [--consumer <c>]...
  [--declaration <path>]... [--automation-key | --automation-key-import -]
  [--config-dir <path>] [--dry-run]
```

Creates `<root>/<env>/`, `targets/<t>/vars.sh` and
`credentials/target-<t>/` for each target, `credentials/target-<t>/<identity>/`
for each identity declared in `target_credentials`, and `credentials/<c>/`
plus `targets/<t>/hosts/<c>.sh` for each consumer and target.

| Flag | Required | Default | Effect |
| --- | --- | --- | --- |
| `--env` | Yes | — | Environment. |
| `--target` | No | none | Target to create. Repeatable. |
| `--consumer` | No | none | Consumer (`provision-*` repository) to create. Repeatable. Consumers of the declarations are added. |
| `--declaration` | No | `./shared-root.yaml` when it exists | Key declaration of a consumer. Repeatable. |
| `--automation-key` | No | `false` | Generate the ed25519 automation key pair. Only on the first controller of an environment. |
| `--automation-key-import -` | No | — | Read the automation private key from standard input (never from an argument) and derive the public key. On every other controller. Mutually exclusive with `--automation-key`. |
| `--config-dir` | No | see above | Shared root. |
| `--dry-run` | No | `false` | Print every action and change nothing. |

- With declarations, `vars.sh` and each hosts file get a commented
  placeholder, `# KEY=`, for exactly the declared `target` and `host` keys,
  preceded by the key's description. To set a key, write its value after the
  `=` and remove the leading `# `.
- Without a declaration, a generic skeleton is written and a warning is
  printed. An invalid declaration is an error, and nothing is written.
- No value is written: the command never prompts for or accepts a value.
- An existing file that lacks declared keys is left unchanged; the command
  prints the placeholder lines to add.
- The automation pair is published as one unit: both files are written in a
  temporary directory under `credentials/` that is renamed to
  `credentials/automation/` only if that directory does not exist.
  `--automation-key-import -` accepts only an ed25519 OpenSSH private key
  without a passphrase. Move the private key between controllers over SSH
  or another encrypted channel, never through a file outside the shared
  root.

Each action is printed on one line: `created`, `exists`, `would create`
(`--dry-run`), or the placeholder lines to add to an existing file.

```bash
platformctl bootstrap-config --env lab --target vsphere-lab \
  --declaration ~/src/provision-example/shared-root.yaml --automation-key
```

## `platformctl credential`

```text
platformctl credential --env <env> (--consumer <c> | --target <t> --identity <i>)
  NAME[=VALUE | =-]... [--length <n>] [--config-dir <path>] [--dry-run]
```

Creates one file per argument, mode `0600`, with no trailing newline, under
`credentials/<c>/` (`--consumer`) or `credentials/target-<t>/<i>/`
(`--target` with `--identity`). Missing directories are created with mode
`0700`.

| Argument | Value written |
| --- | --- |
| `NAME` | Random: `--length` bytes (default 32, at most 1024) from the system's cryptographic random source, base64 encoded with padding, like `openssl rand -base64`. |
| `NAME=VALUE` | `VALUE` literally. Only for non-secret defaults, such as a database user: a command-line argument is visible to other processes. |
| `NAME=-` | Read from standard input, at most 64 KiB; one trailing `\n` or `\r\n` is dropped. Use it for a secret the operator supplies. Only one `NAME=-` per run. |

- `--identity` must be an active identity of the standard's "Identities"
  table. Today only `vcenter-access-admin` is active; reserved
  (`provision-nexus3`, `provision-semaphoreui`, `platform-modules-live`,
  `platform-modules-packer`) and deprecated (`terraform-provisioner`)
  identities are usage errors.
- An existing file is skipped and reported as `exists`, never overwritten.
  Standard input is read only when the `NAME=-` file does not exist.
- Every value is resolved before anything is created: an invalid value on
  standard input leaves nothing behind.
- The output names each file and where its value came from (`random value,
  N characters`, `literal value`, `value from standard input`), never the
  value.
- It supersedes `provision-semaphoreui`'s `gen-credential` with the same
  file contract.

```bash
platformctl credential --env lab --consumer provision-example \
  EXAMPLE_DB_USER=example EXAMPLE_DB_PASS

read -rs GOVC_PASSWORD && printf '%s' "$GOVC_PASSWORD" | \
  platformctl credential --env lab --target vsphere-lab \
  --identity vcenter-access-admin GOVC_USERNAME=svc-access-admin GOVC_PASSWORD=-
```

Standard input from a terminal echoes what is typed; use `read -rs` and a
pipe as above.

## `platformctl doctor`

```text
platformctl doctor --env <env> [--target <t>]... [--consumer <c>]...
  [--declaration <path>]... [--format text|json] [--config-dir <path>]
```

Read-only check of `<root>/<env>/`. It reports every problem, not only the
first, and never creates the root or the environment.

| Flag | Required | Default | Effect |
| --- | --- | --- | --- |
| `--env` | Yes | — | Environment. |
| `--target` | No | every target found | Limits the declared checks to these targets. Repeatable. |
| `--consumer` | No | none | Consumer whose files must exist, besides the declared ones. Repeatable. |
| `--declaration` | No | `./shared-root.yaml` when it exists | Key declaration. Repeatable. |
| `--format` | No | `text` | `text` or `json`. |
| `--config-dir` | No | see above | Shared root. |

Problems (exit `1`):

- owner, mode and type of every path, symbolic links, and temporary files or
  directories left by an interrupted run, in the whole tree;
- the file format of every shared file (`$(...)`, backquotes, an unquoted
  `~`, a duplicate key, a key with the wrong prefix for its file);
- an empty credential file, a file directly under `credentials/target-<t>/`,
  and an identity directory that is not an active identity;
- an automation private key that does not match its public key, or a pair
  with one file missing;
- with declarations: a required key, credential or key pair that is missing,
  empty or still a placeholder; a missing hosts file of a declared consumer;
  a declared credential name or value (8 bytes or longer) found in a file
  readable by others; an invalid declaration.

Undeclared keys, credentials and files are warnings and do not change the
exit status.

`--format text` prints one line per finding with the file, the line and the
key, then the automation key fingerprint and a summary. `--format json`
prints one object:

| Field | Content |
| --- | --- |
| `env`, `root`, `ok` | Environment, environment directory, and `true` when there is no problem. |
| `problems`, `warnings` | Findings: `file`, `line`, `key`, `message`. |
| `targets` | Per target: `vars` and `hosts` (non-secret keys and values) and `credentials` (file names per identity). |
| `credentials` | Credential file names per consumer. |
| `automation_key` | `private_key`, `public_key` (present or not) and `fingerprint` (SHA256). |

The JSON never holds a secret value or a hash of one. When a declared
credential name or value is found in a shared file, all values of that file
are left out. It is meant for comparing the trees of several controllers.

## Completion

Cobra exposes completion generation for Bash, Zsh, Fish, and PowerShell:

```bash
platformctl completion zsh --help
```

Follow the generated help for the selected shell. Completion generation only
prints a script; it does not modify shell startup files.

## Standard output, errors, and exit status

- Successful informational messages and rendered documents go to standard
  output.
- Errors go to standard error from the process entrypoint.
- Success exits `0`.
- The shared-root commands exit `1` when they find problems or cannot
  complete, and `2` on a usage error (unknown flag, missing `--env`, invalid
  name, invalid combination). `doctor` exits `0` when it finds only
  warnings.
- Every error of the YAML model commands exits `1`; they have no
  category-specific codes. Capture standard error and treat its text as
  diagnostic rather than a stable machine API.

## Global behavior and limitations

- Commands operate on one environment at a time.
- There is no `list`, `delete`, `diff`, `apply`, migration, or remote API
  command. No shared-root command removes or rewrites a file; fixing a
  problem that `doctor` reports is a manual step.
- There are no commands to scaffold profiles, services, or hosts.
- Rendering does not write an artifact unless the caller redirects output.
- Rendering does not invoke downstream repositories or resolve credentials.
- Initialization only manages known scaffold files; it never recursively
  replaces or deletes an operator tree.
