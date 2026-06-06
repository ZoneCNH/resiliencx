GOALCLI ?= go run ./cmd/goalcli
XLIB_CONTEXT ?= local_write
GOAL_ID ?= GOAL-20260603-XLIB-GOALCLI-001
GOAL_RUNTIME_MODE ?= FULL
DOCKER_IMAGE ?= $(notdir $(CURDIR))-toolchain:local
DOCKER_GATE ?= GITHUB_ACTIONS=$${GITHUB_ACTIONS:-} GOLANGCI_LINT_VERSION=$${GOLANGCI_LINT_VERSION:-v2.1.6} GOVULNCHECK_VERSION=$${GOVULNCHECK_VERSION:-v1.1.4} GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=safe.directory GIT_CONFIG_VALUE_0=/workspace ./scripts/docker/docker_gate.sh

.PHONY: require-gowork-off
require-gowork-off:
	@if [ "$${GOWORK:-}" != "off" ]; then \
		echo "GOWORK=off is required for release targets"; \
		exit 1; \
	fi

.PHONY: build
build:
	go build ./...

.PHONY: build-check
build-check: build

.PHONY: goalcli
goalcli:
	go build ./cmd/goalcli

.PHONY: goalcli-image
goalcli-image: goalcli

.PHONY: shell
shell:
	bash

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: race
race:
	go test -race ./...

.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed"; \
		exit 1; \
	fi

.PHONY: integration
integration:
	$(GOALCLI) integration

.PHONY: docker-toolchain-check
docker-toolchain-check:
	./scripts/docker/check_toolchain.sh

.PHONY: docker-build
docker-build:
	$(DOCKER_GATE) build


.PHONY: docker-build-check
docker-build-check:
	$(DOCKER_GATE) build-check

.PHONY: docker-shell
docker-shell:
	$(DOCKER_GATE) shell


.PHONY: docker-ci
docker-ci:
	$(DOCKER_GATE) ci

.PHONY: docker-release-check
docker-release-check:
	$(DOCKER_GATE) release-check

.PHONY: docker-release-final-check
docker-release-final-check:
	$(DOCKER_GATE) release-final-check

.PHONY: docker-goalcli
docker-goalcli:
	$(DOCKER_GATE) goalcli

.PHONY: docker-goalcli-image
docker-goalcli-image:
	$(DOCKER_GATE) goalcli-image


.PHONY: docker-goalcli-version
docker-goalcli-version:
	$(DOCKER_GATE) goalcli-version

.PHONY: docker-runtime-check
docker-runtime-check:
	$(DOCKER_GATE) runtime-check

.PHONY: docker-drift-check
docker-drift-check:
	./scripts/docker/check_toolchain.sh --drift

.PHONY: docker-contract
docker-contract:
	$(DOCKER_GATE) contract


.PHONY: dependency-check
dependency-check:
	$(GOALCLI) dependency-check

.PHONY: standard-impact-check
standard-impact-check:
	$(GOALCLI) standard-impact-check

.PHONY: docs-check
docs-check:
	$(GOALCLI) docs-check

.PHONY: adoption-check
adoption-check: require-gowork-off
	$(GOALCLI) adoption-check --verify

.PHONY: rules-verify
rules-verify:
	$(GOALCLI) rules-verify

.PHONY: debt architecture domain docs-drift dependency-debt security-debt testing-debt implementation-debt downstream-debt

.PHONY: debt-evidence
debt-evidence:
	$(GOALCLI) debt-evidence

.PHONY: debt-evidence-hash
debt-evidence-hash:
	$(GOALCLI) debt-evidence-hash

.PHONY: debt-evidence-checksum-check
debt-evidence-checksum-check:
	$(GOALCLI) debt-evidence-checksum-check

.PHONY: secret-check
.PHONY: security

architecture:
	$(GOALCLI) debt --section architecture --mode enforce

domain:
	$(GOALCLI) debt --section domain --mode enforce

docs-drift:
	$(GOALCLI) debt --section docs --mode warn

dependency-debt:
	$(GOALCLI) debt --section dependency --mode warn

testing-debt:
	$(GOALCLI) debt --section testing --mode warn

implementation-debt:
	$(GOALCLI) debt --section implementation --mode observe

security-debt:
	$(GOALCLI) debt --section security --mode warn

downstream-debt:
	$(GOALCLI) downstream-debt

.PHONY: downstream-sync-plan
downstream-sync-plan: standard-impact-check
	$(GOALCLI) downstream-sync-plan

