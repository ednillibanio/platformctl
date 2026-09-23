# CLI reference

This reference describes the command-line interface implemented in
[`internal/cli/cli.go`](../internal/cli/cli.go). For a guided workflow, use the
[Usage guide](usage.md). For YAML fields and merge semantics, use the
[Configuration reference](configuration.md).

## Synopsis

```text
platformctl <command> [flags]
```

| Command | Purpose | Changes state? |
| --- | --- | --- |
| `init` | Initialize the platform configuration home. | Yes, local files only. |
| `environment init <env>` | Initialize one environment directory. | Yes, local files only. |
| `validate --env <env>` | Parse and validate one environment. | No. |
| `render --env <env>` | Validate and render one effective environment. | No; writes only to standard output. |
| `export --env <env>` | Validate and project resolved values as shell `export` statements. | No; writes only to standard output. |
| `completion <shell>` | Generate Cobra shell-completion output. | No; installation is left to the caller. |
| `help [command]` | Show command help. | No. |

Run `platformctl <command> --help` for the executable's current synopsis.

## Configuration home selection

Every command resolves the same configuration root before reading or writing.

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
- Any current command, parsing, validation, or encoding error exits `1`.

Stable, category-specific exit codes are not implemented in the `v1alpha1`
foundation. Consumers should not infer an error category from the numeric code
alone; capture standard error and treat message text as diagnostic rather than
a stable machine API.

## Global behavior and limitations

- Commands operate on one environment at a time.
- There is no `list`, `delete`, `diff`, `apply`, migration, or remote API
  command.
- There are no commands to scaffold profiles, services, or hosts.
- Rendering does not write an artifact unless the caller redirects output.
- Rendering does not invoke downstream repositories or resolve credentials.
- Initialization only manages known scaffold files; it never recursively
  replaces or deletes an operator tree.
