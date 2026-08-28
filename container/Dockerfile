# syntax=docker/dockerfile:1

# Disposable Linux image for the platformctl container test lab. Mirrors the
# Vagrant lab's Ubuntu box family; installs only what
# vagrant/scripts/provision-test-vm.sh needs to bootstrap Go and run it.

ARG BASE_IMAGE=ubuntu:24.04
FROM ${BASE_IMAGE}

RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        bash ca-certificates curl tar coreutils \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --shell /bin/bash lab

USER lab
WORKDIR /workspace

ENTRYPOINT ["/bin/bash", "/workspace/vagrant/scripts/provision-test-vm.sh"]
