#!/usr/bin/env bash
set -euo pipefail

expected_module="github.com/ZoneCNH/resiliencx"
expected_pkg="pkg/resiliencx"

echo "checking resiliencx module identity..."
module_path="$(go list -m)"
if [[ "$module_path" != "$expected_module" ]]; then
  echo "ERROR: module path is $module_path, expected $expected_module" >&2
  exit 1
fi

if [[ ! -d "$expected_pkg" ]]; then
  echo "ERROR: missing public package directory: $expected_pkg" >&2
  exit 1
fi

if [[ -d pkg/templatex ]]; then
  echo "ERROR: stale template package directory remains: pkg/templatex" >&2
  exit 1
fi

if grep -R --line-number --fixed-strings \
  -e "github.com/ZoneCNH/xlib-standard" \
  -e "pkg/templatex" \
  -e "package templatex" \
  -- go.mod go.sum pkg internal contracts examples testkit docs Makefile 2>/dev/null; then
  echo "ERROR: stale template identity found" >&2
  exit 1
fi

if ! grep -Fq 'ModuleName = "github.com/ZoneCNH/resiliencx"' pkg/resiliencx/version.go; then
  echo "ERROR: pkg/resiliencx/version.go does not expose the resiliencx ModuleName" >&2
  exit 1
fi

echo "identity check passed"
