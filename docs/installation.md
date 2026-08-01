# Installation

No platformctl release has been published yet. During the `v1alpha1`
foundation phase, install from the `lab` branch. The bootstrap uses an existing
compatible Go installation or downloads Go 1.26.5 into the ignored `.tools/`
directory and verifies its official checksum.

## Linux

Prerequisites: Git, Bash, `curl`, `tar`, and `sha256sum`.

```bash
git clone https://github.com/ednillibanio/platformctl.git
cd platformctl
git switch lab

./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl

install -d "${HOME}/.local/bin"
install -m 0755 bin/platformctl "${HOME}/.local/bin/platformctl"
export PATH="${HOME}/.local/bin:${PATH}"

platformctl --help
platformctl init
platformctl environment init lab
```

The normal configuration location is `${XDG_CONFIG_HOME}/platformctl` when
`XDG_CONFIG_HOME` is set, otherwise `~/.config/platformctl`.

## Windows

Prerequisites: Git for Windows and PowerShell 5.1 or newer.

```powershell
git clone https://github.com/ednillibanio/platformctl.git
Set-Location platformctl
git switch lab

.\scripts\bootstrap-dev.ps1
.\scripts\go.ps1 test ./...
.\scripts\go.ps1 build -o bin\platformctl.exe .\cmd\platformctl

$env:Path = "$(Resolve-Path .\bin);$env:Path"
platformctl --help
platformctl init
platformctl environment init lab
```

The `PATH` change above applies to the current PowerShell session. To persist
it for the current user:

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

Open a new terminal afterward. The default configuration location is:

```text
%AppData%\platformctl
```

To share this location with a disposable lab controller, expose it
explicitly:

```powershell
$env:PLATFORMCTL_HOME = Join-Path $env:APPDATA "platformctl"
```

See [Lab environments](lab-environments.md) for the available lab
mechanisms.

## macOS

The bootstrap supports Apple Silicon (`arm64`) and Intel (`amd64`).
Prerequisites: Git, Bash, `curl`, `tar`, and `shasum`.

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
platformctl init
platformctl environment init lab
```

The default macOS configuration location follows `os.UserConfigDir`:

```text
~/Library/Application Support/platformctl
```

To share this location with a disposable lab controller, expose it
explicitly:

```bash
export PLATFORMCTL_HOME="${HOME}/Library/Application Support/platformctl"
```

See [Lab environments](lab-environments.md) for the available lab
mechanisms.

Add `~/.local/bin` and `PLATFORMCTL_HOME` to the preferred shell profile when
they should persist across terminal sessions.

## Override the configuration location

All supported operating systems accept an absolute `PLATFORMCTL_HOME`:

```bash
PLATFORMCTL_HOME=/absolute/path platformctl validate --env lab
```

```powershell
$env:PLATFORMCTL_HOME = "D:\platformctl-config"
platformctl validate --env lab
```

The override is useful when multiple lab or automation controllers share one
operator configuration tree. See
[Vagrant controller integration](vagrant-controller-integration.md) for the
live-mount recipe.

## Updating during the foundation phase

Until versioned releases exist:

```bash
git switch lab
git pull --ff-only origin lab
./scripts/bootstrap-dev.sh
./scripts/go.sh test ./...
./scripts/go.sh build -o bin/platformctl ./cmd/platformctl
```

On Windows, use the PowerShell scripts and `.exe` output shown above. The
configuration tree is external to the checkout and is not replaced during an
update.
