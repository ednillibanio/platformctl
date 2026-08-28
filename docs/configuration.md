# Configuration reference

This is the authoritative operator reference for the `v1alpha1` input and
rendered models. For a guided example, use [Usage](usage.md). For command flags,
output, and exit behavior, use the [CLI reference](cli-reference.md).

## Layout

```text
platformctl/
├── defaults.yaml
└── environments/<env>/
    ├── environment.yaml
    ├── profiles/*.yaml
    ├── services/*.yaml
    └── hosts/*.yaml
```

Every document uses `apiVersion: platformctl.io/v1alpha1`, a typed `kind`,
`metadata.name`, and `spec`. Filenames for profiles, services, and hosts must
match `metadata.name`. Identifiers use lowercase letters, digits, and hyphens.

`spec.values` is the extensible module-facing map. All other fields are strict.
Unknown structural fields, duplicate YAML keys, multiple YAML documents, and
invalid references are rejected.

## Loading contract

| Rule | Behavior |
| --- | --- |
| Environment selection | `--env` must match `^[a-z][a-z0-9-]*$`. |
| Root documents | `defaults.yaml` and `environments/<env>/environment.yaml` are loaded from exact paths. |
| Named documents | Only direct files ending in `.yaml` under `profiles/`, `services/`, and `hosts/` are loaded. |
| Ignored entries | Nested directories, `.yml` files, and other extensions are ignored. |
| Filename identity | `<name>.yaml` must contain `metadata.name: <name>`. |
| Environment identity | `environment.yaml` metadata must equal the requested `<env>`. |
| YAML stream | Exactly one YAML document per file. |
| Structural fields | Unknown fields are rejected by typed decoding. |
| Extensible values | Arbitrary consumer data belongs only under `spec.values`. |

Directories for profiles, services, or hosts may be absent; the loader treats
the missing collection as empty. A missing environment file is reported as a
missing environment. Author documents against the public schema even where a
zero-value Go struct might currently allow an omitted empty `spec`; `spec` is
part of the supported document envelope.

## Document field inventory

| Kind | Location | Required content | Optional content |
| --- | --- | --- | --- |
| `Defaults` | `defaults.yaml` | header; `metadata.name: defaults`; `spec` | `spec.values`, `spec.credentials` |
| `Environment` | `environments/<env>/environment.yaml` | header; metadata matching `<env>`; `spec` | `spec.values`, `spec.credentials` |
| `Profile` | `profiles/<name>.yaml` | header and matching metadata; `spec` | `spec.extends`, values, credentials |
| `Service` | `services/<name>.yaml` | header and matching metadata; `spec` | values, credentials, ports |
| `Host` | `hosts/<name>.yaml` | header, matching metadata, `spec.profile` | services, addresses, values, credentials |

Every header contains:

```yaml
apiVersion: platformctl.io/v1alpha1
kind: Defaults|Environment|Profile|Service|Host
metadata:
  name: lowercase-identifier
spec: {}
```

The JSON Schemas under [`schemas/v1alpha1/`](../schemas/v1alpha1/) describe the
public input and output contracts for editors and external tooling. The CLI
uses typed Go decoding plus semantic validation; it does not execute JSON
Schema validation at runtime.

## Directory structure created

The layout above is abstract. Here is the concrete tree from
[`examples/platform-home`](../examples/platform-home), one environment `lab`
with two profiles, one service, and two hosts:

```text
examples/platform-home/
├── defaults.yaml                                # base values + credentials for the whole home
└── environments/
    └── lab/
        ├── environment.yaml                      # env-wide overlay for "lab"
        ├── profiles/
        │   ├── base.yaml                          # root profile, no extends
        │   └── application.yaml                   # extends: base
        ├── services/
        │   └── http.yaml                          # values + ports (with source_hosts)
        └── hosts/
            ├── app.yaml                            # profile: application, services: [http]
            └── controller.yaml                     # profile: base, no services
```