debt:
	$(GOALCLI) debt --config .agent/policies/debt/rules.yaml --exceptions .agent/policies/debt/exceptions.yaml --dependency-purpose .agent/policies/debt/dependency-purpose.yaml --mode enforce --min-score 9.8

.PHONY: debt-register-update debt-trend debt-patch-suggest debt-lifecycle-check
debt-register-update:
	$(GOALCLI) debt register-update

debt-trend:
	$(GOALCLI) debt trend

debt-patch-suggest:
	$(GOALCLI) debt patch-suggest

debt-lifecycle-check:
	$(GOALCLI) debt lifecycle-check

secret-check:
	$(GOALCLI) secret-check

security:
	$(GOALCLI) security

.PHONY: boundary
boundary:
	$(GOALCLI) boundary

.PHONY: contracts
contracts:
	$(GOALCLI) contracts

.PHONY: property
property:
	go test ./... -run 'Test.*Property|Test.*Invariant'

.PHONY: fuzz-smoke
fuzz-smoke:
	./scripts/run_fuzz_smoke.sh

.PHONY: golden
golden:
	go test ./... -run 'Test.*Golden|Test.*Snapshot'

.PHONY: evidence
evidence:
	$(GOALCLI) evidence

.PHONY: goal-score-check
goal-score-check:
	go run ./cmd/goalcli score --min 9.8

.PHONY: release-evidence-hash
release-evidence-hash:
	$(GOALCLI) release-evidence-hash

.PHONY: release-evidence-check
release-evidence-check:
	$(GOALCLI) release-evidence-check

.PHONY: release-evidence-checksum-check
release-evidence-checksum-check:
	$(GOALCLI) release-evidence-checksum-check

.PHONY: score
score: score-check

.PHONY: score-check
score-check:
	# Release evidence verifier default: RELEASE_EVIDENCE_MIN_SCORE=9.8
	# Direct equivalent for docs/CI drift checks: go run ./cmd/goalcli score --min 9.8
	$(GOALCLI) score --min 9.8

.PHONY: version
version:
	$(GOALCLI) version

.PHONY: goalcli-version
goalcli-version: version

.PHONY: doctor
doctor:
	$(GOALCLI) doctor

.PHONY: runtime-check
runtime-check: doctor

.PHONY: drift-check
drift-check:
	$(GOALCLI) docs-check
	$(GOALCLI) command-registry
	$(GOALCLI) makefile-baseline

.PHONY: contract
contract: runtime-check drift-check

.PHONY: main-guard
main-guard:
	$(GOALCLI) main-guard --context $(XLIB_CONTEXT)

.PHONY: worktree-guard
worktree-guard:
	$(GOALCLI) worktree-guard --context $(XLIB_CONTEXT)

.PHONY: evidence-check
evidence-check:
	$(GOALCLI) evidence-check

.PHONY: cli-contract
cli-contract:
	$(GOALCLI) cli-contract

.PHONY: issue-registry
issue-registry:
	$(GOALCLI) issue-registry

.PHONY: command-registry
command-registry:
	$(GOALCLI) command-registry

.PHONY: makefile-baseline
makefile-baseline:
	$(GOALCLI) makefile-baseline

.PHONY: audit-goal
audit-goal:
	$(GOALCLI) audit-goal

.PHONY: dashboard-generate
dashboard-generate:
	$(GOALCLI) dashboard-generate

.PHONY: agent-team-contract scope-lock pr-template acceptance-matrix runtime-health upgrade-standard conformance-profile downstream-registry self-healing-skeleton goal-runtime github-governance supply-chain changelog governance-fixture-test autoresearch policy-schema github-settings toolchain evidence-artifacts naming
agent-team-contract scope-lock pr-template acceptance-matrix runtime-health upgrade-standard conformance-profile downstream-registry self-healing-skeleton goal-runtime github-governance supply-chain changelog governance-fixture-test autoresearch policy-schema github-settings toolchain evidence-artifacts naming:
	$(GOALCLI) $@ --dry-run --verify


.PHONY: install-runtime upgrade-runtime release-ready evidence-replay attest-conformance pack-standard pack-gate pack-evidence downstream-baseline downstream-adoption runtime-file-ownership
install-runtime upgrade-runtime release-ready evidence-replay attest-conformance pack-standard pack-gate pack-evidence downstream-baseline downstream-adoption runtime-file-ownership:
	$(GOALCLI) $@ --dry-run --verify

