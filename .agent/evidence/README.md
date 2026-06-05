# goalcli evidence ledger

`.agent/evidence/ledger.jsonl` 是 goalcli v0.1.0 的 source evidence ledger。`release/evidence/goalcli/` 可以保存 generated evidence packs，但这些 release artifacts 不是 source of truth，也不能作为 canonical ledger input。

G12-G16 commands 是由 `goalcli` 和 Harness Runtime 背书的 goalcli MVA-blocking checks。fresh ledger-backed evidence 只有在 `goal-runtime-final` 调和同一 `GOAL_ID` 下的 `goal-acceptance`、`goal-delivery`、`goal-handover`、`goal-downstream-adoption` 和 `goal-certify` 后，才能报告 `mva_status: complete`。

`goal-downstream-adoption` 和 `goal-runtime-final` 只声明 resiliencx 本地 contract 已覆盖 worker workspace gap，不声明真实 downstream 仓库已采用、已发布或 proof-based adoption。缺少外部 downstream 仓库 Evidence 时，相关 Evidence 必须保留 `adoption_claim=not_claimed`、`downstream_adoption_scope=local_contract_only`、`proof_based_adoption=false`、`downstream_repo_write=false`。