| File | Role |
| --- | --- |
| `defaults.yaml` | base `values`/`credentials` shared by every environment |
| `environments/lab/environment.yaml` | values/credentials that apply only to `lab` |
| `environments/lab/profiles/base.yaml` | root profile; no `extends` |
| `environments/lab/profiles/application.yaml` | leaf profile; `extends: base` |
| `environments/lab/services/http.yaml` | reusable service values plus its ports |
| `environments/lab/hosts/app.yaml` | a host: picks a profile, attaches services, adds its own overrides |
| `environments/lab/hosts/controller.yaml` | a second host on the same profile, with no services |

Everything under `profiles/`, `services/`, and `hosts/` is reusable: a
profile can be shared by many hosts, and a service's values and ports are
merged into every host that lists it. See the worked example below for
exactly how that reuse resolves.

## Resolution

For each host, layers are applied in this order:

1. defaults;
2. environment;
3. profile ancestors, oldest first;
4. selected profile;
5. selected services in declared order;
6. host.

Maps merge recursively. Scalars, nulls, and lists replace the earlier value.
A profile has at most one `extends` parent. Missing parents and cycles are
invalid.

Hosts contain CIDR addresses, one profile, and an ordered service list. Service
ports use numeric ports from 1 through 65535 and the `tcp` or `udp` protocol.
Each `source_hosts` entry refers to a host in the same environment and renders
as that host's sorted, deduplicated CIDR addresses.

## Validation contract

Validation is offline and does not access referenced secrets or remote systems.
The CLI enforces:

- exact `apiVersion: platformctl.io/v1alpha1` and expected `kind`;
- lowercase identifiers matching `^[a-z][a-z0-9-]*$`;
- `metadata.name: defaults` for the defaults document;
- exactly one of `env`, `file`, or `value` for every credential;
- credential names using the identifier pattern;
- environment-variable references matching `^[A-Za-z_][A-Za-z0-9_]*$`;
- an existing parent for every non-empty `Profile.spec.extends`;
- no profile inheritance cycle;
- port range `1..65535` and protocol `tcp` or `udp`;
- unique `source_hosts`, each referring to a host with at least one address;
- one existing profile for every host;
- unique, existing services in each host's ordered service list;
- unique, parseable CIDR prefixes in every host address list.

Independent semantic problems are sorted and returned together. File-open and
YAML parsing failures occur before a complete bundle exists and therefore stop
the pipeline immediately.

## Worked merge example

Resolving host `app` from `examples/platform-home` (`platformctl render
--env lab`), layer by layer:

1. **defaults**: `locale: en_US.UTF-8`, `logging.level: info`; credential
   `automation-token` (env `PLATFORM_AUTOMATION_TOKEN`).
2. **+ environment `lab`**: adds `domain: lab.example.internal`; `logging.level`
   is a scalar, so `debug` replaces `info` rather than merging with it.
3. **+ profile ancestors, oldest first (`base`)**: adds `hardening.enabled:
   true` and `timezone: UTC`.
4. **+ selected profile (`application`)**: adds `role: application` and
   `workers: 2`.
5. **+ selected services, declared order (`http`)**: adds a new `http` map
   (`listen`, `tls`) — no key collision with anything above, so it merges in
   cleanly. Its port `8080/tcp` resolves `source_hosts: [controller]` to
   `controller`'s sorted addresses.
6. **+ host `app`**: `workers: 4` replaces the profile's `workers: 2` (last
   layer wins on scalars); credential `application-password` (file
   `credentials/app-password`) is added as a new key.

Final rendered `app`:

```yaml
name: app
addresses: [192.0.2.20/32]
profile: application
services: [http]
config:
  values:
    locale: en_US.UTF-8
    domain: lab.example.internal
    logging:
      level: debug
    hardening:
      enabled: true
    timezone: UTC
    role: application
    workers: 4
    http:
      listen: 0.0.0.0
      tls: false
  credentials:
    automation-token:
      env: PLATFORM_AUTOMATION_TOKEN
    application-password:
      file: credentials/app-password
ports:
  - service: http
    port: 8080
    protocol: tcp
    sources: [192.0.2.10/32]
```

