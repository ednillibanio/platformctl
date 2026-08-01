# platformctl container test lab

This directory owns the disposable Docker Compose lab used to validate
platformctl outside any service-provisioning repository. It runs the exact
same provisioning contract as the [Vagrant test lab](../vagrant/README.md),
reusing `vagrant/scripts/provision-test-vm.sh` unchanged, and works natively
on `linux/arm64` (including Apple Silicon hosts), where VirtualBox/VMware
support is limited or unavailable.

## Service

| Service | Role | Default base image |
| --- | --- | --- |
| `platformctl-lab` | Build and configuration test environment | `ubuntu:24.04` |

The repository checkout is mounted read-only at `/workspace`. It does not
clone adjacent repositories or receive host credentials.

## Quick start

```bash
cd container
cp .env.example .env
docker compose config
docker compose up --build
```

Use the standalone `docker-compose` binary in place of `docker compose` if
the Compose CLI plugin is not installed; both read the same
`docker-compose.yml`.

Provisioning performs the same six steps as the Vagrant lab, via the shared
script:

1. local Go bootstrap with checksum verification;
2. `go vet ./...`;
3. `go test ./...`;
4. Linux binary build;
5. validation of `examples/platform-home`;
6. deterministic YAML and JSON renders.

Toolchains, caches, the binary, status, and rendered output live in the named
volume `lab-home`, mounted at `/home/lab` — never in the mounted checkout:

```text
/home/lab/.local/share/platformctl-dev/
├── .tools/
├── .cache/
├── bin/platformctl
└── rendered/{lab.yaml,lab.json}

/home/lab/.local/state/platformctl-lab/status
```

Inspect the result:

```bash
docker compose run --rm --entrypoint cat platformctl-lab /home/lab/.local/state/platformctl-lab/status
docker compose run --rm --entrypoint cat platformctl-lab /home/lab/.local/share/platformctl-dev/rendered/lab.yaml
```

Re-run provisioning with `docker compose up --build`; the `lab-home` volume
persists, so the Go bootstrap stays idempotent. Destroy the disposable lab,
including its volume, with:

```bash
docker compose down -v
```

Operator configuration outside this lab is unaffected.

## Command mapping

| Vagrant | Container |
| --- | --- |
| `vagrant validate` | `docker compose config` |
| `vagrant up` | `docker compose up --build` |
| `vagrant provision` | `docker compose up --build` (re-run; volume persists) |
| `vagrant ssh -c "cat ..."` | `docker compose run --rm --entrypoint cat platformctl-lab <path>` |
| `vagrant destroy -f` | `docker compose down -v` |

## Configuration

Copy `.env.example` to the ignored `.env`. Variables:

- `PLATFORMCTL_LAB_BASE_IMAGE`
- `PLATFORMCTL_LAB_CPUS`
- `PLATFORMCTL_LAB_MEMORY`

Run `tests/test-compose-contract.sh` for static checks that do not require
Docker.

## Vagrant vs. container

| | `vagrant/` | `container/` |
| --- | --- | --- |
| Mechanism | VM (VirtualBox/VMware) | Docker Compose |
| Apple Silicon | Provider support limited/unavailable | Native `linux/arm64`, no emulation |
| Isolation | VM user's home | Named Docker volume |
| Teardown | `vagrant destroy -f` | `docker compose down -v` |

Both share the identical provisioning script and validation contract; see
[Lab environments](../docs/lab-environments.md) for the full comparison.

This test lab intentionally validates committed examples rather than private
operator configuration. There is currently no container equivalent of the
[Vagrant controller integration guide](../docs/vagrant-controller-integration.md);
that live-mount pattern remains specific to Vagrant's `synced_folder`.
