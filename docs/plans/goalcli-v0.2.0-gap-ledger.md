# goalcli v0.2.0 gap ledger

Status: #46 决策账本。该文档把 `/home/resiliencx/.worktree/goal-patch.md` 中的命令提案收敛到 v0.2.0 scope，并为 #48-#52 的实现任务和 #53 的 mutating automation guardrails 提供反向链接。

## Scope 和硬约束

- Parent issue: #45。
- 直接 issue: #46。
- 提案来源: `/home/resiliencx/.worktree/goal-patch.md`。
- 当前已落地基线: #47 已把 `audit-goal` 接入 `cmd/goalcli`，并保持 read-only audit report 合约。
- 唯一实现入口: `cmd/goalcli`。不得新增或复活 `tools/goalcli`、独立脚本式 CLI、或绕过 command registry 的第二入口。
- v0.2.0 的默认姿态是 read-only verification/reporting。任何本地写入、GitHub 写入、PR 写入、release 发布、downstream repo 写入都必须先满足 #53 guardrails，并在未来 issue 中单独实现。

## Classification

| 分类 | 含义 | v0.2.0 处理规则 |
| --- | --- | --- |
| `implement` | v0.2.0 必须补齐的 canonical command 或现有 planned marker 的语义升级 | 只能在 `cmd/goalcli` 中实现，并同步 registry、governance check、tests 和文档 evidence |
| `alias-to-existing` | 提案命名可映射到现有 canonical command、Makefile target 或 registry entry | 不新增命令；在文档或 help 中保持别名解释，避免 command surface 膨胀 |
| `defer` | 需要写入文件系统、Git、GitHub、release endpoint、downstream repo，或需要额外授权/rollback/evidence contract | 当前只记录取舍和 guardrails；不得实现 mutating automation |
| `reject` | 与唯一入口、read-only boundary、安全策略或 v0.2.0 scope 冲突 | 不建新 implementation issue；保留理由防止重复探索 |

## Decision matrix

