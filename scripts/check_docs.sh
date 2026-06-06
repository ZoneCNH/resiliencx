#!/usr/bin/env bash
set -euo pipefail

required_files=(
  "README.md"
  "docs/standard/README.md"
  "docs/standard/xlib-standard.md"
  "docs/standard/repository-roles.md"
  "docs/standard/layering.md"
  "docs/standard/module-boundary.md"
  "docs/standard/harness-gates.md"
  "docs/standard/release-standard.md"
  "docs/standard/security-and-secret-policy.md"
  "docs/standard/retrospective-and-patches.md"
  "docs/standard/evidence-protocol.md"
  "docs/standard/conformance-profiles.md"
  "docs/standard/downstream-registry.md"
  "docs/standard/acceptance-matrix.md"
  "docs/standard/goal-runtime.md"
  "docs/standard/agent-team-contract.md"
  "docs/standard/goalcli-cli-contract.md"
  "docs/standard/template-generation-contract.md"
  "docs/standard/docker-toolchain-standard.md"
  "docs/standard/dod.md"
  "docs/standard/downstream-compatibility.md"
  "docs/standard/layer-governance-rules.md"
  "docs/downstream-sync-policy.md"
  "docs/private-business-consumer-guide.md"
  "docs/adr/ADR-20260604-001-layer-governance.md"
  "docs/scorecard.md"
  ".agent/policies/layer-governance.yaml"
  ".agent/runtime/standard/goal-runtime-canonical.md"
  ".agent/docs/standard/goalcli-mapping.md"
  ".agent/archive/standard/audit-2026-06-03.md"
  ".agent/registries/command-registry.yaml"
  ".agent/registries/command-implementation-status.yaml"
  ".agent/registries/makefile-baseline.yaml"
  ".agent/registries/makefile-target-registry.yaml"
  ".agent/harness/harness.yaml"
  ".agent/harness/gates.md"
  ".agent/rules/iron-rules.md"
  ".agent/rules/registry.yaml"
  ".agent/rules/README.md"
  "contracts/layer-governance.schema.json"
  "contracts/goalcli-report.schema.json"
  "internal/goalcli/README.md"
)

for file in "${required_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "ERROR: required documentation file missing: $file" >&2
    exit 1
  fi
done

require_text() {
  local file="$1"
  local needle="$2"

  if ! grep -Fq -- "$needle" "$file"; then
    echo "ERROR: $file must mention: $needle" >&2
    exit 1
  fi
}

require_goalcli_sync_contract() {
  require_text "cmd/goalcli/main.go" "command-registry"
  require_text "cmd/goalcli/main.go" "makefile-baseline"
  require_text "cmd/goalcli/main.go" "cli-contract"
  require_text "cmd/goalcli/main.go" "docs-check"
  require_text "cmd/goalcli/main.go" "release-final-check"
  require_text "cmd/goalcli/main.go" "goal-runtime-final"
  require_text "cmd/goalcli/main.go" "execution-context"

  require_text "cmd/goalcli/main_test.go" "TestUsageDocumentsCommandRegistryRequiredCommands"
  require_text "cmd/goalcli/main_test.go" "TestCommandRegistryRequiredCommandsMatchRegistryFile"
  require_text "cmd/goalcli/main_test.go" "TestCommandRegistryCommandsStayDocumentedInUsage"
  require_text "cmd/goalcli/main_test.go" "TestCommandImplementationStatusCommandsStayRegistered"
  require_text "cmd/goalcli/main_test.go" "commandRegistryRequiredCommands"
  require_text "cmd/goalcli/main_test.go" "implementationStatusCommandsFromText"
  require_text "cmd/goalcli/main_test.go" "usage missing command"
  require_text "cmd/goalcli/main_test.go" "Makefile must define GOALCLI as the cmd/goalcli execution surface"
  require_text "cmd/goalcli/main_test.go" ".agent/registries/command-registry.yaml missing name: execution-context"

  require_text "Makefile" "GOALCLI ?= go run ./cmd/goalcli"
  require_text "Makefile" '$(GOALCLI) command-registry'
  require_text "Makefile" '$(GOALCLI) makefile-baseline'
  require_text "Makefile" '$(GOALCLI) cli-contract'
  require_text "Makefile" '$(GOALCLI) docs-check'
  require_text "Makefile" '$(GOALCLI) release-evidence-checksum-check'
  require_text "Makefile" "GOWORK=off is required for release targets"

  require_text ".agent/registries/command-registry.yaml" 'schema_version: "2.9.3"'
  require_text ".agent/registries/command-registry.yaml" "name: command-registry"
  require_text ".agent/registries/command-registry.yaml" "name: makefile-baseline"
  require_text ".agent/registries/command-registry.yaml" "name: goal-runtime-final"
  require_text ".agent/registries/command-registry.yaml" "name: execution-context"

  require_text ".agent/registries/command-implementation-status.yaml" "source_registry: .agent/registries/command-registry.yaml"
  require_text ".agent/registries/command-implementation-status.yaml" "truth_state: .agent/evidence/truth-state.yaml"
  require_text ".agent/registries/command-implementation-status.yaml" "goalcli_v0_1_0_mva_blocking"
  require_text ".agent/registries/command-implementation-status.yaml" "goal-runtime-final"
  require_text ".agent/registries/command-implementation-status.yaml" "execution-context"

  require_text ".agent/registries/makefile-baseline.yaml" 'schema_version: "2.9.3"'
  require_text ".agent/registries/makefile-baseline.yaml" 'command-registry: "$(GOALCLI) command-registry"'
  require_text ".agent/registries/makefile-baseline.yaml" 'makefile-baseline: "$(GOALCLI) makefile-baseline"'
  require_text ".agent/registries/makefile-baseline.yaml" 'cli-contract: "$(GOALCLI) cli-contract"'
  require_text ".agent/registries/makefile-baseline.yaml" 'goal-runtime-final: "$(GOALCLI) goal-runtime-final'

  require_text ".agent/registries/makefile-target-registry.yaml" "command-registry"
  require_text ".agent/registries/makefile-target-registry.yaml" "makefile-baseline"
  require_text ".agent/registries/makefile-target-registry.yaml" "cli-contract"
  require_text ".agent/registries/makefile-target-registry.yaml" "goal-runtime-final"
  require_text ".agent/registries/makefile-target-registry.yaml" "execution-context"

  require_text ".agent/harness/harness.yaml" 'GOWORK: "off"'
  require_text ".agent/harness/harness.yaml" "cli_contract"
  require_text ".agent/harness/harness.yaml" "command_registry"
  require_text ".agent/harness/harness.yaml" "makefile_baseline"
  require_text ".agent/harness/harness.yaml" "goalcli_v0_1_0"
  require_text ".agent/harness/harness.yaml" "goalcli_mva_gates"

  require_text "docs/standard/goalcli-cli-contract.md" "GoalCLI 同步契约"
  require_text ".agent/docs/standard/goalcli-mapping.md" "GoalCLI 同步契约"
  require_text "internal/goalcli/README.md" "GoalCLI 同步契约"
}

