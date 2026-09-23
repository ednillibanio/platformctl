# Usage

A step-by-step guide to initializing a configuration home, authoring each
document type by hand, connecting them, and reading what `render` produces.
Complete [Installation](installation.md) first. Use the
[CLI reference](cli-reference.md) for exact flags and the
[Configuration reference](configuration.md) for the complete data contract;
this page is a guided operator workflow.

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
