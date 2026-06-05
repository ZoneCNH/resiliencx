# Full Goal Runtime v3.1

Runtime: `resiliencx` Full Goal Runtime v3.1.

## 目的

把 Goal、Spec、Requirement、Acceptance Criteria、Design、ADR、Plan、Task、Test、Evidence、Risk、Decision、Review、Release、Retrospective 和 Patch 连接为同一条可审计链。完成不是 MVA，而是 `DONE with evidence:`：REQ-001..REQ-010 全部关闭、release-final-check 通过、score >= 9.8、kernel downstream integration 通过、manifest 完整。

## 必需工件

- `.agent/runtime/object-model.md`
- `.agent/runtime/state-machine.md`
- `.agent/traceability/traceability-matrix.md`
- `.agent/harness/harness.yaml`
- `.agent/evidence/evidence-protocol.md`
- `.agent/docs/review-template.md`
- `.agent/release/release-template.md`
- `.agent/docs/retrospective-template.md`
- `.agent/traceability/risk-register.md`
- `.agent/traceability/decision-log.md`
- `.agent/runtime/rollback-protocol.md`
- `.agent/docs/prompt-patches.md`
- `.agent/harness/harness-patches.md`
- `.agent/docs/rule-patches.md`
- `.agent/evidence/truth-state.yaml`
- `.agent/registries/command-implementation-status.yaml`
- `.agent/release/release-required-gates.yaml`
- `.agent/evidence/evidence-usability.yaml`
- `.agent/registries/downstream-adoption-status.yaml`

## 停止条件

只有当 Evidence 证明当前 slice 完成，或 blocker 已记录 owner、scope 和 next action 时才停止。最终 goal claim 必须包含命令、输出、manifest/checksum 状态、score 状态和 known gaps。
