# goalcli mutating automation guardrails

Status: #53 future guardrail design。该文档定义未来 mutating automation 的最小安全合约；当前 v0.2.0 不实现 `worktree create/clean`、`issues sync`、`pr create/update`、`release publish` 等 mutating commands。

## Authority 和非目标

- Parent issue: #45。
- Direct issue: #53。
- Gap ledger: `docs/plans/goalcli-v0.2.0-gap-ledger.md`。
- 唯一实现入口保持为 `cmd/goalcli`。
- 非目标: 当前不新增任何 mutating command，不写 GitHub Issue/PR/release，不写 downstream repo，不读取 secrets，不删除 worktree，不创建 release artifact。

## Universal guardrails

每个未来 mutating command 必须同时满足下列条件：

1. **Dry-run first**: 默认只输出计划、diff、目标资源、权限需求、rollback plan 和 evidence plan；不得有隐式写入。
2. **Explicit apply**: 真正执行必须提供显式 apply flag，例如 `--apply`，并在高风险资源上追加 `--confirm <resource>` 或等价确认。
3. **Permission boundary**: 输出必须标注 side-effect class：`local_write`、`git_write`、`github_write`、`release_publish`、`downstream_write`。外部生产副作用必须先获得明确授权。
4. **Preflight gates**: 在执行前验证 clean state、目标资源唯一性、权限/token scope、execution context、required evidence freshness。
5. **Rollback plan**: 执行前显示可行 rollback；不可完全 rollback 的动作必须显式说明只能 close/comment/revert/patch，而不能“撤销历史”。
6. **Evidence ledger**: 执行后必须记录 command、inputs hash、dry-run summary、preflight verdict、changed resources、rollback note、external URLs/checksums。
7. **Idempotency**: 重跑必须检测已存在资源，优先 report/no-op，不得重复创建 issue、PR、tag 或 release。
8. **Secret safety**: 不读取 `/home/k8s/secrets/env/*`，不把 token、cookie、private key、secret env value 写入 evidence。
9. **No downstream writes by default**: downstream adoption proof 只能验证；写 downstream repo 需要单独 future issue 和明确授权。

## Future command matrix

| Future command | Side-effect class | Dry-run output | Apply requirements | Rollback / evidence |
| --- | --- | --- | --- | --- |
| `worktree create` | `local_write` + optional `git_write` | base ref、目标 path、branch name、是否已有 worktree、将创建的目录/分支 | `--apply`；目标 path 不存在或为空；base clean；branch name 唯一；不能覆盖 evidence | rollback 使用 `git worktree remove` 和 branch cleanup，仅在 clean state；evidence 记录 path、base、branch、preflight。 |
| `worktree clean` | destructive `local_write` | 将删除的 path、dirty/untracked/evidence 文件清单、保全计划 | `--apply --confirm <path>`；worktree clean；untracked evidence 已复制或明确保留；不得跨越 repo root | dirty 或 evidence 未保全时 block；rollback 通常不可完全恢复，所以 evidence 必须先保全。 |
| `issues create` | `github_write` | issue title/body/labels/milestone、目标 repo、dedupe key、API request preview | 明确外部副作用授权；token scope 检查；`--apply`；dedupe 未命中 | rollback 只能 close/comment，不保证删除；evidence 记录 URL、request hash、dedupe result。 |
| `issues sync` | `github_write` | 每个 issue 的 field diff、labels diff、comment/update plan、冲突清单 | 明确外部副作用授权；`--apply`；远端 version 未漂移；rate-limit 预检 | rollback 只能追加更正 comment 或反向 update；evidence 记录 before/after URL 和 diff hash。 |
| `pr create` | `github_write` | branch、base、title/body、reviewers/labels、linked issues、CI expectation | 明确外部副作用授权；branch pushed/可访问；`--apply`；body 通过 docs/governance checks | rollback close PR；不能删除 review history；evidence 记录 PR URL、body hash、check URLs。 |
| `pr update` | `github_write` | body/labels/reviewers/comment diff、远端 current state | 明确外部副作用授权；`--apply`；远端 state 未漂移或显式接受 merge | rollback 用反向 update/comment；evidence 记录 before/after diff 和 URL。 |
| `release publish` | `release_publish` + `github_write` | version/tag、manifest checksum、artifact list、release notes、gate verdict、不可逆风险 | `release_verify` context；`release-ready` PASS；`evidence-replay` PASS；explicit version confirmation；`--apply --confirm <version>` | rollback 通常是 yanking/patch release/errata，不是完全撤销；evidence 记录 checksums、release URL、gate output。 |
| `commit create` | `git_write` | staged diff summary、Lore commit message、affected files、test evidence | `--apply`；working tree scope clean；message 符合 Lore trailers；无 unrelated staged files | rollback 用 reset/revert；evidence 记录 commit hash 和 tested commands。 |
| `release prepare` that writes artifacts | `local_write` | 将生成的 manifest/evidence pack/release notes path 和 checksum plan | `--apply`；输出目录受控；不会覆盖 source ledger；生成物标记为 artifact | rollback 删除生成 artifact；evidence 记录 checksum、source inputs、generated paths。 |

## Required command contract

未来任何 mutating command 的 CLI contract 至少包含：

- `--dry-run` 或默认 dry-run behavior；如果默认 dry-run，则 help 必须说明。
- `--apply` 作为唯一执行开关。
- 高风险资源的 `--confirm <resource>`，例如 worktree path、issue number、PR number、release version。
- `--context <execution-context>`，并由 `execution-context` gate 判断该 command 是否允许在当前 context 执行。
- JSON output 中的 `side_effects[]`、`preflight`、`rollback`、`evidence`、`external_authorization_required` fields。

## Current v0.2.0 block list

以下内容在当前 v0.2.0 scope 中必须保持未实现：

- `worktree create`、`worktree clean` 的实际创建/删除。
- `issues create`、`issues sync` 的 GitHub 写入。
- `pr create`、`pr update` 的 GitHub 写入。
- `release publish` 的 tag/release/artifact 发布。
- 任何 downstream repository 写入。
- 任何 `tools/goalcli` 或非 `cmd/goalcli` 的第二实现入口。

## Acceptance checklist

- 每个 future mutating command 都有 dry-run-first、explicit apply、rollback、evidence、side-effect 说明。
- GitHub Issue/PR/release 相关动作明确标注为 external side effect，并要求显式授权。
- Worktree deletion 明确要求 clean state 和 evidence preservation。
- 当前文档只提供 guardrails，不提供 mutating implementation。
