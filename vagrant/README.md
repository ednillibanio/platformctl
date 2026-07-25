# platformctl Vagrant test lab

This directory owns the disposable Linux VM used to validate platformctl
outside any service-provisioning repository.

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

- `PLATFORMCTL_VAGRANT_BOX`
- `PLATFORMCTL_VM_MEMORY`
- `PLATFORMCTL_VM_CPUS`
- `PLATFORMCTL_VAGRANT_PROVIDER`
- `PLATFORMCTL_TEST_SSH_PORT`
- `PLATFORMCTL_VAGRANT_CONFIG`

Run `tests/test-vagrantfile-contract.sh` for static checks that do not require
Ruby, Vagrant, or a provider.