require_goalcli_sync_contract

require_text "README.md" "GOWORK=off make docs-check"
require_text "README.md" "GOWORK=off make dependency-check"
require_text "README.md" "GOWORK=off make standard-impact-check"
require_text "README.md" "GOWORK=off make release-check"
require_text "README.md" "DONE with evidence:"
require_text "README.md" "release/manifest/latest.json"
require_text "README.md" "release/manifest/latest.json.sha256"
require_text "README.md" "release/standard-impact/latest.md"
require_text "README.md" "renovate.json"
require_text "README.md" ".github/dependabot.yml"
require_text "README.md" "downstream_sync_required"
require_text "README.md" "FUZZ_SMOKE_TIME"
require_text "README.md" "docs/downstream-sync-policy.md"
require_text "README.md" "kernel"
require_text "README.md" "Docker Toolchain Runtime"
require_text "README.md" "docs/standard/docker-toolchain-standard.md"
require_text "README.md" "make docker-toolchain-check"
require_text "README.md" "make docker-ci"
require_text "README.md" "make docker-release-check"
require_text "docs/standard/README.md" "docker-toolchain-standard.md"
require_text "docs/standard/README.md" "Docker Toolchain Runtime"
require_text "docs/standard/docker-toolchain-standard.md" "不是第二套 gate"
require_text "docs/standard/docker-toolchain-standard.md" "parent plan #62"
require_text "docs/standard/docker-toolchain-standard.md" ".git"
require_text "docs/standard/docker-toolchain-standard.md" "GOWORK=off"
require_text "docs/standard/docker-toolchain-standard.md" "XLIB_CONTEXT"
require_text "docs/standard/docker-toolchain-standard.md" "VERSION"
require_text "docs/standard/docker-toolchain-standard.md" "DOWNSTREAM"
require_text "docs/standard/docker-toolchain-standard.md" "XLIB_ENABLE_VULNCHECK"
require_text "docs/standard/docker-toolchain-standard.md" "GITHUB_ACTIONS"
require_text "docs/standard/docker-toolchain-standard.md" "python3-yaml"
require_text "docs/standard/docker-toolchain-standard.md" "golangci-lint v2.1.6"
require_text "docs/standard/docker-toolchain-standard.md" "govulncheck v1.1.4"
require_text "docs/standard/docker-toolchain-standard.md" "BuildKit"
require_text "docs/standard/docker-toolchain-standard.md" "XLIB_CONTEXT=release_verify GOWORK=off"
require_text "docs/standard/docker-toolchain-standard.md" "GOWORK=off make integration DOWNSTREAM=kernel"
require_text "docs/standard/docker-toolchain-standard.md" "goalcli doctor"
require_text "docs/standard/docker-toolchain-standard.md" "score"
require_text "docs/testing.md" "GOWORK=off make docker-toolchain-check"
require_text "docs/testing.md" "GOWORK=off make integration DOWNSTREAM=kernel"
require_text "docs/release.md" "XLIB_CONTEXT=release_verify GOWORK=off"
require_text "docs/release.md" "Docker Toolchain Runtime"
require_text "docs/troubleshooting.md" "Docker Toolchain Runtime"
require_text "docs/troubleshooting.md" "docker buildx inspect --bootstrap"
require_text "docs/generation.md" "docker-toolchain-check"
require_text "docs/generation.md" "scripts/docker/docker_gate.sh"
require_text "docs/standard/template-generation-contract.md" "scripts/docker/docker_gate.sh"
require_text "docs/standard/template-generation-contract.md" "Docker 不是第二套 gate"
require_text "docs/downstream-matrix.md" "docker_contract_required"
require_text ".agent/registries/downstream-adoption-status.yaml" "docker_contract_status"
require_text ".agent/registries/downstream-registry.yaml" "docker_contract"
require_text ".github/workflows/docker-contract.yml" "docker-contract"
require_text ".github/workflows/docker-contract.yml" "make docker-toolchain-check"
require_text ".github/workflows/docker-contract.yml" "make docker-ci"
require_text ".github/workflows/docker-contract.yml" "make docker-release-check"
require_text "Makefile" "docker-toolchain-check"
require_text "Makefile" "docker-ci"
require_text "Makefile" "docker-release-check"
require_text "Makefile" 'GITHUB_ACTIONS=$${GITHUB_ACTIONS:-}'
require_text "Makefile" 'GOLANGCI_LINT_VERSION=$${GOLANGCI_LINT_VERSION:-v2.1.6}'
require_text "Makefile" "GIT_CONFIG_VALUE_0=/workspace"
require_text "Dockerfile" "python3-yaml"
require_text "Dockerfile" "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
require_text "Dockerfile" "golang.org/x/vuln/cmd/govulncheck"
require_text "Dockerfile" "safe.directory /workspace"
require_text ".agent/policies/toolchain.yaml" "python3-yaml"
require_text "scripts/docker/docker_gate.sh" 'GITHUB_ACTIONS=${GITHUB_ACTIONS:-}'
require_text "scripts/docker/docker_gate.sh" 'GOLANGCI_LINT_VERSION:-v2.1.6'
require_text "scripts/docker/docker_gate.sh" 'GOVULNCHECK_VERSION:-v1.1.4'
require_text "scripts/docker/docker_gate.sh" "GIT_CONFIG_VALUE_0=/workspace"
require_text "scripts/check_rendered_template.sh" "Dockerfile"
require_text "scripts/check_rendered_template.sh" "docker-release-check"
require_text "docs/standard/README.md" "GOWORK=off make docs-check"
require_text "docs/standard/README.md" "GOWORK=off make dependency-check"
require_text "docs/standard/README.md" "GOWORK=off make standard-impact-check"
require_text "docs/standard/README.md" "GOWORK=off make release-check"
require_text "docs/standard/README.md" "release/manifest/latest.json"
require_text "docs/standard/README.md" "release/manifest/latest.json.sha256"
require_text "docs/standard/README.md" "FUZZ_SMOKE_TIME"
require_text "docs/standard/README.md" "../downstream-sync-policy.md"
require_text "docs/standard/README.md" "layer-governance-rules.md"
require_text "docs/standard/layer-governance-rules.md" "resiliencx"
require_text "docs/standard/layer-governance-rules.md" 'L3 | `x.go`'
require_text "docs/standard/layer-governance-rules.md" "L3 私有"
require_text "docs/standard/layer-governance-rules.md" "natsx"
require_text "docs/standard/layer-governance-rules.md" "GOPRIVATE"
require_text "docs/standard/layer-governance-rules.md" "P0 没有临时例外"
require_text "docs/standard/layer-governance-rules.md" "/home/k8s/secrets/env/*"
require_text "docs/standard/layer-governance-rules.md" "owner"
require_text "docs/standard/layer-governance-rules.md" "回滚方案"
require_text "docs/downstream-matrix.md" '`natsx`'
require_text ".agent/registries/downstream-adoption-status.yaml" "name: natsx"
require_text "docs/standard/downstream-compatibility.md" '`natsx`'
require_text "docs/adr/ADR-20260604-001-layer-governance.md" "L3 私有"
require_text "docs/adr/ADR-20260604-001-layer-governance.md" "docs-check"
require_text ".agent/docs/rule-patches.md" "ADR-20260604-001"
require_text "docs/downstream-sync-policy.md" "private-business-consumer-guide.md"
require_text "docs/private-business-consumer-guide.md" "L3 私有业务系统"
require_text "docs/private-business-consumer-guide.md" "GOPRIVATE"
require_text "docs/private-business-consumer-guide.md" "GONOSUMDB"
require_text "docs/private-business-consumer-guide.md" "go list -m all"
require_text "docs/private-business-consumer-guide.md" "go mod why -m"
require_text "docs/private-business-consumer-guide.md" "go test ./..."
require_text "docs/private-business-consumer-guide.md" "/home/k8s/secrets/env/*"
require_text "docs/private-business-consumer-guide.md" "脱敏"
require_text "docs/private-business-consumer-guide.md" "不得提交"
require_text "docs/private-business-consumer-guide.md" "owner"
require_text ".agent/policies/layer-governance.yaml" "dependency_direction"
require_text ".agent/policies/layer-governance.yaml" "natsx"
require_text ".agent/policies/layer-governance.yaml" "market-data"
require_text ".agent/policies/layer-governance.yaml" "public_release"
require_text "contracts/layer-governance.schema.json" "resiliencx layer governance registry"
require_text "contracts/layer-governance.schema.json" "L3>L2>L1>L0>Standard"
require_text "cmd/goalcli/schema_check.go" ".agent/policies/layer-governance.yaml"
require_text "docs/downstream-sync-policy.md" "resiliencx"
require_text "docs/downstream-sync-policy.md" "kernel"
require_text "docs/downstream-sync-policy.md" "corekit"
require_text "docs/downstream-sync-policy.md" "L1 基础库"
require_text "docs/downstream-sync-policy.md" "x.go 仅作为基础库消费方"
require_text "docs/downstream-sync-policy.md" "downstream_release_decision"
require_text "docs/downstream-sync-policy.md" "release/standard-impact/latest.md"
require_text "docs/downstream-sync-policy.md" "downstream_sync_required"
require_text "docs/supply-chain.md" "kernel"
require_text "docs/supply-chain.md" '旧 `foundationx` 只作为迁移兼容扫描项'
require_text "docs/standard/evidence-protocol.md" "release/manifest/template.json"
require_text "docs/standard/evidence-protocol.md" "release/manifest/latest.json"
require_text "docs/standard/evidence-protocol.md" "artifact_url"
require_text "docs/standard/evidence-protocol.md" "sha256"
require_text "docs/standard/evidence-protocol.md" "workflow_run_id"
require_text "docs/standard/evidence-protocol.md" "standard_impact"
require_text "docs/standard/evidence-protocol.md" "downstream_sync_required"
require_text "docs/standard/evidence-protocol.md" 'downstream_release_decision` 的 allowed values 只能是 `required` 或 `not_required`'
require_text "docs/standard/evidence-protocol.md" 'repository_rules_release_decision` 的 allowed values 只能是 `audit_required` 或 `not_required`'
require_text "docs/standard/evidence-protocol.md" "generator_evidence"
require_text "docs/standard/evidence-protocol.md" "dependency_check"
require_text "docs/standard/evidence-protocol.md" "GOWORK=off make dependency-check"
require_text "docs/standard/evidence-protocol.md" "GOWORK=off make standard-impact-check"
require_text "README.md" 'downstream_release_decision`（只允许 `required` / `not_required`）'
require_text "README.md" 'repository_rules_release_decision`（只允许 `audit_required` / `not_required`）'
require_text "docs/release.md" 'standard_impact.downstream_release_decision` 只能使用 `required` 或 `not_required`'
require_text "docs/release.md" 'standard_impact.repository_rules_release_decision` 只能使用 `audit_required` 或 `not_required`'
require_text "docs/standard/evidence-protocol.md" 'standard_impact.downstream_release_decision` 的 allowed values 只能是 `required` 或 `not_required`'
require_text "docs/standard/evidence-protocol.md" 'standard_impact.repository_rules_release_decision` 的 allowed values 只能是 `audit_required` 或 `not_required`'
require_text "docs/downstream-sync-policy.md" 'downstream_release_decision` 的 allowed values 只能是 `required` 或 `not_required`'
require_text "docs/downstream-sync-policy.md" 'repository_rules_release_decision` 的 allowed values 只能是 `audit_required` 或 `not_required`'
require_text "docs/standard/harness-gates.md" 'downstream_release_decision`（`required` / `not_required`）'
require_text "docs/standard/harness-gates.md" 'repository_rules_release_decision`（`audit_required` / `not_required`）'
require_text "docs/standard/release-standard.md" "release/manifest/latest.json.sha256"
require_text "release/manifest/template.json" "release/manifest/latest.json.sha256"
require_text "release/manifest/template.json" '"dependencies"'
require_text "release/manifest/template.json" '"standard_impact"'
require_text "release/manifest/template.json" '"downstream_sync_required"'
require_text "release/manifest/template.json" '"generator_evidence"'
require_text "release/manifest/template.json" '"dependency_check"'
require_text "docs/scorecard.md" "go run ./cmd/goalcli score --min 9.8"
require_text "docs/scorecard.md" "RELEASE_EVIDENCE_MIN_SCORE=9.8"
require_text "release/manifest/template.json" '"score"'
require_text "release/manifest/template.json" '"workflow_run_id"'
require_text "internal/tools/releasemanifest/main.go" "min-score"
require_text "Makefile" "go run ./cmd/goalcli score --min 9.8"
require_text "Makefile" "RELEASE_EVIDENCE_MIN_SCORE=9.8"
require_text ".agent/release/release-template.md" "go run ./cmd/goalcli score --min 9.8"
require_text ".agent/docs/retrospective-template.md" "Score"
require_text ".agent/harness/harness.yaml" "go run ./cmd/goalcli score --min 9.8"
require_text "internal/tools/releasemanifest/main.go" "release/manifest/latest.json.sha256"
require_text "cmd/goalcli/main.go" "docs-check"
require_text "cmd/goalcli/main.go" "dependency-check"
require_text "cmd/goalcli/main.go" "standard-impact-check"
require_text "cmd/goalcli/main.go" "boundary"
require_text "cmd/goalcli/main.go" "contracts"
require_text "cmd/goalcli/main.go" "render-check"
require_text "cmd/goalcli/main.go" "release-final-check"
require_text "cmd/goalcli/main.go" "score"
require_text "cmd/goalcli/main.go" "main-guard"
require_text "cmd/goalcli/main.go" "worktree-guard"
require_text "cmd/goalcli/main.go" "issue-registry"
require_text "cmd/goalcli/main.go" "command-registry"
require_text "docs/standard/goalcli-cli-contract.md" "goalcli"
require_text "docs/standard/goalcli-cli-contract.md" "contracts/goalcli-report.schema.json"
require_text "cmd/goalcli/main.go" "--min"
require_text "Makefile" "GOWORK=off is required for release targets"
require_text "Makefile" "GOALCLI ?= go run ./cmd/goalcli"
require_text "Makefile" '$(GOALCLI) docs-check'
require_text "Makefile" '$(GOALCLI) dependency-check'
require_text "Makefile" '$(GOALCLI) standard-impact-check'
require_text "Makefile" '$(GOALCLI) boundary'
require_text "Makefile" '$(GOALCLI) contracts'
require_text "Makefile" '$(GOALCLI) integration'
require_text "Makefile" '$(GOALCLI) score --min 9.8'
require_text "Makefile" '$(GOALCLI) release-evidence-checksum-check'
require_text "scripts/run_fuzz_smoke.sh" 'fuzz_time="${FUZZ_SMOKE_TIME:-10s}"'
require_text ".github/workflows/ci.yml" "make release-check"
require_text ".github/workflows/ci.yml" "release/manifest/latest.json.sha256"
require_text ".github/workflows/ci.yml" 'XLIB_ENABLE_VULNCHECK: "0"'
require_text ".github/workflows/ci.yml" "env.XLIB_ENABLE_VULNCHECK == '1'"
require_text ".github/workflows/release.yml" "make release-final-check"
require_text ".github/workflows/release.yml" "release/manifest/latest.json.sha256"
require_text ".github/workflows/release.yml" "ARTIFACT_URL"
require_text ".github/workflows/release.yml" 'XLIB_ENABLE_VULNCHECK: "0"'
require_text ".github/workflows/release.yml" "env.XLIB_ENABLE_VULNCHECK == '1'"
require_text ".github/workflows/release.yml" "contents: write"
require_text ".github/workflows/release.yml" "gh release create"
require_text ".github/workflows/release.yml" "gh release edit"
require_text ".github/workflows/release.yml" "gh release view"
require_text ".github/workflows/release.yml" "--verify-tag"
require_text ".github/workflows/security.yml" "schedule:"
require_text ".github/workflows/security.yml" 'cron: "17 3 * * 1"'
require_text ".github/workflows/security.yml" "github.event_name == 'schedule'"
require_text ".github/workflows/security.yml" 'XLIB_FORCE_VULNCHECK: ${{ github.event_name =='
require_text ".github/workflows/security.yml" 'XLIB_VULNCHECK_INTERVAL_HOURS: "168"'
require_text ".github/workflows/security.yml" "env.XLIB_ENABLE_VULNCHECK == '1'"
require_text ".github/workflows/release-auto-patch.yml" "branches: [main]"
require_text ".github/workflows/release-auto-patch.yml" "contents: write"
require_text ".github/workflows/release-auto-patch.yml" "fetch-depth: 0"
require_text ".github/workflows/release-auto-patch.yml" "release-auto-patch-main"
require_text ".github/workflows/release-auto-patch.yml" 'XLIB_ENABLE_VULNCHECK: "0"'
require_text ".github/workflows/release-auto-patch.yml" "env.XLIB_ENABLE_VULNCHECK == '1'"
require_text ".github/workflows/release-auto-patch.yml" "git tag --points-at"
require_text ".github/workflows/release-auto-patch.yml" "already_released=true"
require_text ".github/workflows/release-auto-patch.yml" "git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname"
require_text ".github/workflows/release-auto-patch.yml" 'next_patch=$((patch + 1))'
require_text ".github/workflows/release-auto-patch.yml" "GOWORK=off make release-final-check"
require_text ".github/workflows/release-auto-patch.yml" "git tag -a"
require_text ".github/workflows/release-auto-patch.yml" 'git push origin "refs/tags/${RELEASE_TAG}"'
require_text ".github/workflows/release-auto-patch.yml" "gh release create"
require_text ".github/workflows/release-auto-patch.yml" "gh release edit"
require_text ".github/workflows/release-auto-patch.yml" "gh release view"
require_text ".github/workflows/release-auto-patch.yml" "--verify-tag"
require_text ".github/workflows/release-auto-patch.yml" ".url | length > 0"
require_text ".github/workflows/release-auto-patch.yml" "govulncheck@v1.1.4"
require_text "docs/release.md" ".github/workflows/release-auto-patch.yml"
require_text "docs/release.md" "vX.Y.(Z+1)"
require_text "docs/release.md" "GOWORK=off make release-final-check"
require_text "docs/release.md" "already_released=true"
require_text "docs/release.md" "release-auto-patch-main"
require_text "docs/standard/release-standard.md" ".github/workflows/release-auto-patch.yml"
require_text "docs/standard/release-standard.md" "vX.Y.(Z+1)"
require_text "docs/standard/release-standard.md" "already_released=true"
require_text ".github/workflows/ci.yml" "ARTIFACT_URL"


