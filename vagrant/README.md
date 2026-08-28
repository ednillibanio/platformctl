# platformctl Vagrant test lab

This directory owns the disposable Linux VM used to validate platformctl
outside any service-provisioning repository.

For the cross-lab comparison and safety boundary, start with
[`docs/lab-environments.md`](../docs/lab-environments.md). For development
commands outside the lab, see [`docs/development.md`](../docs/development.md).

## Machine

| VM | Role | Default box | Host ports |
| --- | --- | --- | --- |
| `platformctl-test` | Build and configuration test environment | `bento/ubuntu-24.04` | SSH `2266` |

The VM uses NAT only. It does not forward the host SSH agent, clone adjacent
repositories, or receive private keys. VMware Desktop is the default provider;
VirtualBox is also supported.

## Quick start

```bash
cd vagrant
cp config.local.example.yaml config.local.yaml
vagrant validate
vagrant up
```

Provisioning performs:

1. local Go bootstrap with checksum verification;
2. `go vet ./...`;
3. `go test ./...`;
4. Linux binary build;
5. validation of `examples/platform-home`;
6. deterministic YAML and JSON renders.

The checkout is mounted at `/home/vagrant/platformctl`, but toolchains, caches,
the binary, status, and rendered output stay under the VM user's home:

```text
~/.local/share/platformctl-dev/
├── .tools/
├── .cache/
├── bin/platformctl
└── rendered/{lab.yaml,lab.json}

~/.local/state/platformctl-lab/status
```

Inspect the result:

```bash
vagrant ssh
cat ~/.local/state/platformctl-lab/status
cat ~/.local/share/platformctl-dev/rendered/lab.yaml
```

Re-run provisioning with `vagrant provision`. Destroy the disposable VM with
`vagrant destroy -f`; operator configuration outside this lab is unaffected.

## Configuration

Copy `config.local.example.yaml` to the ignored `config.local.yaml`. Environment
variables override local YAML:

| YAML key | Environment override | Default | Constraint |
| --- | --- | --- | --- |
| `box` | `PLATFORMCTL_VAGRANT_BOX` | `bento/ubuntu-24.04` | Non-empty, no newlines. |
| `memory` | `PLATFORMCTL_VM_MEMORY` | `2048` | Positive integer MiB. |
| `cpus` | `PLATFORMCTL_VM_CPUS` | `2` | Positive integer. |
| `provider` | `PLATFORMCTL_VAGRANT_PROVIDER` | `vmware_desktop` | `virtualbox` or `vmware_desktop`. |
| `ssh_port` | `PLATFORMCTL_TEST_SSH_PORT` | `2266` | Integer from `1024` through `65535`. |

`PLATFORMCTL_VAGRANT_CONFIG` optionally selects a different local YAML file;
the default is `config.local.yaml` in this directory. Unknown YAML keys are
rejected. Environment-variable overrides are converted to the expected numeric
types where applicable and validated before Vagrant defines the machine.

Run `tests/test-vagrantfile-contract.sh` for static checks that do not require
Ruby, Vagrant, or a provider.

This test VM intentionally validates committed examples rather than private
operator configuration. To expose a live `PLATFORMCTL_HOME` to an automation
or service controller, use the separate
[Vagrant controller integration guide](../docs/vagrant-controller-integration.md).

For an Apple Silicon-compatible, Docker-based alternative with the same
build→vet→test→render contract, see the
[container lab](../container/README.md) or the
[lab environments comparison](../docs/lab-environments.md).
