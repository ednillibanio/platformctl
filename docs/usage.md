# Usage

This page has two guided workflows. Complete [Installation](installation.md)
first and use the [CLI reference](cli-reference.md) for exact flags.

- [Shared root of the `provision-*` repositories](#shared-root-of-the-provision--repositories):
  prepare and check, on a controller, the configuration tree that every
  `provision-*` repository reads. This is the current workflow.
- [YAML model (paused)](#yaml-model-paused): author and render a
  `PLATFORMCTL_HOME`. The commands keep working, but no consumer adopts the
  model now.

## Shared root of the `provision-*` repositories

The shared root is `~/.config/assembleiarr/<env>/` on every controller (or
`ASSEMBLEIARR_CONFIG_DIR`, or `--config-dir`). Its layout, file format,
names and identities are defined by the platform-catalog
[provision shared root standard](https://github.com/ednillibanio/platform-catalog/blob/main/docs/standards/provision-shared-root-standard.md). The shared-root commands run on
Linux and macOS.

New to the shared root? Read [Shared root concepts](shared-root-concepts.md)
first: what the tree is for, its layout, who writes and reads each file, and
how several controllers stay aligned. Repository maintainers who write a
`shared-root.yaml` follow [Writing a key declaration](shared-root-declaration.md).

### 1. Collect the declarations

Each `provision-*` repository declares, in a `shared-root.yaml` at its root,
the keys, credentials and automation key it reads. Pass every declaration of
the repositories that run on this controller with `--declaration`. A
repository without a declaration yet gets a generic skeleton and a warning;
its own documentation then lists the keys to set.

### 2. Create the skeleton

On the first controller of an environment, also generate the automation key
pair:

```bash
platformctl bootstrap-config --env lab --target vsphere-lab \
  --declaration ~/src/provision-example/shared-root.yaml \
  --declaration ~/src/provision-other/shared-root.yaml \
  --automation-key --dry-run
```

Review the `would create` lines, then run the same command without
`--dry-run`. On every other controller of the environment, import the same
private key over SSH instead of generating a new one:

```bash
ssh first-controller cat .config/assembleiarr/lab/credentials/automation/ssh_ed25519 | \
  platformctl bootstrap-config --env lab --target vsphere-lab \
  --declaration ~/src/provision-example/shared-root.yaml \
  --automation-key-import -
```

Running the command again is safe: existing files are reported as `exists`
and never changed. If an existing file lacks a declared key, the command
prints the placeholder lines to add.

With one target and one consumer the result is the tree drawn in
[Shared root concepts: The tree](shared-root-concepts.md#the-tree):
`targets/<t>/vars.sh`, `targets/<t>/hosts/<consumer>.sh`, an empty
`credentials/<consumer>/` and, with `--automation-key`,
`credentials/automation/ssh_ed25519{,.pub}`.

### 3. Set the shared values

`targets/<t>/vars.sh` (`GOVC_*`, `TARGET_*`, `DISPOSABLE_*`) and
`targets/<t>/hosts/<consumer>.sh` (`HOST_*`) hold one commented placeholder
per declared key:

```bash
# GOVC_URL (required): vCenter SDK URL of the target
# Example: https://vcenter.example.test/sdk
# GOVC_URL=
```

Write the value after the `=` and remove the leading `#` and the space after it. Only literal
values are allowed: unquoted without spaces or shell characters,
`'single quoted'`, or `"double quoted"` without `$`, backquote, `\` or `"`.
No `$(...)`, no variables, no unquoted `~`.

### 4. Create the credentials

Consumer secrets go under `credentials/<consumer>/`:

```bash
platformctl credential --env lab --consumer provision-example \
  EXAMPLE_DB_USER=example EXAMPLE_DB_PASS
```

`EXAMPLE_DB_PASS` gets a random value; `EXAMPLE_DB_USER=example` is a
literal non-secret default. A secret that the operator supplies goes through
standard input:

```bash
read -rs GOVC_PASSWORD && printf '%s' "$GOVC_PASSWORD" | \
  platformctl credential --env lab --target vsphere-lab \
  --identity vcenter-access-admin GOVC_USERNAME=svc-access-admin GOVC_PASSWORD=-
```

Target secrets live under `credentials/target-<t>/<identity>/`, and only an
active identity of the standard is accepted. Today that is
`vcenter-access-admin` (`provision-vsphere-platform`). Another repository's
identity, named after it by convention, is usable only once the catalog's
Identities table lists it as active (ADR-0007); check the table for its
current status. A repository that still reads files directly under
`credentials/target-<t>/` keeps its legacy files until its adaptation issue
is closed; `doctor` reports them.

### 5. Check the tree

```bash
platformctl doctor --env lab \
  --declaration ~/src/provision-example/shared-root.yaml
```

It exits `0` when there is no problem (warnings allowed), `1` with problems,
and prints each one with its file and, where there is one, its line. Fix
problems by hand; no command rewrites a file.

Right after step 2, before any value is set (`<root>` stands for the root
path):

```text
problem  <root>/lab/credentials/provision-example/EXAMPLE_DB_PASS: EXAMPLE_DB_PASS: required credential of provision-example missing; run platformctl credential --consumer provision-example
problem  <root>/lab/targets/vsphere-lab/hosts/provision-example.sh:10: HOST_IP: required key is still a placeholder
problem  <root>/lab/targets/vsphere-lab/vars.sh:12: GOVC_URL: required key is still a placeholder
problem  <root>/lab/targets/vsphere-lab/vars.sh:15: GOVC_DATACENTER: required key is still a placeholder
automation key SHA256:WS5zIGaIN7yGD6hFmEWP+UX93khflLW/y9ii/3TFjSE
<root>/lab: 4 problem(s), 0 warning(s)
doctor found 4 problem(s) in <root>/lab
```

After steps 3 and 4 (exit `0`):

```text
automation key SHA256:WS5zIGaIN7yGD6hFmEWP+UX93khflLW/y9ii/3TFjSE
<root>/lab: 0 problem(s), 0 warning(s)
```

A line that is not a literal assignment, here `TARGET_FOLDER=$(id)`:

```text
problem  <root>/lab/targets/vsphere-lab/vars.sh:19: TARGET_FOLDER: only KEY=value lines with a literal value are allowed (no declare, printf -v, functions, commands, $ or ` expansion, unquoted ~)
```

The value is never printed, only the file, line and key. More messages and
their fixes are in [Troubleshooting: Shared root](troubleshooting.md#shared-root).

To compare two controllers, run `platformctl doctor --env lab --format json`
on both and diff the output. It holds the non-secret values, the credential
names (never their values) and the key fingerprint:

```json
{
  "env": "lab",
  "ok": true,
  "problems": [],
  "warnings": [],
  "targets": {
    "vsphere-lab": {
      "vars": {
        "GOVC_DATACENTER": "dc-lab",
        "GOVC_URL": "https://vcenter.lab.example.test/sdk"
      },
      "hosts": { "provision-example": { "HOST_IP": "192.0.2.10" } },
      "credentials": {}
    }
  },
  "credentials": { "provision-example": ["EXAMPLE_DB_PASS"] },
  "automation_key": {
    "private_key": true,
    "public_key": true,
    "fingerprint": "SHA256:WS5zIGaIN7yGD6hFmEWP+UX93khflLW/y9ii/3TFjSE"
  }
}
```

The real output also has `root`, which differs between controllers when
their home directories differ; leave it out of the comparison.

### Try it without touching the real tree

Point the commands at a scratch root:

```bash
export ASSEMBLEIARR_CONFIG_DIR="$(mktemp -d)/assembleiarr"
platformctl bootstrap-config --env lab --target vsphere-lab --automation-key
platformctl doctor --env lab
```

`scripts/e2e-shared-root.sh` runs the full sequence against a scratch root;
see [Development](development.md).

## YAML model (paused)

The rest of this page initializes a `PLATFORMCTL_HOME`, authors each
document type by hand, connects them, and reads what `render` produces. Use
the [Configuration reference](configuration.md) for the complete data
contract.

## 1. Initialize

```bash
platformctl init
platformctl environment init prod
```

Run these commands as the normal operator, not with `sudo`. They write to the
operating system's configuration directory unless an absolute
`PLATFORMCTL_HOME` override is set. Confirm the active location from the success
messages before adding real data.

This creates, under `PLATFORMCTL_HOME`:

```text
defaults.yaml
environments/prod/
├── environment.yaml
├── profiles/
├── services/
└── hosts/
```

`defaults.yaml` and `environment.yaml` are scaffolded with empty
`values`/`credentials`. `profiles/`, `services/`, and `hosts/` start empty —
platformctl has no subcommand to create a profile, service, or host; you
author those YAML files yourself.

On POSIX, generated directories use `0700` and generated files use `0600`.
Apply equally restrictive permissions to documents and credential files you
create manually.

## 2. Author documents by hand

Every document shares the same envelope:

```yaml
apiVersion: platformctl.io/v1alpha1
kind: <Profile|Service|Host>
metadata:
  name: <identifier>   # must match the filename (without .yaml)
spec:
  ...
```

Identifiers are lowercase letters, digits, and hyphens
(`^[a-z][a-z0-9-]*$`).

### Profiles — reusable roles and inheritance

Create the root profile referenced by the database profile:

```yaml
# environments/prod/profiles/base.yaml
apiVersion: platformctl.io/v1alpha1
kind: Profile
metadata:
  name: base
spec:
  values:
    timezone: UTC
```

Then create a child profile:

```yaml
# environments/prod/profiles/database.yaml
apiVersion: platformctl.io/v1alpha1
kind: Profile
metadata:
  name: database
spec:
  extends: base            # optional — at most one parent
  values:
    role: database
```

`extends` names a profile in the same environment. File loading does not depend
on creation order, but `validate` rejects a missing or cyclic parent.

### Service — reusable values (and, optionally, ports) any host can attach to

```yaml
# environments/prod/services/monitoring.yaml
apiVersion: platformctl.io/v1alpha1
kind: Service
metadata:
  name: monitoring
spec:
  values:
    monitoring:
      server_address: 10.0.1.5
      server_port: 10051
  ports:
    - port: 10050
      protocol: tcp
```

### Host — one real machine

```yaml
# environments/prod/hosts/db1.yaml
apiVersion: platformctl.io/v1alpha1
kind: Host
metadata:
  name: db1
spec:
  profile: database         # required — exactly one
  services: [monitoring]    # optional — any number
  addresses: [10.0.2.20/32] # the host's address(es), CIDR notation
  values:
    hostname: db1.prod.internal
  credentials:
    monitoring-agent-psk:
      value: 3f9a7c1e9b8d4f0a9c2e5b7d1a3f6c8e
```

## 3. How they connect

- A **host** picks exactly one **profile** (`spec.profile`) and any number
  of **services** (`spec.services`, by name).
- A **profile** may `extend` one parent profile, forming a single
  inheritance chain (root profile first).
- A **service**'s `ports[].source_hosts` lists other host names in the
  same environment; platformctl resolves each to that host's addresses.
- Every layer — defaults, environment, each profile in the chain, each
  attached service, then the host itself — contributes `values` (deep
  merged) and `credentials` (last one wins per key), in that order. See
  [Configuration reference § Worked merge example](configuration.md#worked-merge-example)
  for a full walkthrough.

Attaching multiple services to one host is how you reuse values across
many hosts: any host that lists `monitoring` automatically gets the
`monitoring.server_address`/`server_port` values above, the same way a
config-management "group" applies shared variables to every host in it.

## 4. Credentials

Three forms, exactly one per credential — see
[Configuration reference § Credentials](configuration.md#credentials) for
details:

```yaml
credentials:
  api-token:
    env: PLATFORM_API_TOKEN     # reference: read from this env var
  ssh-key:
    file: credentials/id_ed25519 # reference: read from this file
  agent-psk:
    value: 3f9a7c1e9b8d4f0a...   # the secret itself, stored inline
```

## 5. Validate and render

```bash
platformctl validate --env prod
platformctl render --env prod                    # YAML, inline values redacted
platformctl render --env prod --format json       # JSON, same redaction
platformctl render --env prod --reveal-secrets     # include real inline values
```

Run `validate` after every edit and before passing data to another repository.
It is read-only. `render` validates again and writes only to standard output.
Do not use `--reveal-secrets` in shared terminals, logs, CI output, or ordinary
debugging.

`render` output for `db1` (redacted):

```yaml
name: db1
addresses: [10.0.2.20/32]
profile: database
services: [monitoring]
config:
  values:
    timezone: UTC
    role: database
    hostname: db1.prod.internal
    monitoring:
      server_address: 10.0.1.5
      server_port: 10051
  credentials:
    monitoring-agent-psk:
      value: <redacted>
ports:
  - service: monitoring
    port: 10050
    protocol: tcp
```

Nothing provisions or applies this output. It's meant to be consumed by
whatever tool actually acts on it — see [Ecosystem](ecosystem.md) for where
that fits and for the current integration limitations.

## 6. Export for shell consumers

Some consumers read environment variables instead of a rendered document.
`platform-modules` is one of them: its tools read `PM_*` variables. Model the
tool's layers as services and export them:

```bash
PLATFORMCTL_HOME="${PWD}/examples/platform-modules-home" \
  platformctl export --env lab \
    --service vsphere-esxi --service packer --prefix PM_
```

That example maps `base/` to `defaults.yaml`, the `lab` overlay to
`environment.yaml`, the vSphere target to `services/vsphere-esxi.yaml`, and the
Packer module layer to `services/packer.yaml`. To use the result, evaluate it
in the shell that runs the tool:

```bash
eval "$(platformctl export --env lab \
  --service vsphere-esxi --service packer --prefix PM_)"
```

Credential files and variables are read by that shell at that moment. The
example references `GITHUB_TOKEN` and two files under
`~/.config/assembleiarr/lab/credentials/platform-modules/`. If a file is
missing or unreadable, or the variable is unset or empty, nothing is exported
and the command fails. See
[`platformctl export`](cli-reference.md#platformctl-export).

## Field reference

| Kind | Required fields | Optional fields |
| --- | --- | --- |
| Defaults | `metadata.name` (must be `defaults`) | `spec.values`, `spec.credentials` |
| Environment | `metadata.name` (must match the environment folder) | `spec.values`, `spec.credentials` |
| Profile | `metadata.name` | `spec.extends`, `spec.values`, `spec.credentials` |
| Service | `metadata.name` | `spec.values`, `spec.credentials`, `spec.ports` |
| — `ports[]` | `port` (1-65535), `protocol` (`tcp`\|`udp`) | `source_hosts` |
| Host | `metadata.name`, `spec.profile` | `spec.services`, `spec.addresses`, `spec.values`, `spec.credentials` |
| — `credentials.<name>` | exactly one of `env`, `file`, `value` | — |

All `values` maps are free-form — anything your downstream consumer needs.
For identifier syntax, loading behavior, validation rules, merge precedence,
credential semantics, and the rendered schema, continue with the
[Configuration reference](configuration.md).
