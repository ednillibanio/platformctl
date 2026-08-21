# Installation

No versioned `platformctl` release is published yet. During the `v1alpha1`
foundation phase, build the `lab` branch from source. This guide installs only
the CLI binary; operator configuration remains in a separate configuration
home and is not replaced by source updates.

## Toolchain policy

[`go.mod`](../go.mod) requires Go 1.24 or newer. The bootstrap:

1. uses an existing compatible Go installation when available;
2. otherwise downloads pinned Go 1.26.5 into the ignored `.tools/` directory;
3. verifies the official archive with a repository-pinned SHA-256 checksum;
4. downloads Go module dependencies;
5. keeps build/module caches under ignored `.cache/` paths through the wrapper.

Always use `scripts/go.sh` or `scripts/go.ps1` instead of invoking `go`
directly in this repository.

## Linux

Prerequisites:

- Git;
- Bash;
- `curl` and `tar` when the bootstrap must download Go;
- `sha256sum` or `shasum` for archive verification;
- network access to GitHub, `go.dev`, and the configured Go module proxy.

From a parent development directory:

```bash
git clone https://github.com/ednillibanio/platformctl.git
cd platformctl
git switch lab

./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
```

Install for the current user:

```bash
install -d "${HOME}/.local/bin"
install -m 0755 bin/platformctl "${HOME}/.local/bin/platformctl"
export PATH="${HOME}/.local/bin:${PATH}"
platformctl --help
```

Persist the `PATH` change in the appropriate shell profile if required. The
default configuration root is `$XDG_CONFIG_HOME/platformctl` when
`XDG_CONFIG_HOME` is set, otherwise normally `~/.config/platformctl`.

## macOS

The bootstrap supports Apple Silicon (`arm64`) and Intel (`amd64`).
Prerequisites are Git, Bash, `curl`, `tar`, and `shasum` (or `sha256sum`).

```bash
git clone https://github.com/ednillibanio/platformctl.git
cd platformctl
git switch lab

./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl

mkdir -p "${HOME}/.local/bin"
install -m 0755 bin/platformctl "${HOME}/.local/bin/platformctl"
export PATH="${HOME}/.local/bin:${PATH}"
platformctl --help
```

The native default configuration root follows Go's `os.UserConfigDir`:

```text
~/Library/Application Support/platformctl
```

Quote this path whenever it is supplied explicitly.

## Windows

Prerequisites:

- Git for Windows;
- PowerShell 5.1 or newer;
- network access to GitHub, `go.dev`, and the configured Go module proxy.

The fallback bootstrap supports Windows `amd64` and verifies the pinned Go ZIP
before extraction.

```powershell
git clone https://github.com/ednillibanio/platformctl.git
Set-Location platformctl
git switch lab

.\scripts\bootstrap-dev.ps1
.\scripts\go.ps1 test ./...
.\scripts\go.ps1 build -o bin\platformctl.exe .\cmd\platformctl

$env:Path = "$(Resolve-Path .\bin);$env:Path"
platformctl --help
```

The `PATH` update above affects only the current PowerShell process. To persist
the repository's `bin` directory for the current user:

```powershell
$PlatformctlBin = (Resolve-Path .\bin).Path
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")

if (($UserPath -split ";") -notcontains $PlatformctlBin) {
    [Environment]::SetEnvironmentVariable(
        "Path",
        "$PlatformctlBin;$UserPath",
        "User"
    )
}
```

Open a new terminal afterward. The default configuration root is:

```text
%AppData%\platformctl
```

## Verify the installed binary

First inspect the interface:

```bash
platformctl --help
platformctl render --help
```

Then use the checkout's committed, secret-free example. Run this from the
repository root:

```bash
PLATFORMCTL_HOME="${PWD}/examples/platform-home" \
  platformctl validate --env lab

PLATFORMCTL_HOME="${PWD}/examples/platform-home" \
  platformctl render --env lab --format json
```

On Windows PowerShell:

```powershell
$env:PLATFORMCTL_HOME = (Resolve-Path .\examples\platform-home).Path
platformctl validate --env lab
platformctl render --env lab --format json
Remove-Item Env:PLATFORMCTL_HOME
```

Expected validation output is `Environment "lab" is valid`. The JSON render
must contain `kind: RenderedEnvironment` and two hosts.

## Initialize operator configuration

After verification, remove any temporary example override and initialize the
normal operating-system location:

```bash
unset PLATFORMCTL_HOME
platformctl init
platformctl environment init lab
```

PowerShell:

```powershell
Remove-Item Env:PLATFORMCTL_HOME -ErrorAction SilentlyContinue
platformctl init
platformctl environment init lab
```

These commands create local files. Review [Usage](usage.md) before authoring a
real profile, service, host, or credential.

## Override the configuration location

All platforms accept an absolute `PLATFORMCTL_HOME`:

```bash
export PLATFORMCTL_HOME=/absolute/operator-owned/path
platformctl validate --env lab
```

```powershell
$env:PLATFORMCTL_HOME = "D:\platformctl-config"
platformctl validate --env lab
```

The override is useful for isolated tests or a host configuration tree shared
with a Vagrant controller. It is not a development cache setting; use
`PLATFORMCTL_DEV_ROOT` for toolchains/caches. See the
[CLI reference](cli-reference.md#configuration-home-selection) and
[Vagrant controller integration](vagrant-controller-integration.md).

## Update from `lab`

Until releases exist, run from the checkout:

```bash
git switch lab
git pull --ff-only origin lab
./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
install -m 0755 bin/platformctl "${HOME}/.local/bin/platformctl"
```

Use the PowerShell wrappers and `.exe` output on Windows. Updating source and
the binary does not migrate or overwrite the external configuration tree.
Because the API is still `v1alpha1`, review release/merge notes and validate all
operator environments before replacing a previously used binary.

## Uninstall

Remove the installed binary from the path chosen above. The operator
configuration home is intentionally independent and is not removed
automatically. Back it up and delete it only through an explicit operator data
retention decision.
