#!/usr/bin/env bash
set -euo pipefail

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

cases=(
  "kernel|github.com/ZoneCNH/kernel|kernel"
  "configx|github.com/ZoneCNH/configx|configx"
  "redisx|github.com/ZoneCNH/redisx|redisx"
)

for spec in "${cases[@]}"; do
  IFS='|' read -r module_name module_path package_name <<< "$spec"
  out_dir="$tmpdir/$module_name"

  ./scripts/render_template.sh \
    --module-name "$module_name" \
    --module-path "$module_path" \
    --package-name "$package_name" \
    --out "$out_dir"

  ./scripts/check_rendered_template.sh "$out_dir" "$module_name" "$module_path" "$package_name"

  (
    cd "$out_dir"
    git init -q
    git config user.email "ci@example.invalid"
    git config user.name "Template Integration"
    git add .
    git commit -qm "Initial rendered template"

    env -u GOAL_ID GOWORK=off go mod tidy
    git diff --exit-code -- go.mod go.sum
    env -u GOAL_ID GOWORK=off go test ./...
    env -u GOAL_ID GOWORK=off make contracts
    env -u GOAL_ID GOWORK=off make boundary
    env -u GOAL_ID GOWORK=off make standard-impact-check
    env -u GOAL_ID GOWORK=off make debt
    env -u GOAL_ID GOWORK=off make debt-evidence
    env -u GOAL_ID GOWORK=off make debt-evidence-checksum-check
    env -u GOAL_ID CHECK_STATUS=passed GOWORK=off make evidence
    env -u GOAL_ID RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check
  )
done

echo "integration check passed"