# Goal v2.9.3 executable governance contract checks.
require_text "cmd/goalcli/main.go" "main-guard"
require_text "cmd/goalcli/main.go" "policy-schema"
require_text "cmd/goalcli/main.go" "downstream-adoption"
require_text "cmd/goalcli/main.go" "runtime-file-ownership"
require_text "cmd/goalcli/main.go" "adoption-check"
require_text "cmd/goalcli/adoption_check.go" "xlib-standard.lock"
require_text "scripts/render_template.sh" "--enable-governance"
require_text "Makefile" "adoption-check"
require_text ".github/workflows/adoption-check.yml" "GOWORK=off make adoption-check"
require_text ".github/rulesets/protect-main.json" '"bypass_actors": []'
require_text ".github/rulesets/protect-main.json" "adoption-check"
require_text ".github/rulesets/protect-main.json" "governance-check"
require_text ".github/rulesets/protect-main.json" "release-check"
require_text "mk/governance.mk" "adoption-check"
require_text ".agent/registries/command-registry.yaml" "adoption-check"
require_text ".agent/registries/makefile-baseline.yaml" "adoption-check"
require_text ".agent/registries/makefile-target-registry.yaml" "adoption-check"
require_text ".agent/harness/harness.yaml" "adoption_check"
require_text ".agent/harness/gates.md" "adoption-check"
require_text "docs/standard/harness-gates.md" "adoption-check"
require_text "docs/standard/goalcli-cli-contract.md" "adoption-check"
require_text "docs/generation.md" "--enable-governance"
require_text "docs/generation.md" "xlib-standard.lock"
require_text "Makefile" "governance-check"
require_text "Makefile" "p1-governance-check"
require_text "Makefile" "p2-runtime-check"
require_text "Makefile" "execution-context"
require_text ".github/workflows/ci.yml" "GOWORK=off XLIB_CONTEXT=ci_pull_request make release-check"
require_text ".agent/registries/command-registry.yaml" "downstream-adoption"
require_text ".agent/registries/command-registry.yaml" "runtime-file-ownership"
require_text ".agent/registries/command-registry.yaml" "execution-context"
require_text ".agent/registries/issue-registry.yaml" "GOAL-V293-P0"
require_text ".agent/registries/makefile-baseline.yaml" "score-check"
require_text ".agent/registries/makefile-baseline.yaml" "execution-context"
require_text ".agent/registries/makefile-target-registry.yaml" "execution-context"
require_text ".agent/harness/harness.yaml" "execution-context"
require_text ".agent/harness/gates.md" "execution-context"
require_text "docs/standard/goalcli-cli-contract.md" "不读取真实 secrets"
require_text "docs/standard/goalcli-cli-contract.md" "downstream-adoption"
require_text "docs/standard/goalcli-cli-contract.md" "execution-context"
require_text "docs/standard/harness-gates.md" "execution-context"
require_text "docs/standard/agent-team-contract.md" "leader"
require_text "docs/standard/goal-runtime.md" "runtime-health"
require_text "docs/standard/acceptance-matrix.md" "governance-check"
require_text "docs/standard/downstream-registry.md" "kernel/configx"
require_text "docs/standard/conformance-profiles.md" "l0-kernel"

