# goalcli CLI Contract

`goalcli` 是 resiliencx v2.9.3 的 machine-verifiable gate surface。除明确委托给既有脚本的命令外，所有命令必须输出包含 `command`、`status`、可选 `details` 和可选 `gaps` 的 JSON report，并符合 `contracts/goalcli-report.schema.json`。

该 CLI 只执行本地非破坏性 contract checks：不读取真实 secrets，不连接外部生产系统，也不修改 downstream 仓库。涉及 GitHub、release、runtime 或 downstream 状态的命令，在当前版本中以本地 contract、manifest 和 dry-run 证据表达。

## 退出码语义

- `status=passed` 必须返回 `0`。
- `status=failed`、`status=planned` 和 `status=gap` 必须返回 `1`。
- 未知命令、非法参数或非法 context 必须返回 `2`。
- downstream 仓库不存在时必须报告 `status=gap` 且返回非 0；该结果不能作为 release gate 成功证据。
- 带 `--verify` 或 `--strict` 的命令遇到 `planned`/`gap` 时必须阻断 gate。

## P0 commands

- `version`
- `doctor`
- `minimal-kernel`
- `main-guard --context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify`
- `worktree-guard --context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify`
- `worktree-check --context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify`
- `context-check`
- `spec-check`
- `design-check`
- `task-check`
- `pr-check --context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify`
- `evidence-check`
- `done-assertion`
- `cli-contract`
- `issue-registry`
- `command-registry`
- `makefile-baseline`
- `audit-goal`
- `dashboard-generate`
- `traceability-check [--matrix .agent/traceability/traceability-matrix.md] [--json]`
- `context-profile`
- `context-profile-check`
- `context-schema-check`
- `context-fast-check`
- `context-standard-check`
- `context-full-check`
- `boundary`
- `contracts`
- `dependency-check`
- `docs-check`
- `evidence`
- `manifest`
- `integration`
- `rules-verify`
- `release-evidence-check`
- `release-evidence-checksum-check`
- `release-evidence-hash`
- `release-final-check`
- `render-check`
- `rules-consistency-check`
- `score`
- `secrets`
- `security`
- `standard-impact-check`
- `downstream-sync-plan`
- `debt`
- `architecture`
- `domain`
- `docs-drift`
- `dependency-debt`
- `security-debt`
- `testing-debt`
- `implementation-debt`
- `downstream-debt`
- `debt-evidence`
- `debt-evidence-checksum-check`
- `debt-evidence-hash`
- `docker-toolchain-check`
- `docker-build`
- `docker-build-check`
- `docker-shell`
- `docker-ci`
- `docker-release-check`
- `docker-release-final-check`
- `docker-goalcli`
- `docker-goalcli-image`
- `docker-goalcli-version`
- `docker-runtime-check`
- `docker-drift-check`
- `docker-contract`

## 下游同步计划命令

`goalcli downstream-sync-plan [--impact-report <path>] [--output <path>|-] [--workspace-root <path>] [--format markdown|json]` 读取 `release/standard-impact/latest.md` 的同步判定，默认生成 `release/downstream-sync/latest.md`，并在 stdout 输出符合 `contracts/goalcli-report.schema.json` 的 JSON report。传入 `--output -` 时才把 markdown/json 计划写入 stdout。

该命令只生成本地同步计划和命令清单，不修改 downstream 仓库，不更新 `.agent/registries/downstream-adoption-status.yaml` 或 `.agent/evidence/truth-state.yaml`，不得作为 proof-based adoption。计划必须列出 `kernel`、L1、L2 和 `x.go` 的 blocked/not_required 结论，并保留 `adoption_claim=not_claimed`。

## P1 commands

- `agent-team-contract`
- `scope-lock`
- `pr-template`
- `acceptance-matrix`
- `runtime-health`
- `goal-runtime`
- `goal-acceptance --goal-id <GOAL-ID> --mode FULL --json`
- `goal-delivery --goal-id <GOAL-ID> --mode FULL --json`
- `goal-handover --goal-id <GOAL-ID> --mode FULL --json`
- `goal-downstream-adoption --goal-id <GOAL-ID> --mode FULL --json`
- `goal-certify --goal-id <GOAL-ID> --mode FULL --json`
- `goal-runtime-final --goal-id <GOAL-ID> --mode FULL --json`
- `naming`
- `upgrade-standard --dry-run --repo <path>`
- `conformance-profile`
- `downstream-registry`
- `self-healing-skeleton`
- `policy-schema`
- `github-settings --verify`
- `github-governance`
- `governance-fixture-test`
- `toolchain`
- `evidence-artifacts`
- `supply-chain`
- `autoresearch`
- `changelog`

