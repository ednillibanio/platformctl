# Installation

This guide installs only the CLI binary; operator configuration remains in a
separate configuration home and is not replaced by an update. `lab` is an
integration branch for pre-promotion validation, not an installation channel.

The shared-root commands (`bootstrap-config`, `credential`, `doctor`) need a
Linux or macOS controller; on Windows they exit with "not supported". Every
controller that runs a `provision-*` repository needs the binary on its
`PATH`.

## Install a release (Linux, macOS)

Releases are published from `vX.Y.Z` tags of `main` by
[`.github/workflows/release.yml`](../.github/workflows/release.yml). Each
release has one binary per platform, `platformctl-<os>-<arch>` for `linux`
and `darwin` on `amd64` and `arm64` (built without cgo; static on Linux), and
a `SHA256SUMS` file. Until the first tag exists, build from source (next
sections).

Prerequisites: `curl`, and `sha256sum` (Linux) or `shasum` (macOS).

```bash
version=v0.1.0   # the release to install
os=$(uname -s | tr '[:upper:]' '[:lower:]')        # linux or darwin
arch=$(uname -m); case $arch in x86_64) arch=amd64 ;; aarch64) arch=arm64 ;; esac
base=https://github.com/ednillibanio/platformctl/releases/download/${version}

file=platformctl-${os}-${arch}

# Each step runs only if the previous one succeeded: a failed download or a
# missing checksum line stops before anything is installed.
cd "$(mktemp -d)" &&
  curl -fsSLO "${base}/${file}" &&
  curl -fsSLO "${base}/SHA256SUMS" &&
  grep -x "[0-9a-f]\{64\}  ${file}" SHA256SUMS > check &&
  if command -v sha256sum >/dev/null; then sha256sum -c check; else shasum -a 256 -c check; fi &&
  install -d "${HOME}/.local/bin" &&
  install -m 0755 "${file}" "${HOME}/.local/bin/platformctl" &&
  platformctl --version   # platformctl version v0.1.0
```

The checksum step prints `platformctl-<os>-<arch>: OK`; anything else
means the file was not verified and nothing was installed.

`${HOME}/.local/bin` must be on `PATH`. To update, repeat with the new
version; to remove, delete the binary. The configuration roots are not
touched. A build from source prints `platformctl version dev`.

To build the same release files locally (for example to test the
workflow), run `scripts/release-build.sh <version> <out-dir>`, where
`<version>` is `vX.Y.Z[-pre]` or `dev-<commit>`.

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

On Linux or macOS, check the shared-root commands against a scratch root, so
the real `~/.config/assembleiarr` is not touched:

```bash
scratch="$(mktemp -d)"
platformctl bootstrap-config --env lab --target vsphere-lab \
  --config-dir "${scratch}" --automation-key
platformctl doctor --env lab --config-dir "${scratch}"
rm -rf "${scratch}"
```

Both commands exit `0`. Without a declaration, both print a "no declaration
given" warning; `doctor` ends with `<scratch>/lab: 0 problem(s), 1
warning(s)` after the automation key fingerprint.

## Prepare the shared root

On a controller, create and check the real shared root with the workflow in
[Usage: Shared root](usage.md#shared-root-of-the-provision--repositories). It
writes under `~/.config/assembleiarr/<env>/` unless `ASSEMBLEIARR_CONFIG_DIR`
or `--config-dir` selects another root.

## Initialize operator configuration

This section is for the paused YAML model. After verification, remove any
temporary example override and initialize the normal operating-system
location:

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

## Update from `main`

Until releases exist, run from the checkout:

```bash
git switch main
git pull --ff-only origin main
./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
install -m 0755 bin/platformctl "${HOME}/.local/bin/platformctl"
```

Use the PowerShell wrappers and `.exe` output on Windows. Updating source and
the binary does not migrate or overwrite the external configuration tree.
Because the API is still `v1alpha1`, review release/merge notes and validate all
operator environments before replacing a previously used binary.

## Test the integration branch intentionally

`lab` is the integration branch where changes are validated before promotion to
`main`. It may contain work that is not yet part of the principal branch. Use it
only when participating in that validation workflow:

```bash
git fetch origin
git switch lab
git pull --ff-only origin lab
./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
```

Do not describe an installation from `lab` as a stable/default installation.
After validation and review, the repository promotes `lab` to `main` through a
separate pull request.

## Uninstall

Remove the installed binary from the path chosen above. The operator
configuration home is intentionally independent and is not removed
automatically. Back it up and delete it only through an explicit operator data
retention decision.
