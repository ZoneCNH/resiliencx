# Object Model

| Object | Required fields | Evidence link |
| --- | --- | --- |
| Goal | id, target outcome, non-goals, final gates | docs/goal.md, .agent/runtime/goal.md |
| Spec | behavior, constraints, compatibility | docs/spec.md, .agent/docs/spec.md |
| Requirement | REQ id, owner, status, AC ids | .agent/traceability/traceability-matrix.md |
| AC | observable condition, proof command/artifact | docs/goal.md |
| Design | boundary, interface, dependency direction | docs/design.md, .agent/docs/design.md |
| ADR | decision, context, consequences | docs/adr/*.md |
| Plan | phases, sequencing, risk gates | .agent/docs/plan.md |
| Task | id, owner/scope, lifecycle status | .agent/docs/tasks.md, OMX task state |
| Test | command, expected result, owner | .agent/harness/harness.yaml |
| Evidence | artifact, command, checksum, actor, timestamp | .agent/evidence/evidence-protocol.md |
| Risk | risk, impact, mitigation, owner | .agent/traceability/risk-register.md |
| Decision | decision, reason, rejected alternative | .agent/traceability/decision-log.md |
| Review | findings, verdict, required fixes | .agent/docs/review-template.md |
| Release | version, gates, manifest, checksum | .agent/release/release-template.md |
| Retrospective | incident/learning, patch candidates | .agent/docs/retrospective-template.md |
| Patch | prompt/harness/rule patch, trigger, test | .agent/docs/prompt-patches.md, .agent/harness/harness-patches.md, .agent/docs/rule-patches.md |