xlib_standard_url="https://github.com/ZoneCNH/resiliencx"
require_text "README.md" "$xlib_standard_url"
require_text "docs/standard/README.md" "$xlib_standard_url"
require_text "docs/spec.md" "$xlib_standard_url"
require_text "docs/design.md" "$xlib_standard_url"
require_text "docs/generation.md" "$xlib_standard_url"
require_text "docs/standard/xlib-standard.md" "$xlib_standard_url"
require_text "docs/standard/repository-roles.md" "$xlib_standard_url"
require_text "docs/standard/harness-gates.md" "GOWORK=off make dependency-check"
require_text "docs/standard/harness-gates.md" "GOWORK=off make standard-impact-check"
require_text "docs/standard/harness-gates.md" "Context Runtime v4.0 Profile Baseline"
require_text "docs/standard/harness-gates.md" "REQ-014"
require_text "docs/standard/harness-gates.md" ".agent/context/*"
require_text "docs/standard/harness-gates.md" "release-final-check"
require_text "docs/standard/evidence-protocol.md" "governance_runtime"
require_text "docs/standard/evidence-protocol.md" "REQ-014"
require_text "docs/scorecard.md" "context_runtime"
require_text "docs/downstream-sync-policy.md" "templates/context-consumer/*"
require_text ".gitignore" ".agent/context/packs/*.md"
require_text ".gitignore" "!.agent/context/packs/example.md"
require_text ".gitignore" ".agent/context/**/schema-snapshots/*.json"
require_text ".gitignore" "!.agent/context/**/schema-snapshots/example.json"
require_text ".gitignore" "*.schema.snapshot.json"
require_text ".gitignore" "github-rules-observed.json"
require_text ".gitignore" ".agent/github/rules/observed/"
require_text ".gitignore" ".github/observed-rules/"
require_text ".gitignore" ".github/rules/observed/"
require_text ".gitignore" ".terraform/"
require_text ".gitignore" "*.tfstate"
require_text ".gitignore" "*.tfplan"
require_text ".gitignore" "*.tfvars"
require_text ".gitignore" "!*.tfvars.example"
require_text ".gitignore" "!examples/context-packs/**"
require_text ".gitignore" "!examples/schema-snapshots/**"
require_text "renovate.json" '"gomod"'
require_text "renovate.json" '"github-actions"'
require_text ".github/dependabot.yml" 'package-ecosystem: "gomod"'
require_text ".github/dependabot.yml" 'package-ecosystem: "github-actions"'


