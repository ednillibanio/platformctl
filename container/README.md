# platformctl container test lab

This directory owns the disposable Docker Compose lab used to validate
platformctl outside any service-provisioning repository. It runs the exact
same provisioning contract as the [Vagrant test lab](../vagrant/README.md),
reusing `vagrant/scripts/provision-test-vm.sh` unchanged, and works natively
on `linux/arm64` (including Apple Silicon hosts), where VirtualBox/VMware
support is limited or unavailable.

For the cross-lab comparison and safety boundary, start with
[`docs/lab-environments.md`](../docs/lab-environments.md). For development
commands outside the lab, see [`docs/development.md`](../docs/development.md).

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
./lab.sh up
```

`lab.sh` wraps `docker compose` (falling back to the standalone
`docker-compose` binary automatically if the Compose CLI plugin isn't
installed) with Vagrant-like lifecycle commands — every command is
repeatable and idempotent:

| Command | Equivalent to |
| --- | --- |
| `./lab.sh up` | `vagrant up` |
| `./lab.sh status` | inspect the persisted status |
| `./lab.sh render` | print the rendered YAML |
| `./lab.sh ssh` | `vagrant ssh` (interactive shell) |
| `./lab.sh destroy` | `vagrant destroy -f` |

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
./lab.sh status
./lab.sh render
```

Or open a shell inside the lab, same as `vagrant ssh`:

```bash
./lab.sh ssh
```

Re-run provisioning with `./lab.sh up`; the `lab-home` volume persists, so
the Go bootstrap stays idempotent. Destroy the disposable lab, including its
volume, with:

```bash
./lab.sh destroy
```

Operator configuration outside this lab is unaffected.

## Command mapping

| Vagrant | `lab.sh` | Raw `docker compose` |
| --- | --- | --- |
| `vagrant validate` | — | `docker compose config` |
| `vagrant up` | `./lab.sh up` | `docker compose up --build --exit-code-from platformctl-lab` |
| `vagrant provision` | `./lab.sh up` (re-run; volume persists) | same as above |
| `vagrant ssh -c "cat ..."` | `./lab.sh status` / `./lab.sh render` | `docker compose run --rm --entrypoint cat platformctl-lab <path>` |
| `vagrant ssh` | `./lab.sh ssh` | `docker compose run --rm --entrypoint bash platformctl-lab` |
| `vagrant destroy -f` | `./lab.sh destroy` | `docker compose down -v` |

## Configuration

Copy `.env.example` to the ignored `.env`. Variables:

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `PLATFORMCTL_LAB_BASE_IMAGE` | No | `ubuntu:24.04` | Base image used by the lab Dockerfile. |
| `PLATFORMCTL_LAB_CPUS` | No | `2` | Compose CPU limit. |
| `PLATFORMCTL_LAB_MEMORY` | No | `2g` | Compose memory limit. |

These values control disposable compute only and must not contain credentials.
The lab always uses the committed `examples/platform-home` configuration.

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
