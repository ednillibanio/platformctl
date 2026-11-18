# Writing a key declaration (`shared-root.yaml`)

This guide is for the maintainer of a `provision-*` repository. It explains
how to write the repository's `shared-root.yaml`, the file that tells
`platformctl` what the repository reads from the
[shared root](shared-root-concepts.md), and how to check it. The format is
defined by the platform-catalog
[provision shared root standard](https://github.com/ednillibanio/platform-catalog/blob/main/docs/standards/provision-shared-root-standard.md),
section "Key declaration"; that section wins over this guide.

## What the file is for

`shared-root.yaml` sits at the root of the repository and lists everything
the repository reads from the shared root, and nothing else. The operator
passes it to `platformctl`:

- `bootstrap-config --declaration <file>` creates one commented placeholder
  per key, with the key's description, and the consumer's directories;
- `doctor --declaration <file>` reports a required key that is missing or
  still a placeholder, a required credential file that does not exist, and
  a declared key in the wrong file.

The repository owns the file and changes it in the same pull request that
changes what the repository reads.

## Example

```yaml
version: 1
consumer: provision-example
automation_key: true
target:
  - name: GOVC_URL
    required: true
    description: vCenter SDK URL of the target
    example: https://vcenter.example.test/sdk
  - name: TARGET_FOLDER
    required: false
    description: VM folder of the service hosts
host:
  - name: HOST_IP
    required: true
    description: Static IPv4 address of the service host
credentials:
  - name: EXAMPLE_DB_PASS
    required: true
    description: Database password, generated at random
```

The example has no `target_credentials`: a repository declares there only
the target secrets of the identity it runs as (see
[Consumers and identities](shared-root-concepts.md#consumers-and-identities)),
and `platformctl` accepts only an `active` identity. Today the only active
one is `vcenter-access-admin`, the identity of `provision-vsphere-platform`.
Each other `provision-*` repository runs as its own identity, named after it
by convention. Check its row in the standard's Identities table: until the
row exists and is `active`, the repository leaves `target_credentials` out.
Once it is active, an entry looks like this:

```yaml
target_credentials:
  - name: GOVC_PASSWORD
    identity: provision-example   # the repository's own identity, once active
    required: true
    description: Password of the vCenter account
```

## Fields

| Field | Meaning |
| --- | --- |
| `version` | Format version. Required; currently `1`. |
| `consumer` | The repository name, `provision-<x>`. Required. |
| `automation_key` | `true` when the repository uses the automation SSH key pair. Default `false`. |
| `target` | Keys read from `targets/<t>/vars.sh`. Names start with `GOVC_`, `TARGET_` or `DISPOSABLE_`. |
| `host` | Keys read from `targets/<t>/hosts/<consumer>.sh`. Names start with `HOST_`. |
| `credentials` | Secret files read under `credentials/<consumer>/`. |
| `target_credentials` | Secret files read under `credentials/target-<t>/<identity>/`. Each entry needs `identity`. |

Every entry has:

| Field | Rule |
| --- | --- |
| `name` | Upper case, digits and `_`, starting with a letter (`^[A-Z][A-Z0-9_]*$`). Required. |
| `required` | `true` or `false`. Required. A required key that is missing makes `doctor` fail. |
| `description` | One line. Required; it becomes the placeholder comment the operator reads. |
| `example` | Optional, only in `target` and `host`; printed as `# Example:` under the description. Never a real or secret value. |
| `identity` | Only in `target_credentials`, and required there. |

Parsing is strict. These are errors, and nothing is written:

- an unknown field, or a field in the wrong section;
- the same name twice in one section, or in both `target` and `host`;
- a `target` or `host` name with a prefix that section does not allow;
- a consumer name that is not `provision-<x>`, or an identity that is not
  active.

## What the operator sees

From the example above, `bootstrap-config` writes this into `vars.sh`:

```bash
# GOVC_URL (required): vCenter SDK URL of the target
# Example: https://vcenter.example.test/sdk
# GOVC_URL=

# TARGET_FOLDER (optional): VM folder of the service hosts
# TARGET_FOLDER=
```

The operator writes the value after `=` and removes the leading `#` and the space after it. So
write descriptions that let an operator fill the value without reading the
repository's code: what it is, and its form when that is not obvious.

## Check the declaration

Against a scratch root, never the operator's tree:

```bash
export ASSEMBLEIARR_CONFIG_DIR="$(mktemp -d)/assembleiarr"
platformctl bootstrap-config --env lab --target vsphere-lab \
  --declaration ./shared-root.yaml --dry-run
platformctl bootstrap-config --env lab --target vsphere-lab \
  --declaration ./shared-root.yaml
platformctl doctor --env lab --declaration ./shared-root.yaml
```

The first command exits `1` and prints the problem when the file is
invalid, and lists the paths it would create when it is valid. On a fresh
skeleton, `doctor` exits `1` and lists each required key and credential
still missing: that list is what an operator will have to fill.

To prove that the repository's own loader reads such a tree, fill the
values in the scratch tree, create the credentials with `platformctl
credential`, and run the loader against it. That test belongs in the
repository, not in `platformctl`.

## Changing the declaration

- Adding a key: add it to `shared-root.yaml`. On an existing tree,
  `bootstrap-config` does not change the file; it prints the placeholder
  lines to add, and `doctor` reports the key until the operator adds it.
- Removing a key: remove it from the declaration; the operator deletes the
  line or file by hand (`platformctl` never deletes).
- The format version changes only when the standard changes it; see
  "Key declaration" in the standard.