.PHONY: execution-context
execution-context:
	$(GOALCLI) $@ --dry-run --verify

.PHONY: goal-acceptance
goal-acceptance: require-gowork-off
	$(GOALCLI) $@ --goal-id "$(GOAL_ID)" --mode "$(GOAL_RUNTIME_MODE)" --json --write-evidence

.PHONY: goal-delivery
goal-delivery: require-gowork-off
	$(GOALCLI) $@ --goal-id "$(GOAL_ID)" --mode "$(GOAL_RUNTIME_MODE)" --json --write-evidence

.PHONY: goal-handover
goal-handover: require-gowork-off
	$(GOALCLI) $@ --goal-id "$(GOAL_ID)" --mode "$(GOAL_RUNTIME_MODE)" --json --write-evidence

.PHONY: goal-downstream-adoption
goal-downstream-adoption: require-gowork-off
	$(GOALCLI) $@ --goal-id "$(GOAL_ID)" --mode "$(GOAL_RUNTIME_MODE)" --json --write-evidence

.PHONY: goal-certify
goal-certify: require-gowork-off
	$(GOALCLI) $@ --goal-id "$(GOAL_ID)" --mode "$(GOAL_RUNTIME_MODE)" --json --write-evidence

.PHONY: goal-runtime-final
goal-runtime-final: require-gowork-off goal-acceptance goal-delivery goal-handover goal-downstream-adoption goal-certify
	$(GOALCLI) $@ --goal-id "$(GOAL_ID)" --mode "$(GOAL_RUNTIME_MODE)" --json --write-evidence

.PHONY: traceability-check
traceability-check:
	$(GOALCLI) traceability-check

.PHONY: governance-check
governance-check: require-gowork-off main-guard worktree-guard evidence-check adoption-check boundary architecture domain security security-debt contracts docs-check cli-contract issue-registry command-registry makefile-baseline audit-goal rules-consistency-check debt traceability-check

.PHONY: rules-consistency-check
rules-consistency-check:
	$(GOALCLI) rules-consistency-check

.PHONY: p1-governance-check
p1-governance-check: agent-team-contract scope-lock pr-template acceptance-matrix runtime-health upgrade-standard conformance-profile downstream-registry self-healing-skeleton goal-runtime github-governance supply-chain changelog governance-fixture-test autoresearch policy-schema github-settings toolchain evidence-artifacts naming

.PHONY: p2-runtime-check
p2-runtime-check: install-runtime upgrade-runtime release-ready evidence-replay attest-conformance pack-standard pack-gate pack-evidence downstream-baseline downstream-adoption runtime-file-ownership execution-context

.PHONY: context-profile
context-profile:
	$(GOALCLI) context-profile --profile $${PROFILE:-standard}

.PHONY: context-profile-check
context-profile-check:
	$(GOALCLI) context-profile-check

.PHONY: context-schema-check
context-schema-check:
	$(GOALCLI) context-schema-check

.PHONY: schema-check
schema-check:
	$(GOALCLI) schema validate --all --report reports/schema-check.json

.PHONY: context-lite
context-lite: require-gowork-off governance-check

.PHONY: context-standard
context-standard: require-gowork-off governance-check p1-governance-check docs-check

.PHONY: context-full
context-full: require-gowork-off governance-check p1-governance-check p2-runtime-check

.PHONY: context-release
context-release: require-gowork-off context-full integration dependency-check standard-impact-check score-check debt-evidence
	CHECK_STATUS=passed $(MAKE) evidence
	$(MAKE) release-evidence-hash
	$(MAKE) release-evidence-check
	$(MAKE) release-evidence-checksum-check

.PHONY: context-fast-check
context-fast-check: context-lite

.PHONY: context-standard-check
context-standard-check: context-standard

.PHONY: context-full-check
context-full-check: context-full

.PHONY: ci
ci: doctor-hooks-local fmt vet lint test race boundary architecture domain secret-check security security-debt contracts governance-check debt score rules-verify

.PHONY: ci-extended
ci-extended: ci property golden fuzz-smoke docs-drift

.PHONY: release-check
release-check: require-gowork-off ci integration dependency-check standard-impact-check docs-check docs-drift score-check governance-check p1-governance-check p2-runtime-check debt-evidence
	CHECK_STATUS=passed $(MAKE) evidence
	$(MAKE) release-evidence-hash
	$(MAKE) release-evidence-check
	$(MAKE) release-evidence-checksum-check

