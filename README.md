# platformctl

Cross-platform control plane for infrastructure environment configuration and
provisioning orchestration.

The current foundation release is intentionally configuration-only. It creates,
validates, merges, and renders private operator configuration without invoking
Vagrant, Ansible, Terraform, govc, glpictl, or any service provisioner.

## Development setup

Linux and macOS:

```bash
./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
```

Windows PowerShell:

```powershell
.\scripts\bootstrap-dev.ps1
.\scripts\go.ps1 test ./...
.\scripts\go.ps1 build -o bin\platformctl.exe .\cmd\platformctl
```

The bootstrap uses an existing Go installation when available. Otherwise it
downloads the pinned Go toolchain into the ignored `.tools/` directory,
verifies its official SHA-256 checksum, and downloads module dependencies.
Build caches remain under the ignored `.cache/` directory.

## Quick start

```bash
platformctl init
platformctl environment init lab
platformctl validate --env lab
platformctl render --env lab
platformctl render --env lab --format json
```

`PLATFORMCTL_HOME` overrides the configuration root. Without it, platformctl
uses the operating system's user configuration directory: normally
`$XDG_CONFIG_HOME/platformctl` or `~/.config/platformctl` on Linux,
`%AppData%\platformctl` on Windows, and
`~/Library/Application Support/platformctl` on macOS.

See [Installation](docs/installation.md), [Architecture](docs/architecture.md),
[Configuration reference](docs/configuration.md), the
[Vagrant controller integration guide](docs/vagrant-controller-integration.md),
the complete [example](examples/platform-home/), and
[implementation progress](docs/progress.md).

## Disposable lab environments

The repository owns two standalone, disposable Linux lab environments;
neither depends on `provision-glpi` or any other service repository. See
[Lab environments](docs/lab-environments.md) for the comparison.

### Vagrant

```bash
cd vagrant
cp config.local.example.yaml config.local.yaml
vagrant validate
vagrant up
```

See the [Vagrant lab guide](vagrant/README.md) for providers, isolation,
artifacts, diagnostics, and cleanup.

### Docker

```bash
cd container
cp .env.example .env
docker compose config
docker compose up --build
```

See the [container lab guide](container/README.md) for isolation, artifacts,
diagnostics, and cleanup. Works natively on Apple Silicon (`linux/arm64`).
