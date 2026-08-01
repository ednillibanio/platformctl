# Vagrant controller integration

This guide mounts the operator's live platformctl configuration into a Linux
Vagrant controller. It is intended for lab and development workflows where
values change frequently and must become visible in the VM immediately.

## Supported behavior

The host remains the source of truth:

```text
host platformctl home  <-------------->  /home/vagrant/.config/platformctl
                           Vagrant
                        synced_folder
```

When platformctl runs as the `vagrant` user, it automatically discovers the
guest path. No guest-side `PLATFORMCTL_HOME` is required.

This integration currently provides:

- live, bidirectional configuration visibility;
- automatic platformctl home discovery inside the controller;
- `platformctl validate` and `platformctl render` when the Linux binary is
  installed in the controller.

It does not yet install platformctl or make `infra-gitops` and `provision-*`
consume the rendered model. Those are separate adapters planned after the
configuration contract stabilizes.

## 1. Prepare the host

Install platformctl using the instructions for
[Windows, Linux, or macOS](installation.md).

Windows PowerShell:

```powershell
$env:PLATFORMCTL_HOME = Join-Path $env:APPDATA "platformctl"
platformctl init
platformctl environment init lab
platformctl validate --env lab
Test-Path $env:PLATFORMCTL_HOME
```

Linux:

```bash
export PLATFORMCTL_HOME="${XDG_CONFIG_HOME:-${HOME}/.config}/platformctl"
platformctl init
platformctl environment init lab
platformctl validate --env lab
```

macOS:

```bash
export PLATFORMCTL_HOME="${HOME}/Library/Application Support/platformctl"
platformctl init
platformctl environment init lab
platformctl validate --env lab
```

Set the variable in the same terminal that will run Vagrant.

## 2. Copy the portable Vagrantfile blocks

Copy this block near the top of the target `Vagrantfile`, before
`Vagrant.configure`:

```ruby
platformctl_host_home = ENV.fetch('PLATFORMCTL_HOME', '').strip

abort(<<~ERROR) if platformctl_host_home.empty?
  PLATFORMCTL_HOME is required in the host shell that runs Vagrant.
ERROR

platformctl_host_home = File.expand_path(platformctl_host_home)
abort("PLATFORMCTL_HOME does not exist: #{platformctl_host_home}") unless
  File.directory?(platformctl_host_home)
```

Copy this block inside each controller machine definition:

```ruby
machine.vm.synced_folder platformctl_host_home,
                         '/home/vagrant/.config/platformctl',
                         id: 'platformctl-config'
```

This is the recommended provider-neutral form. Do not set `type`; Vagrant will
select the best available synced-folder implementation. Vagrant mounts folders
as the guest SSH user by default, and some folder types do not support explicit
`owner` or `group` changes.

Complete example:

```ruby
platformctl_host_home = ENV.fetch('PLATFORMCTL_HOME', '').strip
abort('PLATFORMCTL_HOME is required.') if platformctl_host_home.empty?

platformctl_host_home = File.expand_path(platformctl_host_home)
abort("PLATFORMCTL_HOME does not exist: #{platformctl_host_home}") unless
  File.directory?(platformctl_host_home)

Vagrant.configure('2') do |config|
  config.vm.define 'controller', primary: true do |machine|
    machine.vm.box = 'bento/ubuntu-24.04'

    machine.vm.synced_folder platformctl_host_home,
                             '/home/vagrant/.config/platformctl',
                             id: 'platformctl-config'
  end
end
```

## 3. Optional explicit SMB on Windows

Use this only when automatic selection does not provide the desired Windows
performance:

```ruby
machine.vm.synced_folder platformctl_host_home,
                         '/home/vagrant/.config/platformctl',
                         id: 'platformctl-config',
                         type: 'smb'
```

SMB is bidirectional. Vagrant may request Administrator elevation and SMB
credentials. Never put `smb_password` in a Vagrantfile or repository.

## 4. Apply and verify

For a new controller:

```powershell
vagrant up controller
```

For an existing controller after changing its Vagrantfile:

```powershell
vagrant reload controller
```

Verify the live mount without requiring platformctl in the guest:

```powershell
vagrant ssh controller -c "test -f /home/vagrant/.config/platformctl/defaults.yaml"
vagrant ssh controller -c "find /home/vagrant/.config/platformctl -maxdepth 4 -type f -print"
```

When the Linux platformctl binary is installed in the controller:

```powershell
vagrant ssh controller -c "platformctl validate --env lab"
vagrant ssh controller -c "platformctl render --env lab"
```

Edit a non-secret YAML value on the host and run validation again. The updated
value is immediately visible; `vagrant provision` is not required. Use
`vagrant reload` only when the Vagrantfile or mount definition changes.

## Placement in a service controller repository

Add the host-path block once near the target Vagrantfile's configuration
constants. Add the synced-folder block only inside the controller machine
definition(s) that consume platform configuration.

Do not add it to application or database targets. Controllers consume platform
configuration; service targets receive only values applied by their
provisioning workflow.

### Example: provision-glpi

One illustrative consumer, not part of this guide's own contract: in
`provision-glpi/vagrant/Vagrantfile`, add the host-path block once near its
configuration constants, and the synced-folder block only inside the existing
`glpi-controller` machine definition.

## Operational rules

- Initialize and edit configuration on the host.
- Treat the mount as read-only by convention even though it is bidirectional.
- Write renders, caches, logs, and generated artifacts outside the mount.
- Do not run `platformctl init` or `platformctl environment init` in a
  controller.
- Never commit the operator configuration tree.
- A file credential reference must point to a path visible inside the mounted
  tree when a downstream consumer needs to read it.
- Multiple controllers may mount the same tree and select different
  environments.

## Troubleshooting

### `PLATFORMCTL_HOME is required`

Set it in the same host terminal that launches Vagrant.

### Host path does not exist

Run `platformctl init` on the host and verify the environment variable.

### Configuration is not mounted after editing the Vagrantfile

Run `vagrant reload <controller-name>`. Provisioning is not necessary.

### SMB requests credentials

This is expected for explicit SMB. Let Vagrant prompt interactively and never
store the Windows password in the Vagrantfile.

### Changes inside the VM affect the host

Synced folders are normally bidirectional. Do not edit or generate files in the
mounted tree from a controller.

## Vagrant references

- [Synced folders](https://developer.hashicorp.com/vagrant/docs/synced-folders)
- [Synced-folder basic usage and options](https://developer.hashicorp.com/vagrant/docs/synced-folders/basic_usage)
- [SMB synced folders](https://developer.hashicorp.com/vagrant/docs/synced-folders/smb)
