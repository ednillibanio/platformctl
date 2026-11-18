#!/usr/bin/env bash
# @file scripts/release-build.sh
# @brief Builds the release binaries of platformctl and their SHA256SUMS.
# @description
#   Builds the release binaries of platformctl and their SHA256SUMS.
#
#   Usage: scripts/release-build.sh <version> <out-dir>
#
#     <version>  embedded in the binary (platformctl --version), for example
#                v0.1.0 for a tag or dev-<commit> for a test build
#     <out-dir>  created if missing; must not already hold release files
#
#   Writes platformctl-<os>-<arch> for linux and darwin on amd64 and arm64
#   (CGO_ENABLED=0: static on Linux, only libSystem on macOS) and SHA256SUMS.
#   Uses scripts/go.sh, so the pinned toolchain policy of the repository
#   applies.
# @example
#   scripts/release-build.sh dev-"$(git rev-parse --short=12 HEAD)" .cache/dist
# @arg $1 string Version embedded in the binary.
# @arg $2 path Output directory.

set -euo pipefail

usage() { sed -n '/^# @description$/,/^# @example$/{/^# @/d;s/^# \{0,3\}//;p;}' "$0"; }
if [[ ${1:-} == -h || ${1:-} == --help ]]; then
  usage
  exit 0
fi
if [[ $# -ne 2 ]]; then
  usage >&2
  exit 2
fi
version=$1
out=$2

if [[ ! $version =~ ^(v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?|dev-[0-9a-f]{7,40})$ ]]; then
  echo "invalid version: ${version} (want vX.Y.Z[-pre] or dev-<commit>)" >&2
  exit 2
fi

repo=$(cd "$(dirname "$0")/.." && pwd)
mkdir -p "$out"
out=$(cd "$out" && pwd)
if compgen -G "${out}/platformctl-*" >/dev/null || [[ -e ${out}/SHA256SUMS ]]; then
  echo "${out} already holds release files" >&2
  exit 1
fi

ldflags="-s -w -X github.com/ednillibanio/platformctl/internal/cli.Version=${version}"
names=()
for os in linux darwin; do
  for arch in amd64 arm64; do
    name="platformctl-${os}-${arch}"
    (cd "$repo" && CGO_ENABLED=0 GOOS=$os GOARCH=$arch \
      ./scripts/go.sh build -trimpath -ldflags "$ldflags" -o "${out}/${name}" ./cmd/platformctl)
    names+=("$name")
    echo "built ${name}"
  done
done

cd "$out"
if command -v sha256sum >/dev/null; then
  sha256sum "${names[@]}" >SHA256SUMS
else
  shasum -a 256 "${names[@]}" >SHA256SUMS
fi
echo "wrote ${out}/SHA256SUMS"