.PHONY: release-check-extended
release-check-extended: require-gowork-off ci-extended integration dependency-check standard-impact-check docs-check score-check governance-check p1-governance-check p2-runtime-check debt-evidence
	CHECK_STATUS=passed $(MAKE) evidence
	$(MAKE) release-evidence-hash
	$(MAKE) release-evidence-check
	$(MAKE) release-evidence-checksum-check

.PHONY: release-final-check
release-final-check:
	XLIB_CONTEXT=release_verify GOWORK=off $(MAKE) context-release
	$(MAKE) debt-evidence-checksum-check
	$(GOALCLI) score --min 9.8
	RELEASE_EVIDENCE_REQUIRE_PASSED=1 RELEASE_EVIDENCE_REQUIRE_CLEAN=1 RELEASE_EVIDENCE_MIN_SCORE=9.8 ./scripts/check_release_evidence.sh
	$(MAKE) release-evidence-checksum-check

.PHONY: release-preflight
release-preflight:
	./scripts/check_release_preflight.sh "$(VERSION)"
	GOWORK=off XLIB_CONTEXT=release_verify VERSION="$(VERSION)" $(MAKE) release-final-check

# ── Goal Gate Targets ──────────────────────────────────────────
# 以下 target 对应 .agent/harness/gates/*.yaml 中 commands 引用

.PHONY: worktree-check
worktree-check:
	$(GOALCLI) worktree-check --context $(XLIB_CONTEXT)

.PHONY: context-check
context-check:
	$(GOALCLI) context-check

.PHONY: spec-check
spec-check:
	$(GOALCLI) spec-check

.PHONY: design-check
design-check:
	$(GOALCLI) design-check

.PHONY: task-check
task-check:
	$(GOALCLI) task-check

.PHONY: pr-check
pr-check:
	$(GOALCLI) pr-check --context $(XLIB_CONTEXT)

.PHONY: retro-check self-improving-check
retro-check: self-improving-check

self-improving-check:
	$(GOALCLI) self-improving-check

install-hooks:
	@git config core.hooksPath .githooks
	@echo "✅ git hooks 已启用（core.hooksPath=.githooks）"

doctor-hooks:
	@[ "$$(git config --get core.hooksPath)" = ".githooks" ] || { \
	  echo "ERROR: core.hooksPath 未指向 .githooks，请运行 make install-hooks"; \
	  exit 1; \
	}
	@echo "✅ hooks 配置正确"

# doctor-hooks-local: ci 链的本地 fail-fast 检查。
# CI 环境（$$CI 或 $$GITHUB_ACTIONS 已设）跳过，因为 CI 不需要 git hooks；
# 本地环境强制要求 core.hooksPath=.githooks，避免开发者跳过 pre-commit secret 扫描。
doctor-hooks-local:
	@if [ -n "$$CI" ] || [ -n "$$GITHUB_ACTIONS" ]; then \
	  echo "doctor-hooks-local: CI 环境，跳过 hooks 检查"; \
	else \
	  [ "$$(git config --get core.hooksPath)" = ".githooks" ] || { \
	    echo "ERROR: 本地 git hooks 未启用 (core.hooksPath != .githooks)"; \
	    echo "       这会跳过 pre-commit secret 扫描，请运行: make install-hooks"; \
	    exit 1; \
	  }; \
	  echo "✅ 本地 hooks 已启用"; \
	fi

# sync-main: 拉取远端 main 并尽量 fast-forward 本地 main。
# 对应 .agent/runtime/standard/goal-runtime-canonical.md RULE-MAIN-SYNC-002：
# 每个 worktree 创建前必须基于最新 main。
sync-main:
	@git fetch origin main
	@CUR=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$CUR" = "main" ]; then \
	  git merge --ff-only origin/main && echo "✅ main 已同步至 origin/main"; \
	else \
	  LOCAL=$$(git rev-parse main 2>/dev/null || echo ""); \
	  REMOTE=$$(git rev-parse origin/main); \
	  if [ "$$LOCAL" = "$$REMOTE" ]; then \
	    echo "✅ 本地 main 已是最新（当前在 $$CUR）"; \
	  else \
	    echo "⚠️  当前不在 main 分支（$$CUR）"; \
	    echo "   本地 main: $$LOCAL"; \
	    echo "   远端 main: $$REMOTE"; \
	    echo "   请到主 worktree 执行: git merge --ff-only origin/main"; \
	    exit 1; \
	  fi; \
	fi