## P2 commands

- `install-runtime --dry-run`
- `upgrade-runtime --dry-run`
- `release-ready`
- `evidence-replay`
- `attest-conformance --profile standard-source|l0-kernel`
- `pack-standard`
- `pack-gate`
- `pack-evidence`
- `runtime-file-ownership`
- `downstream-baseline --repo kernel/configx --mode patch-only`
- `downstream-adoption --repo kernel/configx --mode patch-only`
- `adoption-check [--verify] [--json] [--root <path>]`
- `execution-context`


## Adoption check command

`goalcli adoption-check [--verify] [--json] [--root <path>]` 是本地只读采纳验证器。它在渲染 downstream 仓库内检查 `resiliencx.lock`、`.githooks/pre-commit`、`.githooks/pre-push`、`.github/workflows/adoption-check.yml`、`.github/rulesets/protect-main.json`、`mk/governance.mk`、`.agent/harness/harness.yaml`、command/makefile registries 和 `Makefile` 的 `adoption-check` target；在 `github.com/ZoneCNH/resiliencx` 标准源仓库内运行时不要求 downstream lock，但仍验证 Repository Governance Pack 和 main ruleset。`--root` 用于指向待检查仓库；`--verify` 是与其他 blocking gate 对齐的兼容旗标；`--json` 是统一输出旗标，当前命令始终输出 JSON report。main ruleset 必须处于 active enforcement、保护默认分支、禁止 bypass actors，并要求 `adoption-check`、`governance-check` 和 `release-check`。标准源仓库没有 downstream lock，因此 source-template workflow 会跳过本命令；只有启用 governance pack 的渲染 downstream 仓库必须执行 `GOWORK=off make adoption-check`。发现缺口时输出 `status=failed` 并返回非 0。该命令不访问外部系统、不创建 downstream 仓库，也不写 Evidence。

## Goalcli MVA commands

Goalcli MVA commands（`goal-acceptance`、`goal-delivery`、`goal-handover`、`goal-downstream-adoption`、`goal-certify` 和 `goal-runtime-final`）用于证明 `goalcli v0.1.0` 的 G12-G16 evidence contracts，并且只在 goalcli MVA evidence scope 内是 blocking gate。它们不会创建第二套并列 goalcli 执行面，也不会修改 downstream 仓库。source authority 是 `cmd/goalcli` 执行面和 `.agent/evidence/ledger.jsonl`；`release/evidence/goalcli/` 下的 generated packs 只是派生产物。root goalcli plan 仍是 roadmap authority，完成证据来自同一 `GOAL_ID` 下已调和的 source ledger 和 final report。

## Goal audit command

`goalcli audit-goal [--matrix .agent/traceability/traceability-matrix.md] [--json]` 是本地只读聚合审计入口，用于一次性验证 goal、REQ、task、issue、evidence 与 release readiness 的关键链路。它复用 `context-check`、`spec-check`、`design-check`、`task-check`、`evidence-check`、`cli-contract`、`issue-registry`、`command-registry`、`makefile-baseline` 和 `traceability-check`，并以 `--dry-run --verify` 调用 `goal-acceptance`、`goal-delivery`、`goal-handover`、`goal-downstream-adoption`、`goal-certify`、`goal-runtime-final`。

该命令不传入 `--write-evidence`，不会写 `.agent/evidence/ledger.jsonl`，也不会修改 downstream 仓库。所有组件通过时返回 `0`；任一组件发现 gap 时返回 `1` 并在 `gaps` 中记录组件名、退出码和摘要；非法参数返回 `2`。

## Goal dashboard command

`goalcli dashboard-generate [--goal-id <id>] [--matrix .agent/traceability/traceability-matrix.md] [--format json|markdown]` 基于 `audit-goal` 的同一组本地只读 component checks 生成稳定 dashboard。默认输出 JSON，符合 `contracts/goalcli-dashboard.schema.json`；传入 `--format markdown` 时输出确定性的 Markdown 表格，便于人工审阅和 release handoff。

