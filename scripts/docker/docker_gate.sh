#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: scripts/docker/docker_gate.sh <make-target> [extra make args...]" >&2
  exit 2
fi

target="$1"
shift

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
image="${DOCKER_IMAGE:-$(basename "$repo_root")-toolchain:local}"
go_version="${GO_VERSION:-1.23}"
golangci_lint_version="${GOLANGCI_LINT_VERSION:-v2.1.6}"
govulncheck_version="${GOVULNCHECK_VERSION:-v1.1.4}"

export DOCKER_BUILDKIT=1

"$repo_root/scripts/docker/check_toolchain.sh"

docker buildx build \
  --load \
  --target toolchain \
  --build-arg "GO_VERSION=$go_version" \
  --build-arg "GOLANGCI_LINT_VERSION=$golangci_lint_version" \
  --build-arg "GOVULNCHECK_VERSION=$govulncheck_version" \
  --tag "$image" \
  "$repo_root"

docker run --rm \
  --workdir /workspace \
  --volume "$repo_root:/workspace" \
  --volume go-build-cache:/root/.cache/go-build \
  --volume go-mod-cache:/go/pkg/mod \
  --env "GOWORK=${GOWORK:-off}" \
  --env "XLIB_CONTEXT=${XLIB_CONTEXT:-docker_toolchain}" \
  --env "VERSION=${VERSION:-}" \
  --env "DOWNSTREAM=${DOWNSTREAM:-}" \
  --env "XLIB_ENABLE_VULNCHECK=${XLIB_ENABLE_VULNCHECK:-}" \
  --env "CI=${CI:-}" \
  --env "GITHUB_ACTIONS=${GITHUB_ACTIONS:-}" \
  --env "GIT_CONFIG_COUNT=1" \
  --env "GIT_CONFIG_KEY_0=safe.directory" \
  --env "GIT_CONFIG_VALUE_0=/workspace" \
  "$image" make "$target" "$@"