check_ignored() {
  local path="$1"
  if ! git check-ignore -q -- "$path"; then
    echo "ERROR: .gitignore must ignore: $path" >&2
    exit 1
  fi
}

check_not_ignored() {
  local path="$1"
  if git check-ignore -q -- "$path"; then
    echo "ERROR: .gitignore must keep example path unignored: $path" >&2
    exit 1
  fi
}

check_ignored ".agent/context/packs/generated.md"
check_ignored ".agent/context/schema-snapshots/runtime.schema.snapshot.json"
check_not_ignored ".agent/context/packs/example.md"
check_not_ignored ".agent/context/packs/runtime.example.md"
check_not_ignored ".agent/context/schema-snapshots/example.json"
check_not_ignored ".agent/context/runtime/schema-snapshots/example.json"
check_ignored "github-rules-observed.json"
check_ignored ".github/observed-rules/rules.json"
check_ignored ".github/rules/observed/rules.json"
check_ignored ".terraform/terraform.tfstate"
check_ignored "terraform.tfstate"
check_ignored "release.tfplan"
check_ignored "local.auto.tfvars"
check_not_ignored "terraform.tfvars.example"
check_not_ignored "examples/context-packs/README.md"
check_not_ignored "examples/schema-snapshots/runtime.schema.json"