`values` merge with `internal/merge.Maps`: maps merge key by key, recursively;
any other type (scalar, list, null) replaces the earlier layer's value
outright. `credentials` merge with a plain key overwrite — a later layer
redefining a credential name (say, switching `automation-token` from an `env`
to a `file` reference) replaces the whole reference object, not just the
changed field.

## Credentials

Credentials are a map of names to exactly one of three forms:

```yaml
credentials:
  api-token:
    env: PLATFORM_API_TOKEN          # read from this environment variable
  ssh-key:
    file: credentials/id_ed25519     # read from this file
  build-password:
    value: s0meGeneratedSecret       # the secret itself, stored inline
```

`env` and `file` are references: they only point at where a secret lives, and
platformctl never reads or prints the value they point to. `value` is
different — it stores the secret itself, inside your local
`PLATFORMCTL_HOME` tree (which lives outside git). Use it when a secret has
no other natural home, for example a generated build password; prefer `env`
or `file` when the secret is already managed elsewhere (a secrets manager, an
existing key file).

Because `value` genuinely contains a secret, `platformctl render` redacts it
by default — the field is printed as `<redacted>`. Pass `--reveal-secrets` to
include the real value in the output. `env` and `file` references are never
redacted, since they never contain the secret in the first place.

Credential behavior by form:

| Form | Stored in configuration | CLI validation | Render behavior | Resolution owner |
| --- | --- | --- | --- | --- |
| `env` | Environment-variable name | Non-empty and shell-style name syntax | Name is preserved and never redacted | Downstream consumer |
| `file` | Opaque path/reference string | Non-empty only | String is preserved and never redacted | Downstream consumer |
| `value` | Secret itself | Non-empty and exclusive | `<redacted>` by default; real value with `--reveal-secrets` | Already resolved in config |

`platformctl` does not define whether a `file` path is absolute, relative to
`PLATFORMCTL_HOME`, relative to another working directory, or a logical name.
It also does not check existence, permissions, symlinks, or file contents. The
consumer must define and enforce that contract. Similarly, an `env` reference
need not be set in the `platformctl` process; only its name is validated.

Credential maps merge by whole-object replacement. For example, a later
`automation-token: {file: ...}` completely replaces an earlier
`automation-token: {env: ...}`.

## Initialization behavior

Initialization is idempotent when managed files already contain the expected
scaffold. Divergent managed files are preserved and produce an error unless
`--force` is supplied. Force replaces only the known scaffold file and never
deletes additional operator files.

For initialization, `platformctl` rejects symbolic links at the configuration
home root, its managed directories, and the managed `defaults.yaml` and
`environment.yaml` files. Do not use a symbolic link to relocate this managed
tree: select its real absolute location with `PLATFORMCTL_HOME` instead. This
check protects scaffold writes and permission changes; it does not change the
opaque semantics of credential `file` references.

External controllers that need live visibility into an operator's
configuration tree should mount `PLATFORMCTL_HOME` using the pattern in the
[Vagrant controller integration guide](vagrant-controller-integration.md).

On POSIX systems, initialization sets managed directories to `0700` and managed
files to `0600`. Manually created profiles, services, hosts, and credential
targets are the operator's responsibility and should use permissions
appropriate to their contents.

## Rendered document

`render` produces one `RenderedEnvironment`:

```yaml
apiVersion: platformctl.io/v1alpha1
kind: RenderedEnvironment
metadata:
  name: lab
hosts: []
```

Hosts are sorted by name. Each host contains:

| Field | Source |
| --- | --- |
| `name` | Host metadata/filename |
| `addresses` | Host declaration, original order |
| `profile` | Selected host profile |
| `services` | Declared service order |
| `config.values` | Effective recursive merge |
| `config.credentials` | Effective credential map, redacted by CLI default |
| `ports` | Attached service ports with resolved, sorted source CIDRs |

YAML is the default. JSON uses indented output and a final newline. Rendering
is read-only and writes to standard output; file creation, permissions, and
cleanup for redirected output belong to the caller.