该命令不包含时间戳、随机 ID 或外部状态字段，不传入 `--write-evidence`，不会写 `.agent/evidence/ledger.jsonl`，也不会修改 downstream 仓库。所有组件通过时返回 `0`；任一组件发现 gap 时返回 `1`，在 `components` 中保留组件顺序和状态，并在 `gaps` 中记录组件名、退出码和稳定摘要；非法参数或未知 format 返回 `2`。

## Debt governance commands

Debt governance commands 是 P0 release-blocking gates。`goalcli debt` 运行完整 debt scanner，`architecture`、`domain`、`docs-drift`、`dependency-debt`、`security-debt`、`testing-debt`、`implementation-debt` 和 `downstream-debt` 运行同一策略的聚焦切片。scanner 复用本地 boundary、docs、dependency diff 和 secret checks；scanner 失败必须返回非 0，不能作为 passed evidence。

`goalcli debt-evidence` 将生成的 evidence 写入 `release/debt/latest.json`、`release/debt/latest.md` 和 `release/debt/latest.json.sha256`。这些 latest evidence 文件是可复现 release artifacts，并且故意被 git 忽略。P0 debt rules 不允许例外：`.agent/policies/debt/` 下的 policy 文件和 `.agent/registries/debt/` 下的 registry 文件不得引入 P0 exception markers，且 release verification 必须在 debt status 不是 `passed` 时失败。

## Docker Toolchain Runtime commands

Docker Toolchain Runtime commands 由 `scripts/docker/check_toolchain.sh` 与 `scripts/docker/docker_gate.sh` 委托执行，作为同一 `goalcli` gate surface 的容器化运行时契约。`docker-toolchain-check` 校验 Dockerfile、Compose、devcontainer、CI、manifest、docs 和 downstream 模板锚点；`docker-build`、`docker-build-check`、`docker-shell`、`docker-ci`、`docker-release-check`、`docker-release-final-check`、`docker-goalcli`、`docker-goalcli-image`、`docker-goalcli-version` 提供镜像和容器内 gate 入口；`docker-runtime-check`、`docker-drift-check`、`docker-contract` 证明运行时、漂移和契约面没有分叉。

这些命令不得绕过 `make ci`、`make release-check`、Harness gate 或 release manifest evidence。缺少 Docker daemon 时，静态 contract check 可以作为 drift evidence，但不能替代实际 image build evidence。

## Goalcli v0.1.0 MVA runtime commands

goalcli MVA commands 是由 `.agent/harness/harness.yaml` 背书的本地 `cmd/goalcli` evidence commands；它们把 command、Makefile 和 harness coverage 变成可执行契约。只有当同一 `GOAL_ID` 的 G12-G16 检查都已写入 source ledger 并由 final rollup 调和后，才能证明 goalcli v0.1.0 MVA completion。

- `goal-acceptance` 校验 `G12_ACCEPTANCE`。
- `goal-delivery` 校验 `G13_DELIVERY`。
- `goal-handover` 校验 `G14_HANDOVER`。
- `goal-downstream-adoption` 校验 `G15_DOWNSTREAM_ADOPTION`。
- `goal-certify` 校验 `G16_CERTIFY` 并保留 source-ledger evidence claim。
- `goal-runtime-final` 校验五个 prerequisite local gates 的 `G12_G16_FINAL` rollup。
- 直接运行 `goalcli <command> --json` 是只读检查，不会写 evidence。
- 只有显式传入 `--write-evidence` 时，命令才会写入 `.agent/evidence/ledger.jsonl`。
- `goal-runtime-final --write-evidence` 只有在 source ledger 已存在同一 `GOAL_ID` 的五个 prerequisite entries 后，才会写入 generated evidence pack。

## GoalCLI 同步契约

`goalcli` 整套体系必须作为同一个 contract surface 同步。新增、重命名、删除或改变任一命令语义时，同一变更必须同步以下权威面：`cmd/goalcli/main.go` 的 dispatch 与 usage、`cmd/goalcli/main_test.go` 的契约测试、`Makefile` 的 `GOALCLI` gate 路由、`.agent/registries/command-registry.yaml`、`.agent/registries/command-implementation-status.yaml`、`.agent/registries/makefile-baseline.yaml`、`.agent/registries/makefile-target-registry.yaml`、`.agent/harness/harness.yaml`、`.agent/harness/gates.md`、`contracts/goalcli-report.schema.json`、本文件、`.agent/docs/standard/goalcli-mapping.md` 和 `internal/goalcli/README.md`。