python3 - "$PWD" <<'PY'
import sys
from pathlib import Path

root = Path(sys.argv[1])
requirements = {
    "docs/standard/xlib-standard.md": [
        "resiliencx",
        "resiliencx",
        "模板",
        "generator",
        "Harness",
        "Evidence",
    ],
    "docs/standard/repository-roles.md": [
        "resiliencx",
        "resiliencx",
        "标准权威源",
        "模板、generator、Harness、Evidence 实现仓库",
    ],
    "docs/standard/layering.md": [
        "resiliencx",
        "resiliencx",
        "Standard 规则的独立来源",
        "Go 基础库模板中的实现仓库",
    ],
    "docs/standard/layer-governance-rules.md": [
        "resiliencx",
        "kernel",
        "natsx",
        "L3 私有",
        "GOPRIVATE",
        "P0 没有临时例外",
    ],
    "docs/standard/module-boundary.md": [
        "resiliencx",
        "resiliencx",
        "go.mod",
        "module path",
    ],
    "docs/downstream-sync-policy.md": [
        "Standard Source",
        "Go Reference Template",
        "Generator",
        "L0 代表下游",
        "L1 基础库",
        "x.go 仅作为基础库消费方",
        "kernel",
        "corekit",
        "downstream_release_decision",
    ],
}

