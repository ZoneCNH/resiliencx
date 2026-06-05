# goalcli v0.1.0 runtime standard

Status: v0.1.0 MVA slice 的 normative authority。

## Authority

- `cmd/goalcli` 是 goalcli v0.1.0 commands 的唯一机器执行面。
- Harness Runtime 是 routing、policy 和 evidence interpretation 的 control plane。
- goalcli v0.1.0 不引入第二套并列执行面。
- `.agent/evidence/ledger.jsonl` 是 source evidence ledger。
- `release/evidence/goalcli/` 是 generated evidence pack 目录，不是 source ledger。

## Command surface

PR-4 command-backed slice 通过委托给 `cmd/goalcli` 的 Makefile targets 暴露 G12-G16 等价 gate：

| Gate | Command / target | Blocking |
| --- | --- | --- |
| G12 acceptance | `goal-acceptance` | yes |
| G13 delivery | `goal-delivery` | yes |
| G14 handover | `goal-handover` | yes |
| G15 downstream adoption | `goal-downstream-adoption` | yes |
| G16 certify | `goal-certify` | yes |
| G12-G16 final report | `goal-runtime-final` | yes |

每个命令都需要 `GOAL_ID` 或 `--goal-id`。同一 goal 的 fresh ledger-backed evidence 只有在 `goal-runtime-final` 调和全部 G12-G16 checks 后，才能报告 `mva_status: complete`。

`audit-goal` 是本地只读聚合审计入口，用于在不写 evidence 的前提下验证 goal/REQ/task/issue/evidence/release 链路。它会运行本地治理检查、traceability matrix 检查，以及 G12-G16 runtime 命令的 `--dry-run --verify` 形式；它不能替代带 `--write-evidence` 的 fresh ledger-backed `goal-runtime-final`。

`dashboard-generate` 复用 `audit-goal` 的本地只读基线，并将 component 状态、gap 摘要、goal id、matrix、scope 和 write-evidence policy 渲染为稳定 JSON 或 Markdown。该 dashboard 不包含时间戳，不写 source ledger，也不能替代 fresh ledger-backed completion evidence。

## Completion rule

只有同时满足以下条件，才能声明 goalcli v0.1.0 MVA complete：

- Harness policy 将 G12-G16 标记为 required。
- fresh command evidence 已记录到 `.agent/evidence/ledger.jsonl`。
- `release/evidence/goalcli/` 下的 generated packs 由 source ledger 派生。
- root plan 与 roadmap aliases 已在 evidence ledger 中调和。
- final report 的同一 `GOAL_ID` 已包含五个 prerequisite command entries。
