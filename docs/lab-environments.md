# Lab environments

platformctl ships two disposable, self-owned Linux lab environments for
validating the project outside any service-provisioning repository. Both run
the identical contract — local Go bootstrap, `go vet`, `go test`, a Linux
build, and validate/render against
[`examples/platform-home`](../examples/platform-home) — via the same shared
script, [`vagrant/scripts/provision-test-vm.sh`](../vagrant/scripts/provision-test-vm.sh).
Pick whichever mechanism fits your host.

| | [`vagrant/`](../vagrant/README.md) | [`container/`](../container/README.md) |
| --- | --- | --- |
| Mechanism | VM (VirtualBox or VMware Desktop) | Docker Compose |
| Apple Silicon | Provider support limited or unavailable | Native `linux/arm64`, no emulation |
| Isolation | VM user's home | Named Docker volume |
| Spin-up | `vagrant up` | `docker compose up --build` |
| Teardown | `vagrant destroy -f` | `docker compose down -v` |

Both intentionally validate committed examples rather than private operator
configuration, and neither depends on `infra-gitops`, `provision-*`, or any
other adjacent repository.

This page is about the disposable validation labs only. It is unrelated to
the live `PLATFORMCTL_HOME` mount feature described in
[Vagrant controller integration](vagrant-controller-integration.md), which
lets an external service or automation controller see an operator's real
configuration tree — that pattern is specific to Vagrant's `synced_folder`
and has no container equivalent today.