errors = []
for rel, needles in requirements.items():
    text = (root / rel).read_text(encoding="utf-8")
    for needle in needles:
        if needle not in text:
            errors.append(f"{rel} must mention: {needle}")

if errors:
    for error in errors:
        print(f"ERROR: {error}", file=sys.stderr)
    sys.exit(1)
PY



python3 - "$PWD" <<'PY'
import re
import sys
from pathlib import Path

root = Path(sys.argv[1])
scan_files = [
    root / "README.md",
    root / "docs/supply-chain.md",
    *sorted((root / "docs/standard").glob("*.md")),
]

bad_current_name_patterns = [
    (re.compile(r"渲染\s*`?foundationx`?"), "current downstream render target must be kernel/corekit"),
    (re.compile(r"生成\s*`?foundationx`?"), "current generated downstream target must be kernel/corekit"),
    (re.compile(r"默认下游[^。\n]*foundationx"), "default downstream must be kernel"),
    (re.compile(r"foundationx[^。\n]*默认下游"), "default downstream must be kernel"),
    (re.compile(r"foundationx[^。\n]*代表下游"), "representative downstream must be kernel/corekit"),
]

errors = []
for path in scan_files:
    text = path.read_text(encoding="utf-8")
    rel = path.relative_to(root)
    for pattern, message in bad_current_name_patterns:
        for match in pattern.finditer(text):
            sentence_start = max(text.rfind("。", 0, match.start()), text.rfind("\n", 0, match.start())) + 1
            sentence_end_candidates = [idx for idx in (text.find("。", match.end()), text.find("\n", match.end())) if idx != -1]
            sentence_end = min(sentence_end_candidates) if sentence_end_candidates else len(text)
            sentence = text[sentence_start:sentence_end]
            if any(word in sentence for word in ("旧", "迁移", "兼容", "历史")):
                continue
            errors.append(f"{rel}: {message}: {match.group(0)}")

