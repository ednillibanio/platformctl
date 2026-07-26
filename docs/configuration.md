# Configuration reference

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

## Credentials

Credentials are a map of names to exactly one reference:

```yaml
credentials:
  api-token:
    env: PLATFORM_API_TOKEN
  ssh-key:
    file: credentials/id_ed25519
```

Validation checks only the reference shape. Render output preserves the
reference and never reads the environment variable or file.

## Initialization behavior

Initialization is idempotent when managed files already contain the expected
scaffold. Divergent managed files are preserved and produce an error unless
`--force` is supplied. Force replaces only the known scaffold file and never
deletes additional operator files.

## Vagrant controllers

For lab use, mount the host configuration live at the default Linux location:

```text
/home/vagrant/.config/platformctl
```

Platformctl discovers it automatically when running as `vagrant`. Initialize
and edit the tree only on the host. See the
[Vagrant controller integration guide](vagrant-controller-integration.md) for
copy-and-paste snippets covering Windows, Linux, and macOS.
