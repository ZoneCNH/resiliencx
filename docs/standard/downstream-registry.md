# Downstream Registry

Goal v2.9.3 的代表下游以 patch-only 方式验证，不直接写外部仓库。

| Repo | Role | Mode | Notes |
| --- | --- | --- | --- |
| `kernel/configx` | L0 representative downstream | `patch-only` | 用于 `downstream-baseline` 与 `downstream-adoption` dry-run。 |
| `kernel` | integration smoke | `patch-only` | 保留既有 integration 代表下游语境。 |

`downstream-baseline --repo kernel/configx --mode patch-only` 与 `downstream-adoption --repo kernel/configx --mode patch-only` 只产生命令报告；真实 adoption 需要独立 PR/Evidence。

## Proof contract

Proof-based downstream adoption must be represented by `contracts/downstream-adoption-proof.schema.json` before any registered target can be called adopted. The current registry intentionally has no such proof, so `downstream-adoption --repo kernel/configx --mode patch-only --verify` remains `status: gap` when the downstream repo is unavailable in the worker workspace.

Minimum proof fields:

- `source_repo` and `source_commit`: the source standard repository and exact commit that produced the patch/adoption package.
- `downstream_repo` and `downstream_commit`: the downstream repository and exact commit where the proof was generated.
- `mode`: one of `patch-only`, `dry-run`, or `pr-plan`; mode alone is never proof of adoption.
- `gate_outputs`: command outputs with `command`, `status`, `artifact_path`, and `sha256` for every blocking downstream gate.
- `rollback`: rollback `strategy`, accountable `owner`, and executable `commands` for reverting the adoption.

Absent proof, missing gate outputs, stale artifacts, or a missing downstream workspace must be reported as a `gap`; the source repository must not modify downstream repos while proving this contract.