if errors:
    for error in errors:
        print(f"ERROR: {error}", file=sys.stderr)
    sys.exit(1)
PY

if ! git check-ignore -q release/manifest/latest.json; then
  echo "ERROR: release/manifest/latest.json must remain ignored because it is generated Evidence" >&2
  exit 1
fi

if ! git check-ignore -q release/manifest/latest.json.sha256; then
  echo "ERROR: release/manifest/latest.json.sha256 must remain ignored because it is generated Evidence" >&2
  exit 1
fi

python3 - "$PWD/Makefile" <<'PY'
import re
import sys
from pathlib import Path

makefile = Path(sys.argv[1]).read_text(encoding="utf-8")
errors = []

for target in ("release-check", "release-check-extended"):
    match = re.search(rf"^{re.escape(target)}:([^\n]*)", makefile, re.MULTILINE)
    if not match:
        errors.append(f"Makefile target missing: {target}")
        continue
    deps = match.group(1).split()
    body_match = re.search(
        rf"^{re.escape(target)}:[^\n]*\n((?:\t.*\n)+)",
        makefile,
        re.MULTILINE,
    )
    body = body_match.group(1) if body_match else ""
    for dep in ("dependency-check", "standard-impact-check", "docs-check", "require-gowork-off"):
        if dep not in deps:
            errors.append(f"Makefile {target} must depend on {dep}")
    if "release-evidence-hash" not in body:
        errors.append(f"Makefile {target} must generate release Evidence checksum")
    if "release-evidence-checksum-check" not in body:
        errors.append(f"Makefile {target} must verify release Evidence checksum")

if errors:
    for error in errors:
        print(f"ERROR: {error}", file=sys.stderr)
    sys.exit(1)
PY

scan_files=("README.md")
while IFS= read -r -d '' file; do
  scan_files+=("$file")
done < <(find docs/standard -maxdepth 1 -type f -name '*.md' -print0)

if command -v rg >/dev/null 2>&1; then
  if rg -n '\{\{[^}]+\}\}|TODO_TEMPLATE' "${scan_files[@]}"; then
    echo "ERROR: unresolved template placeholder found in README.md or docs/standard/*.md" >&2
    exit 1
  fi
else
  if grep -nE '\{\{[^}]+\}\}|TODO_TEMPLATE' "${scan_files[@]}"; then
    echo "ERROR: unresolved template placeholder found in README.md or docs/standard/*.md" >&2
    exit 1
  fi
fi

mapfile -t markdown_files < <(find docs -type f -name '*.md' | sort)
python3 - "$PWD" "README.md" "${markdown_files[@]}" <<'PY'
import os
import re
import sys
from pathlib import Path
from urllib.parse import unquote, urlparse

root = Path(sys.argv[1])
files = [Path(p) for p in sys.argv[2:]]
link_re = re.compile(r'!?\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)')
errors = []

for rel in files:
    path = root / rel
    text = path.read_text(encoding="utf-8")
    for match in link_re.finditer(text):
        raw_target = match.group(1).strip()
        if not raw_target or raw_target.startswith("#"):
            continue
        target = raw_target.strip("<>")
        parsed = urlparse(target)
        if parsed.scheme or target.startswith("//") or target.startswith("mailto:"):
            continue
        target_path = unquote(target.split("#", 1)[0])
        if not target_path:
            continue
        resolved = (path.parent / target_path).resolve()
        try:
            resolved.relative_to(root)
        except ValueError:
            errors.append(f"{rel}: local link escapes repository: {raw_target}")
            continue
        if not resolved.exists():
            errors.append(f"{rel}: broken local link: {raw_target}")

if errors:
    for error in errors:
        print(error, file=sys.stderr)
    sys.exit(1)
PY

echo "docs-check passed"