`cmd/goalcli/main_test.go` 是该同步契约的回归入口：`TestCommandRegistryRequiredCommandsMatchRegistryFile` 必须断言 `commandRegistryRequiredCommands()` 与 `.agent/registries/command-registry.yaml` 的 `commands[].name` 一致；`TestCommandRegistryCommandsStayDocumentedInUsage` 必须断言 registry 中所有命令都进入 `usage`；`TestCommandImplementationStatusCommandsStayRegistered` 必须断言 `.agent/registries/command-implementation-status.yaml` 中列出的命令全部已注册。

`docs-check` 是该同步契约的 drift guard：它必须检查上述 surface 的存在、关键命令锚点和 `GoalCLI 同步契约` 文档锚点。同步完成后至少运行：

- `GOWORK=off go test ./cmd/goalcli`
- `GOWORK=off make docs-check`
- `GOWORK=off make command-registry`
- `GOWORK=off make makefile-baseline`
- `GOWORK=off make cli-contract`

## 执行约束

- `--dry-run` 只能执行本地 contract、manifest 或 patch planning 检查。
- `--verify` 或 `--strict` 不能把 `planned`/`gap` 当作成功证据。
- `--context` 仅允许 `local_write`、`local_readonly`、`ci_pull_request`、`ci_main_verify` 和 `release_verify`。
- `--repo` 指向的 downstream 仓库不存在时，命令必须返回 `gap`，且不得自动创建目录。
- 新增 planned dry-run 命令时必须同步 `plannedCommandFiles`；新增非 planned 命令必须同步 `run` dispatch、usage、Makefile gate、registry、CLI contract 和测试表。

## 语义校验

`schema-check` 必须提供 `goalcli schema validate --all` 兼容入口并生成 `reports/schema-check.json`，用于校验 repo-local schema-bearing artifacts。

`issue-registry` 不只是文件存在检查。它必须校验 `.agent/registries/issue-registry.yaml` 中每个条目都具备非空 `title`、`command` 和 `evidence`，`status` 必须为 `implemented`，ID 必须匹配 `P0|P1|P2|CTX-###`、全局唯一，并且每个前缀从 `001` 连续编号。`context-profile-check` 复用该 registry 语义，不能用空文件或非连续 ID 作为通过证据。

planned command 的 dry-run 也必须读取对应文件并检查语义 marker。当前强制 marker 包括：`agent-team-contract` 的 `schema_version:`、`roles:`、`rule:`；`acceptance-matrix` 的 `schema_version:`、`acceptance:`；`runtime-health` 的 `schema_version:`、`checks:`、`toolchain`；goalcli MVA 命令在 `.agent/harness/harness.yaml` 中的 `goalcli_mva_gates:`、对应 `G12_ACCEPTANCE`/`G13_DELIVERY`/`G14_HANDOVER`/`G15_DOWNSTREAM_ADOPTION`/`G16_CERTIFY`/`G12_G16_FINAL` 与命令名；`execution-context` 的 `schema_version:`、`contexts:`、`local_write`、`ci_pull_request`、`release_verify`。这些命令不得退化为单纯路径存在检查。

## Context Runtime v4 命令

Context Runtime v4.0 新增可叠加的 profile baseline，但不替换现有 P0/P1/P2 command registry。以下命令受 registry 治理，必须同时保留在 `run` dispatch、`.agent/registries/command-registry.yaml`、`.agent/registries/issue-registry.yaml`、Makefile targets 和本 contract 中：

- `context-profile --profile lite|standard|full|release`
- `context-profile-check`
- `context-schema-check`
- `schema-check`
- `context-lite`
- `context-standard`
- `context-full`
- `context-release`
- `context-fast-check`
- `context-standard-check`
- `context-full-check`

`context-release` 不得调用 `release-check` 或 `release-final-check`；唯一允许方向是 `release-final-check` 在严格 release evidence check 前委托执行 `context-release`。Legacy aliases（`context-fast-check`、`context-standard-check`、`context-full-check`）必须保持可用。除非仓库内实际存在 `.agent/context` 文件，否则该基线不得宣称已落地这些文件。

## Debt command surface

`goalcli debt` supports `--section`, `--mode`, `--min-score`, and `--output json|markdown`. Aliases `architecture`, `domain`, `docs-drift`, `dependency-debt`, `testing-debt`, `implementation-debt`, and `security-debt` select the corresponding debt sections.
