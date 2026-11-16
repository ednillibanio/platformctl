# Shared root concepts

This page explains what the `provision-*` shared root is, why it exists, and
how `platformctl` fits in. For the steps on a controller see
[Usage: Shared root](usage.md#shared-root-of-the-provision--repositories);
for every flag see the [CLI reference](cli-reference.md). The normative
definition is the platform-catalog
[provision shared root standard](https://github.com/ednillibanio/platform-catalog/blob/main/docs/standards/provision-shared-root-standard.md)
(ADR-0005, ADR-0007); when this page and the standard disagree, the standard
wins.

## The problem it solves

Every `provision-*` repository needs values and secrets that belong to an
environment, not to the repository: the vCenter URL and datacenter, the VM
folder, the address of its host, a database password, the vCenter account
password, the SSH key that automation uses to reach the hosts. Without a
common place, each repository keeps them its own way, the same value is
typed in several places, and nothing checks that two controllers agree.

The shared root is that common place: one directory tree per environment on
each controller, with a fixed layout, a fixed file format, and one owner per
file. Every `provision-*` repository that has adopted the standard reads
it (see [Adoption](#adoption)); none of them writes values into another's
files.

## What `platformctl` does, and does not do

`platformctl` is a generic tool that implements the standard. It does not
know any `provision-*` repository: a repository describes what it reads in
its own `shared-root.yaml` (its [key declaration](shared-root-declaration.md)),
and `platformctl` works from those declarations.

| Command | What it does |
| --- | --- |
| `bootstrap-config` | Creates the directories and files of the tree, with one commented placeholder per declared key, and the automation key pair. |
| `credential` | Creates secret files: a random value, a literal, or a value read from standard input. |
| `doctor` | Checks the tree against the standard and the declarations, and reports every problem with its file and, where there is one, its line. |

It never:

- changes or deletes an existing file (a second run reports `exists`);
- reads or writes `state/` (Terraform state) or `backups/`;
- runs a provisioner, Terraform, Ansible or `govc`;
- copies anything between controllers (see
  [Several controllers](#several-controllers)).

## The tree

`<root>` is `--config-dir`, else `ASSEMBLEIARR_CONFIG_DIR`, else
`~/.config/assembleiarr`. One environment (`lab` here) with one target
(`vsphere-lab`), one consumer (`provision-example`) and, on a controller
that also runs `provision-vsphere-platform`, its identity
`vcenter-access-admin`:

```text
<root>/lab/                                   0700
├── targets/
│   └── vsphere-lab/                          one directory per target
│       ├── vars.sh                   0644    GOVC_*, TARGET_*, DISPOSABLE_*: shared by every consumer
│       └── hosts/
│           └── provision-example.sh  0644    HOST_* of one consumer
└── credentials/
    ├── automation/
    │   ├── ssh_ed25519               0600    automation private key, same on every controller
    │   └── ssh_ed25519.pub           0644
    ├── provision-example/            0700    secrets of one consumer, one file per secret
    │   └── EXAMPLE_DB_PASS           0600    (created by `credential`)
    └── target-vsphere-lab/           0700    secrets of the target, one directory per identity
        └── vcenter-access-admin/     0700    identity of provision-vsphere-platform
            └── GOVC_PASSWORD         0600    (created by `credential --identity`)
```

Two more directories belong to the consumers, never to `platformctl`:
`state/<consumer>/<t>/` (Terraform state, one holder controller per
consumer and target) and `backups/<consumer>/`.

### Who writes and who reads

| File | Written by | Read by |
| --- | --- | --- |
| `targets/<t>/vars.sh` | Operator, after `bootstrap-config` creates the placeholders | Every consumer that deploys on `<t>` |
| `targets/<t>/hosts/<consumer>.sh` | Operator, same way | That consumer only |
| `credentials/<consumer>/<NAME>` | `platformctl credential --consumer` | That consumer only |
| `credentials/target-<t>/<identity>/<NAME>` | `platformctl credential --target --identity` | Every consumer that runs as that identity on `<t>` |
| `credentials/automation/ssh_ed25519*` | `bootstrap-config --automation-key` (first controller) or `--automation-key-import` (others) | Ansible on the controller; the public key goes into the first-boot data of every host |

## File format

`vars.sh` and `hosts/<consumer>.sh` are sourced by shell scripts, so they
may only assign literal values. A line is a comment, a blank line, or
`KEY=value` / `export KEY=value` where the value is:

- unquoted, without spaces, shell characters, quotes or `~`;
- `'single quoted'`;
- `"double quoted"` without `$`, backquote, `\` or `"`.

No `$(...)`, no variables, no functions, no unquoted `~`, and nothing
after the value: a comment goes on its own line, so write

```bash
# vCenter of the lab
GOVC_URL=https://vcenter.example.test/sdk
```

and not `GOVC_URL=https://vcenter.example.test/sdk # vCenter of the lab`.
Each key is set once, in the file its prefix belongs to. A key that is still commented
(`# GOVC_URL=`) is a placeholder: for a consumer it is missing.

Secrets are never in these files. Each secret is its own file under
`credentials/`, holding only the value with no trailing newline, so it can
be read without sourcing anything.

## Consumers and identities

- A **consumer** is a `provision-*` repository; its name is the repository
  name. It owns `hosts/<consumer>.sh` and `credentials/<consumer>/`.
- An **identity** is an account on a target (for vCenter, a vCenter user
  with its own roles, ADR-0007). Target secrets are grouped by identity,
  so two repositories never share a password by accident.

An identity is `active` (usable), `reserved` (name kept, refused until the
catalog activates it) or `deprecated` (refused for new use). Today only
`vcenter-access-admin`, used by `provision-vsphere-platform`, is active.
By convention a `provision-*` repository's identity is named after the
repository, but a name counts only once it is a row of the standard's
Identities table: today `provision-nexus3` and `provision-semaphoreui` are
`reserved` rows, and a name that is not in the table is unknown.
`platformctl` mirrors the table and refuses every identity that is not
active.

## Several controllers

Each controller of an environment (the environment's controller, test
controllers, an operator workstation) keeps its own tree. The values and
the automation key pair must be the same on all of them:

1. On the first controller, run `bootstrap-config --automation-key`.
2. On every other controller, run `bootstrap-config` with
   `--automation-key-import -`, piping the private key from the first one
   over SSH, so the same pair is used everywhere.
3. Fill the values on each controller.
4. Compare: `platformctl doctor --env <env> --format json` prints the
   non-secret values, the credential names (never their values) and the
   key fingerprint; diff the output of two controllers.

There is no synchronization tool on purpose: the operator decides what is
copied, and secrets never leave a controller through `platformctl`.

## Lifecycle of a tree

1. A repository adds or changes its `shared-root.yaml`.
2. The operator runs `bootstrap-config` with every declaration used on that
   controller. New files get placeholders; an existing file that lacks a
   new key is reported with the lines to add.
3. The operator fills values and runs `credential` for the secrets.
4. `doctor` exits `0`; the consumer can run.
5. Later changes repeat from step 1. Nothing is ever overwritten, so a
   value is changed only by editing the file, and a secret is replaced only
   by deleting its file and running `credential` again.

## Adoption

Repositories that predate the standard keep working with their current
configuration until their adaptation issue is closed; the
[ecosystem page](ecosystem.md) lists the state of each. Whether a
repository's own loader reads a tree created by `platformctl` correctly is
tested in that repository.
