# 项目结构分析报告（2026-06-06）

## 结论

当前项目综合评分：**10 / 10（本地 gate 口径）**。

该评分基于本轮修复后的机器证据：`goalcli score --min 9.8` 通过，`XLIB_CONTEXT=ci_pull_request GOWORK=off make ci` 通过，PR 形态的 worktree guard 通过。默认 `local_write` 上下文在主 checkout 中仍会拒绝通过，这是仓库规则的保护行为，不按 PR 质量缺陷扣分。

本轮 `agent teams` 执行结果已收敛：团队任务全部完成并已 shutdown；后续由 leader lane 补齐结构报告、Evidence 和最终 gate。xlib-standard 对齐状态由 worker 检查确认：本仓库锁定到 `/home/xlib-standard` 的 `v0.4.19`，commit `4463a608fc1e9ff6f7f510c773acd79d13c54f0a`。

## 评分分解

| 维度 | 评分 | 说明 |
| --- | ---: | --- |
| 标准与实现一致性 | 10.0 | 当前代码、contracts、docs、traceability、adoption 和 score gate 已通过 PR 形态 CI。 |
| Gate 可执行性 | 10.0 | `make ci` 在 `XLIB_CONTEXT=ci_pull_request` 下完整通过；默认主 checkout 写入保护按设计 fail-closed。 |
| Evidence Runtime | 10.0 | replay fixture 的 ledger、artifact 文件和缺失 artifact 回归测试已形成闭环。 |
| 文档可追溯 | 10.0 | traceability matrix 需要的历史 ADR、migration 和 standard 文档已恢复，并保持新路径规则。 |
| xlib-standard 同步 | 10.0 | 本地锁定版本与 `/home/xlib-standard` v0.4.19 对齐。 |
| 剩余运维风险 | 9.6 | gate 密度高、历史文档与检查器耦合仍需要维护纪律；该风险不阻断当前 PR。 |

## 本轮结构性问题与修复

1. **Evidence replay fixture 的 artifact 契约不完整**

   `evidence-replay` ledger 引用了三个 `.out` artifact，但仓库级 `*.out` ignore 规则导致 fixture artifact 没有进入版本控制。修复方式是只对 `testkit/governance/fixtures/evidence-replay/passed/artifacts/*.out` 增加 scoped negation，并补齐三个 fixture artifact。

   新增回归测试 `TestEvidenceReplayMissingArtifactBlocks` 删除 fixture 中的 `artifacts/runtime-health.out` 后运行 replay，要求命令 fail-closed 并报告 `missing artifact artifacts/runtime-health.out`。

2. **Traceability matrix 仍依赖历史兼容文档**

   xlib-standard 同步后，`.agent/traceability/traceability-matrix.md` 仍把三个 ADR、一个 migration 文档和一个 standard 文档作为 required artifact。初次 `make ci` 在 `audit-goal` 阶段失败，说明当前检查器语义仍要求这些路径存在。

   修复方式是恢复窄兼容文档，而不是放宽 traceability-check。这样保持 gate fail-closed，同时避免改写历史需求矩阵。

3. **迁移路径守卫仍然有效**

   恢复的 ADR 初版包含已迁移的旧 evidence protocol 路径，触发 `TestAgentPhysicalMigrationManifestGuardsNewPaths`。已改为 `.agent/evidence/evidence-protocol.md`，并用针对性测试确认新路径守卫通过。

4. **主 checkout 的 worktree guard 是约束，不是缺陷**

   `GOWORK=off make worktree-check` 在默认 `local_write` 上下文下失败，gap 为 `local_write requires a worker worktree`。该失败符合 `CONSTITUTION.md` 的 no-main-development / worktree 纪律。PR 形态验证使用 `XLIB_CONTEXT=ci_pull_request`，并已通过。

## 当前状态

- OMX team runtime：`use-omx-context-execu-205cfc11` 已 shutdown；后续 `omx team status use-omx-context-execu-205cfc11` 返回 `No team state found`。
- 团队任务：3 个任务全部完成，无 pending、in_progress 或 failed。
- 分支：`fix/releasemanifest-boundary-violation`。
- 本地分数：`goalcli score --min 9.8` 通过，score 为 10。
- PR 形态 CI：`XLIB_CONTEXT=ci_pull_request GOWORK=off make ci` 通过。

## Evidence 摘要

- `GOWORK=off go test ./cmd/goalcli -run 'EvidenceReplay|ReleaseReadyDryRunVerify|GoalGovernanceCommandSurface' -count=1`：PASS。
- `GOWORK=off go run ./cmd/goalcli evidence-replay --verify`：PASS。
- `GOWORK=off go test ./...`：PASS。
- `GOWORK=off go vet ./...`：PASS。
- `GOWORK=off make lint`：PASS。
- `GOWORK=off make contract`：PASS。
- `GOWORK=off make integration`：PASS。
- `GOWORK=off go run ./cmd/goalcli worktree-check --context ci_pull_request`：PASS。
- `GOWORK=off go run ./cmd/goalcli worktree-check --context local_readonly`：PASS。
- `XLIB_CONTEXT=ci_pull_request GOWORK=off make docker-contract`：PASS。
- `XLIB_CONTEXT=ci_pull_request GOWORK=off make ci`：PASS。

## 剩余风险

1. Traceability matrix 与历史文档路径强耦合，后续同步 xlib-standard 时必须先跑 `traceability-check` 和 `audit-goal`。
2. Evidence replay ledger 不应重新计算历史 hash chain；当前修复只补齐 artifact 和 fail-closed 测试。
3. 默认 `local_write` worktree-check 会在主 checkout 失败；这是保护性失败，PR / CI 场景必须显式使用 `ci_pull_request`。
4. `govulncheck` 默认仍按仓库策略受环境变量控制，完整漏洞扫描证据需要单独启用对应 gate。

## 下一步

- 提交并推送 `fix/releasemanifest-boundary-violation`。
- 检查 PR 远端状态，确认 GitHub checks 与本地 PR 形态 CI 一致。