| Root proposal / gap | Decision | Canonical target | Issue / reverse link | Rationale |
| --- | --- | --- | --- | --- |
| `audit-goal` | `alias-to-existing` | `cmd/goalcli audit-goal` | #47 | 已作为 read-only audit baseline 落地；后续命令应复用其 JSON/report 证据，不再新增重复审计入口。 |
| `dashboard-generate` | `implement` | `cmd/goalcli dashboard-generate` | #48 | 需要把 audit/governance/registry evidence 汇总为 dashboard-ready read-only output；不得写 dashboard artifact 或外部系统。 |
| `evidence-replay` | `implement` | `cmd/goalcli evidence-replay` | #49 | 需要 fixture-backed replay，证明 evidence gates 可重复验证；不得依赖 runtime `.omx` state 或生成未声明副作用。 |
| `release-ready` / `release prepare` 的 read-only readiness 部分 | `implement` | `cmd/goalcli release-ready` | #50 | 需要输出 release verdict、reasons、score、gates、replay/context 证据；仅做 readiness 判断，不发布 release。 |
| `runtime-file-ownership` | `implement` | `cmd/goalcli runtime-file-ownership` | #51 | 需要校验 owner/review rule/context 合约，报告 unknown owner、missing review rule、invalid context；不得引入机器绝对路径。 |
| `execution-context` / `context scan` 的语义检查部分 | `implement` | `cmd/goalcli execution-context` | #51 | 需要区分 `local_write`、`release_verify` 等 execution context，并约束 command side-effect boundary。 |
| `downstream-adoption` | `implement` | `cmd/goalcli downstream-adoption` | #52 | 需要明确 downstream proof contract；没有 proof 时必须报告 gap，不得写 downstream repo。 |
| mutating automation 总体 | `defer` | #53 guardrails doc | #53 | 当前只设计权限、dry-run、rollback、evidence contract；不得实现 mutating command。 |
| `worktree create` | `defer` | future guarded command | #53 | 会创建本地 worktree/branch，属于 local write；必须先具备 dry-run、explicit apply、rollback 和 evidence。 |
| `worktree clean` | `defer` | future guarded command | #53 | 可能删除文件；必须验证 clean state、保全 evidence、显式确认路径后才可进入未来实现。 |
| `worktree check` / `worktree-check --context local_write` | `alias-to-existing` | `cmd/goalcli worktree-check`、`worktree-guard`、`context-check` | existing command surface | 当前已有 read-only guard/check 命令；不要新增破折号/空格两套入口。 |
| `issues create` | `defer` | future guarded command | #53 | GitHub Issue 创建是外部副作用；必须要求显式授权、token scope、dry-run diff 和不可完全 rollback 的说明。 |
| `issues sync` | `defer` | future guarded command | #53 | GitHub issue 更新/同步是外部副作用；当前只允许 read-only registry/audit。 |
| `issues status` | `alias-to-existing` | `cmd/goalcli issue-registry`、future `dashboard-generate` | #48 | 状态查询可由现有 registry 和未来 dashboard read-only output 承担，不新增重复命令。 |
| `pr create` | `defer` | future guarded command | #53 | GitHub PR 创建是外部副作用；必须有 dry-run body、branch validation、explicit apply 和 rollback note。 |
| `pr update` | `defer` | future guarded command | #53 | PR body/labels/comments 更新是外部副作用；当前 v0.2.0 不实现。 |
| `pr ready` / `pr-check --context ci_pull_request` | `alias-to-existing` | `cmd/goalcli pr-check`、future `release-ready` | #50 | Readiness check 已有 read-only command；release 合并判断由 #50 扩展，不新增 mutating PR 命令。 |
| `release publish` | `defer` | future guarded command | #53 | 发布 release 是外部/公共副作用；必须经过 `release_verify` context、evidence replay、explicit version confirmation。 |
| `goal init` | `defer` | future guided bootstrap | #53 | 初始化会写项目文件；当前 v0.2.0 不创建 scaffolding。 |
| `spec check` | `alias-to-existing` | `cmd/goalcli spec-check` | existing command surface | 已有 canonical read-only check。 |
| `design check` | `alias-to-existing` | `cmd/goalcli design-check` | existing command surface | 已有 canonical read-only check。 |
| `tasks check` | `alias-to-existing` | `cmd/goalcli task-check` | existing command surface | 已有 canonical read-only check。 |
| `evidence check` / `make evidence-check` | `alias-to-existing` | `cmd/goalcli evidence-check`、Makefile target | existing command surface | 已有 evidence read-only gate；future replay 由 #49 负责。 |
| `evidence collect` | `defer` | future evidence-pack command | #53 / #49 | 收集可能写 release/evidence artifact；当前只允许已声明的 read-only check/replay。 |
| `goalcli schema validate` | `alias-to-existing` | `contracts`、`policy-schema`、`context-schema-check` | existing command surface | 当前 schema gates 已拆为 canonical checks；不新增泛化命令。 |
| `goalcli commit create` | `defer` | future guarded command | #53 | 创建提交是本地 VCS mutation；需要 dry-run message、explicit apply、rollback/revert guidance。 |
| `retro generate` | `alias-to-existing` | `retro-check`、`self-improving-check` | existing command surface | 当前只保留 read-only retro/self-improvement checks；生成 artifact 属于 future write。 |
| `patch propose` | `defer` | future proposal workflow | #53 | 生成 patch 可能写文件或分支；当前 v0.2.0 不实现。 |
| `tools/goalcli` implementation path | `reject` | none | none | 与 `cmd/goalcli` 唯一实现入口冲突，会制造双重 authority。 |
| Downstream repo writes | `reject` for v0.2.0 | none | #52 only verifies proof | v0.2.0 只能验证 downstream adoption proof；不得修改 downstream repository。 |

## Implementation issue reverse links

- #48: `dashboard-generate` read-only dashboard output。
- #49: `evidence-replay` fixture-backed evidence replay。
- #50: `release-ready` release readiness verdict。
- #51: `runtime-file-ownership` and `execution-context` semantic checks。
- #52: `downstream-adoption` proof contract validation。
- #53: future-only mutating automation guardrails；该 issue 不授权当前实现 mutating commands。

## Acceptance checklist

- 每个 deferred/rejected entry 都有理由。
- 新实现项都有 issue reverse link。
- `cmd/goalcli` 是唯一实现入口。
- Mutating automation 只进入 #53 guardrail design，不进入当前 implementation。
- Verification command: `GOWORK=off go run ./cmd/goalcli docs-check`。
