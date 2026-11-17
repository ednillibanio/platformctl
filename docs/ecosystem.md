# Ecosystem and integrations

`platformctl` is the operator-side platform tooling of a multi-repository
automation ecosystem (platform-catalog type "Platform tooling"). This document
defines ownership, handoffs, and current integration status. It does not
replace the authoritative documentation of the related repositories.

## Current integration status

### Shared root (current)

`platformctl bootstrap-config`, `credential` and `doctor` create and check
the shared root of the `provision-*` repositories,
`~/.config/assembleiarr/<env>/`, on every controller. The contract is the
platform-catalog [provision shared root standard](https://github.com/ednillibanio/platform-catalog/blob/main/docs/standards/provision-shared-root-standard.md) (ADR-0005, with the
identities of ADR-0007). The `provision-*` repositories read that tree with
their own loaders; `platformctl` never runs them.

```text
shared-root.yaml (per provision-* repo) --> platformctl bootstrap-config --> <root>/<env>/ skeleton
operator edits vars.sh / hosts/<c>.sh, platformctl credential --> values and secrets
platformctl doctor --> problems, or a JSON summary to compare controllers
provision-* loader <-- reads <root>/<env>/
```

Adoption is in progress. Each consumer has an adaptation issue:
al-rr/provision-semaphoreui#86 and #88 (`shared-root.yaml`, consumer name,
`gen-credential` replaced by `platformctl credential`, own identity),
al-rr/provision-nexus3#100 and #101, al-rr/provision-template#13
and ednillibanio/platform-labs#128 (test controllers). The controllers
prepared by `al-rr/platform-modules` already install the pinned
`platformctl` release on Linux and macOS (al-rr/platform-modules#337,
done); its operator guide for the shared root on a controller is
[Provision Shared Root Guide](https://github.com/al-rr/platform-modules/blob/lab/docs/en/provision-shared-root-guide.md).
[`al-rr/provision-vsphere-platform`](https://github.com/al-rr/provision-vsphere-platform),
which owns the vCenter access of the platform consumers, is the first
repository with a committed `shared-root.yaml` (its target secrets under
the `vcenter-access-admin` identity).

### YAML model (paused)

The handoff of the paused `PLATFORMCTL_HOME` model is:

```text
PLATFORMCTL_HOME -> platformctl validate -> platformctl render -> YAML or JSON
PLATFORMCTL_HOME -> platformctl export   -> shell exports (for example PM_*)
```

The `export` projection is the contract that `platform-modules` is planned to
consume (ADR-0002/0003). That program was paused on 2026-09-23, so no
adoption is planned; `platform-modules` keeps its own configuration.

The rendered document is ready for downstream consumption, but this repository
does not currently ship adapters for `provision-platform-environments`,
`provision-semaphoreui`, `provision-nexus3`, `platform-modules`, or another
`provision-*` repository. Those repositories also do not currently invoke
`platformctl` in their committed workflows.

Therefore:

- the repository relationships below are real ownership and data-contract
  relationships;
- automatic transfer from a render into consumer manifests, inventories, API
  calls, or secret directories is **planned**, not implemented;
- operators must not assume that editing `PLATFORMCTL_HOME` changes GitHub,
  SemaphoreUI, Nexus, a VM, or a container.

## Dependency and link boundary

`platformctl` does not declare a source, build, or runtime dependency on any
of the repositories in this guide. In particular, it has no Git submodules,
does not clone repositories, invoke their CLIs, or call their APIs. Its Go
module dependencies are limited to Cobra and the YAML parser; the container
and Vagrant labs are development-only dependencies, not integration adapters.

The repository links below deliberately point to the related repositories'
`lab` branches, where their pre-promotion integration contracts are reviewed.
On 2026-08-28, each linked repository, branch, and referenced document was
verified through GitHub; the `platform-modules` and `platform-catalog` links
were re-verified on 2026-09-23. `platform-catalog` has no `lab` branch, so its
links point to `main`. The related repositories use `main` as their default
branch, so use the linked `lab` material when evaluating a change that will be
integrated through `lab`; do not infer a runtime dependency from those links.

For a new adapter, start by treating the rendered YAML or JSON and the schemas
in [`schemas/v1alpha1/`](../schemas/v1alpha1/) as the only `platformctl`
contract. Open a scoped issue in the selected consumer repository, define and
test the field and credential mapping there, then deliver the adapter through
that repository's normal review flow. No repository currently performs this
step automatically.

## Repository responsibilities

| Repository | Authoritative responsibility | Real values? | Executes changes? | Relationship to `platformctl` |
| --- | --- | --- | --- | --- |
| [`platform-catalog`](https://github.com/ednillibanio/platform-catalog) | Catalog and governance: what exists, ownership, cross-repository standards, and architecture decisions (ADRs). | No operational values. | No. | Governs environment names and the configuration contract; no runtime integration. |
| [`al-rr/platform-modules`](https://github.com/al-rr/platform-modules) | Reusable Bash, Ansible, Terraform, Packer, and container automation modules (successor to the archived `infra-gitops`). | Its own operator tree (`~/.config/platform-modules`). | Library content is executed by consumers. | Installs `platformctl` on the controllers (al-rr/platform-modules#337); the `export` projection planned for it (ADR-0002/0003) is paused. |
| [`platformctl`](https://github.com/ednillibanio/platformctl) | Creates and checks the `provision-*` shared root on every controller; the paused YAML model of layered values per environment. | Yes, in the operator's trees, never in Git. | No infrastructure or remote APIs. | Implements the catalog shared-root standard; produces the neutral YAML/JSON handoff of the paused model. |
| [`al-rr/provision-platform-environments`](https://github.com/al-rr/provision-platform-environments) | Declarative GitHub Environment variables and secret names, reconciled through `platform-environments-ctl`. | Non-secret values in manifests; secret values arrive from local files at apply time. | Yes, against the GitHub API after operator approval. | Candidate consumer for artifact/Nexus values and credential references; adapter not implemented. |
| [`al-rr/provision-semaphoreui`](https://github.com/al-rr/provision-semaphoreui) | SemaphoreUI container lifecycle and declarative Project/Repository/Inventory/Environment/Template reconciliation. | Secret-free manifests plus operator-supplied secret files. | Yes, against Docker and the Semaphore API. | Candidate consumer for host, Nexus, and deployment values; adapter not implemented. |
| [`al-rr/provision-nexus3`](https://github.com/al-rr/provision-nexus3) | Nexus3 lifecycle, repository structure, roles, and technical accounts through Ansible and SemaphoreUI workflows. | Environment inventory and external credentials. | Yes, against Nexus3/Docker. | Defines authoritative Nexus provisioning contracts that a future `platformctl` service model can describe for consumers. |
| Other `provision-*` repositories | Concrete lifecycle for one workload or platform component. | Usually yes, scoped to that consumer. | Yes. | May consume selected hosts/services from rendered output through an explicit adapter. |

## Intended flow

```mermaid
flowchart LR
    C["platform-catalog<br/>catalog and governance"] -. governs .-> P
    I["platform-modules<br/>reusable automation"] --> P
    H["PLATFORMCTL_HOME<br/>private desired values"] --> R["platformctl render<br/>deterministic YAML/JSON"]
    R --> A["consumer-specific adapter<br/>not implemented here"]
    A --> P["provision-* or control repository"]
    P --> T["GitHub API / Semaphore API<br/>Nexus / VM / container"]
```

The adapter is intentionally explicit. A generic render cannot decide, for
example, whether a credential reference should become a GitHub Environment
secret file, a Semaphore `env` secret, a Semaphore Ansible-variable secret, or
an Ansible lookup. That mapping belongs to the consuming repository and must be
documented and validated there.

## Cross-repository contracts

### `platform-modules`

[`platform-modules`][pm-repo] is the reusable automation library and the
successor to the archived `infra-gitops`. Today its `modulesctl
bootstrap-config` manages the library's own configuration tree under
`~/.config/platform-modules` (layered `vars.sh`/`credentials.sh` files that
export `PM_*` variables). That tree is still separate from `PLATFORMCTL_HOME`.

The [platform configuration alignment program][alignment] explored changing
this relationship. It was paused on 2026-09-23, and these ADRs are on hold:

- [ADR-0001][adr-0001] makes `platformctl` the single source of
  per-environment values.
- [ADR-0002][adr-0002] defines a `platformctl export --format env --prefix PM_`
  projection that `platform-modules` can evaluate without code changes.
- [ADR-0003][adr-0003] migrates away from `bootstrap-config` in reversible
  stages.

Treat the two trees as independent. They are not aliases.

[pm-repo]: https://github.com/al-rr/platform-modules
[alignment]: https://github.com/ednillibanio/platform-catalog/blob/main/docs/architecture/platform-config-alignment.md
[adr-0001]: https://github.com/ednillibanio/platform-catalog/blob/main/docs/architecture/decisions/0001-platform-planes-and-responsibilities.md
[adr-0002]: https://github.com/ednillibanio/platform-catalog/blob/main/docs/architecture/decisions/0002-configuration-contract.md
[adr-0003]: https://github.com/ednillibanio/platform-catalog/blob/main/docs/architecture/decisions/0003-migrate-modulesctl-bootstrap-config.md

Authoritative starting points:

- [platform-modules README](https://github.com/al-rr/platform-modules/blob/lab/README.md)
- [Ansible external configuration](https://github.com/al-rr/platform-modules/blob/lab/docs/en/ansible-external-configuration.md)
- [Project-local Ansible environment](https://github.com/al-rr/platform-modules/blob/lab/docs/en/ansible-project-environment.md)

### `provision-platform-environments`

[`provision-platform-environments`](https://github.com/al-rr/provision-platform-environments)
owns GitHub Environments such as `artifact-lab`. Its committed manifests carry
non-secret values including Nexus endpoints/repository names and declare secret
names. At apply time, same-named local files provide secret values.

Authoritative starting points:

- [Repository README](https://github.com/al-rr/provision-platform-environments/blob/lab/README.md)
- [`ArtifactEnvironmentManifest` reference](https://github.com/al-rr/provision-platform-environments/blob/lab/docs/reference/manifest.md)
- [Nexus technical-account contract](https://github.com/al-rr/provision-platform-environments/blob/lab/docs/nexus-account-contract.md)

A future adapter may select non-secret Nexus values from a rendered service and
materialize the consumer manifest. Credential resolution must remain a separate
apply-time operation: `platformctl` itself never opens `env` or `file`
references, and the consumer requires raw, mode-`0600`, same-named files.

### `provision-semaphoreui`

[`provision-semaphoreui`](https://github.com/al-rr/provision-semaphoreui)
owns both the SemaphoreUI container and declarative API reconciliation through
`semaphoreui-ctl`. Its manifests distinguish process-environment secrets from
Ansible-variable secrets; they are not interchangeable.

Authoritative starting points:

- [Repository README](https://github.com/al-rr/provision-semaphoreui/blob/lab/README.md)
- [`semaphoreui-ctl` CLI reference](https://github.com/al-rr/provision-semaphoreui/blob/lab/docs/cli-reference.md)
- [Java projects lab runbook](https://github.com/al-rr/provision-semaphoreui/blob/lab/docs/lab-java-projects-runbook.md)

Shared root: its `hosts/` loader reads `targets/<t>/vars.sh`,
`targets/<t>/hosts/provision-semaphoreui.sh`, and today the vCenter account
directly under `credentials/target-<t>/` (legacy, reported by `doctor`).
Adoption: al-rr/provision-semaphoreui#86 and #88.

For the paused YAML model, a future adapter must map rendered host/service
data into the correct Inventory and Environment fields and map each credential to the secret type expected by
the consumed playbook. A direct field-name copy is not a sufficient contract.

### `provision-nexus3`

[`provision-nexus3`](https://github.com/al-rr/provision-nexus3) owns real Nexus3
deployment, repository configuration, roles, and technical accounts. It
consumes the `assembleiarr.services.nexus3` Ansible Collection from
`platform-modules`.

Authoritative starting points:

- [Repository README](https://github.com/al-rr/provision-nexus3/blob/lab/README.md)
- [Ansible configuration reference](https://github.com/al-rr/provision-nexus3/blob/lab/ansible/README.md)
- [Lab provisioning runbook](https://github.com/al-rr/provision-nexus3/blob/lab/docs/lab-provisioning-runbook.md)

Shared root: nothing reads it yet; adoption with its Terraform root in
al-rr/provision-nexus3#100 and #101.

The Nexus account contract deliberately separates the CI account used by
GitHub workflows from the deployment account used by SemaphoreUI. A future
`platformctl` model or adapter must preserve that separation and must never
collapse both into an administrative credential.

## Example future handoff

The following is an architectural example, not an implemented command:

```text
platformctl Service "nexus"
  non-secret values: URL, OCI registry, Maven/raw repository names
  credential refs:  CI account, deployment account (separate)

platformctl Host "application-dev"
  addresses: deployment target CIDRs

                    render --format json
                              |
             +----------------+----------------+
             |                                 |
  platform-environments adapter      semaphoreui adapter
  GitHub variables + CI secret       Inventory/Environment + deploy secret
```

Each adapter should:

1. validate the selected render and expected keys;
2. fail closed on missing or ambiguous values;
3. resolve only the credential references it owns;
4. write consumer-specific temporary files with restrictive permissions;
5. support a read-only check/diff/dry-run path where the consumer provides one;
6. avoid committing generated manifests or secret material;
7. document the exact mapping and cleanup behavior in the consumer repository.

## What `platformctl` is not

- It is not the public catalog or ownership registry.
- It is not the `platform-modules` reusable automation library.
- It is not a GitHub Environment, SemaphoreUI, Nexus, Terraform, or Ansible
  client.
- It does not automatically synchronize other repositories.
- It does not resolve referenced credentials or decide consumer secret types.
- Its disposable Vagrant/container labs validate this repository only; they are
  not managed infrastructure environments.

For the render model itself, continue with the
[Configuration reference](configuration.md). For internal boundaries, see
[Architecture](architecture.md).
