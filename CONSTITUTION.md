---
constitution_version: v1.0.0
status: normative
scope:

* resiliencx
* kernel
* L1 libraries
* L2 adapter libraries
* downstream adoption repositories
  owner: ZoneCNH
  effective_date: YYYY-MM-DD
  last_updated: YYYY-MM-DD
  enforcement:
  primary_gate: xlibgate
  runtime_control_plane: .agent
  evidence_required: true

---

# CONSTITUTION.md

## 0. 目的

本宪法定义本仓库及所有采用本标准的下游仓库必须遵守的最高工程规则。

本文件是以下事项的最高事实源：

* 架构边界
* 分层依赖
* 开发流程
* Goal 执行
* Harness 门禁
* Evidence 证明
* Release 纪律
* 下游采纳
* Self-improving 机制

任何 README、Issue、PR、脚本、临时文档、Agent Prompt、生成计划，只要与本宪法冲突，均以本宪法为准。

---

## 1. 权威性与适用范围

本宪法适用于：

* `resiliencx`
* `kernel`
* L1 横切能力库
* L2 基础设施适配库
* L3+ 下游业务系统
* `.agent` 运行时控制平面
* 所有人工贡献者
* 所有自动化 Agent
* 所有 CI、脚本、发布流程

优先级顺序如下：

1. `CONSTITUTION.md`
2. `.agent/rules/`
3. `.agent/harness/`
4. `contracts/`
5. `docs/architecture/`
6. `AGENTS.md`
7. `README.md`
8. Issue / PR 描述
9. 临时生成文档

---

## 2. 规范语言

本文档中的关键词含义如下：

* `必须`：强制要求
* `禁止`：不得违反
* `应当`：默认应遵守，除非有明确证据说明例外
* `可以`：允许但非强制
* `需要 Evidence`：没有可验证证据，不允许标记完成

---

## 3. 核心工程铁律

### LAW-001：标准源铁律

`resiliencx` 是基础库标准、模板、治理规则、Harness、Evidence、下游采纳规则的标准源。

下游仓库禁止静默偏离标准。

### LAW-002：运行时控制平面铁律

`.agent/` 是 Goal 执行、上下文恢复、Harness、Evidence、规则、复盘、自我改进的运行时控制平面。

`.agent/` 必须结构化、版本化，并尽量机器可读。

### LAW-003：Evidence 铁律

没有 Evidence，不允许声明完成。

所有完成声明必须使用：

```text
DONE with evidence:
```

### LAW-004：Harness 门禁铁律

所有有意义的变更必须通过相关 Harness Gates。

Harness Gates 必须 fail-closed，即检查失败时默认禁止通过。

### LAW-005：禁止 main 开发铁律

禁止直接在 `main` 分支开发。

所有实现工作必须使用独立 `git worktree`。

### LAW-006：分层边界铁律

每一层只能依赖被允许的下层。

任何跨层、反向依赖、循环依赖、L2 互相耦合，都必须被 CI 或机器门禁拦截。

### LAW-007：安全与密钥铁律

密钥禁止出现在源码、README、测试日志、Release Manifest、PR 描述、Issue、Evidence 或生成文档中。

密钥必须来自批准的运行时配置或安全路径。

### LAW-008：Self-improving 铁律

任何重大失败、规则逃逸、重复人工修复、测试回归、架构歧义，都必须进入 Retrospective，并在必要时生成：

* Prompt Patch
* Harness Patch
* Rule Patch
* CI Gate Suggestion
* New Issue Candidate

---

## 4. 标准分层模型

标准分层如下：

```text
resiliencx
    ↓
L0: kernel
    ↓
L1: configx / observex / testkitx / resiliencx / schedulex
    ↓
L2: redisx / kafkax / postgresx / taosx / ossx / clickhousex / natsx
    ↓
L3+: x.go / market-data / macro-data / regime-engine / business systems
```

### 分层规则

| 层级            | 职责                                                   | 允许依赖                     | 禁止依赖         |
| ------------- | ---------------------------------------------------- | ------------------------ | ------------ |
| resiliencx | 标准源、模板、治理、门禁                                         | 最小工具依赖                   | 下游业务仓库       |
| L0 kernel     | error / lifecycle / clock / context / validation 等原语 | 标准库优先                    | L1 / L2 / L3 |
| L1            | 横切治理能力                                               | kernel                   | L2 / L3      |
| L2            | 基础设施适配                                               | kernel + L1              | 其他 L2 / L3   |
| L3+           | 业务系统                                                 | L0 / L1 / L2 / contracts | 上游内部实现细节     |

---

## 5. 标准仓库结构

合规仓库应当包含：

```text
.
├── .agent/
│   ├── README.md
│   ├── INDEX.md
│   ├── context/
│   ├── goals/
│   ├── harness/
│   ├── contracts/
│   ├── autoresearch/
│   ├── gstack/
│   ├── superpowers/
│   ├── ce/
│   ├── artifacts/
│   ├── runs/
│   ├── retrospective/
│   └── rules/
├── contracts/
├── docs/
├── examples/
├── internal/
├── pkg/
├── scripts/
├── testkit/
├── release/
│   └── manifest/
├── AGENTS.md
├── CONSTITUTION.md
├── Makefile
├── README.md
└── go.mod
```

---

## 6. Goal Runtime 协议

所有非平凡工作必须遵循以下链路：

```text
Goal
→ Context Recovery
→ Spec
→ Design
→ Plan
→ Tasks
→ Execution
→ Verification
→ Evidence
→ Review
→ Release
→ Retrospective
→ Self-improving
```

### 执行模式

| 模式       | 适用场景          | 必要产物                                                                                            |
| -------- | ------------- | ----------------------------------------------------------------------------------------------- |
| Lite     | 小变更           | Goal / Task / Evidence                                                                          |
| Standard | 普通 Issue      | Goal / Spec / Plan / Tasks / Tests / Evidence                                                   |
| Full     | 架构、发布、迁移、标准变更 | Goal / Spec / Design / ADR / Plan / Tasks / Tests / Evidence / Review / Release / Retrospective |

---

## 7. 标准对象模型

系统必须支持以下对象：

* Goal
* Spec
* Requirement
* Acceptance Criteria
* Design
* ADR
* Plan
* Milestone
* Task
* Test
* Evidence
* Risk
* Decision
* Review
* Release
* Retrospective
* Prompt Patch
* Harness Patch
* Rule Patch

---

## 8. 标准 ID 体系

所有持久对象必须使用稳定 ID：

```text
GOAL-YYYYMMDD-NNN
SPEC-<domain>-vX.Y
REQ-<spec-id>-NNN
AC-<req-id>-NNN
DESIGN-<domain>-vX.Y
ADR-YYYYMMDD-NNN
PLAN-<goal-id>-vX.Y
TASK-<goal-id>-NNN
TEST-<task-id>-NNN
EVID-<task-id>-YYYYMMDD-NNN
RISK-<goal-id>-NNN
DEC-YYYYMMDD-NNN
REV-<target-id>-YYYYMMDD-NNN
REL-YYYYMMDD-<domain>
RETRO-YYYYMMDD-NNN
PATCH-PROMPT-YYYYMMDD-NNN
PATCH-HARNESS-YYYYMMDD-NNN
PATCH-RULE-YYYYMMDD-NNN
```

---

## 9. Goal 状态机

Goal 执行必须遵循以下状态机：

```text
INIT
→ CONTEXT_READY
→ GOAL_READY
→ SPEC_READY
→ DESIGN_READY
→ PLAN_READY
→ TASKS_READY
→ EXECUTING
→ VERIFYING
→ REVIEWING
→ RELEASING
→ RETROSPECTING
→ DONE
```

异常状态包括：

```text
BLOCKED
FAILED
NEEDS_RESEARCH
NEEDS_DECISION
NEEDS_REPLAN
NEEDS_ROLLBACK
NEEDS_HUMAN_APPROVAL
INCONSISTENT_STATE
```

---

## 10. Harness Gates

每个仓库必须定义可机器检查的 Gates。

最低要求如下：

| Gate                | 目的          | 必要 Evidence                   |
| ------------------- | ----------- | ----------------------------- |
| Context Gate        | 上下文已恢复且不过期  | context report                |
| Goal Gate           | Goal 明确、有边界 | goal file                     |
| Spec Gate           | 需求与验收标准存在   | spec file                     |
| Design Gate         | 架构设计已完成     | design / ADR                  |
| Plan Gate           | 任务拆解完整      | plan file                     |
| Task Gate           | 任务可执行       | task list                     |
| Implementation Gate | 实现符合分层与代码规则 | lint / test / boundary report |
| Test Gate           | 测试通过        | test logs                     |
| Evidence Gate       | 完成证明存在      | evidence files                |
| Review Gate         | Review 完成   | review report                 |
| Release Gate        | 发布安全        | release manifest              |
| Retrospective Gate  | 经验已沉淀       | retrospective / patch         |

---

## 11. Evidence 协议

Evidence 必须满足：

* 可复现
* 可审查
* 可追溯到 Task / Test / Issue / Goal / Release
* 存储在稳定位置
* 能证明 Acceptance Criteria 已满足

推荐 Evidence 路径：

```text
.agent/runs/<run-id>/
docs/evidence/<goal-id>/
release/manifest/<release-id>/
```

没有 Evidence 的 Task、Issue、Goal、Release，不允许标记为完成。

---

## 12. Definition of Done

### Task DoD

Task 完成必须满足：

* 实现完成
* 测试通过
* Evidence 存在
* 没有违反分层依赖
* 没有泄露密钥
* 必要文档已更新

### Issue DoD

Issue 完成必须满足：

* 所有关联 Task 完成
* Acceptance Criteria 满足
* CI 通过
* Review 完成
* Evidence 已附加

### Goal DoD

Goal 完成必须满足：

* 所有 Requirement 满足
* Traceability Matrix 完整
* 风险已处理
* Release 或 Handoff 完成
* Retrospective 完成

### Release DoD

Release 完成必须满足：

* Release Manifest 存在
* Changelog 已更新
* 兼容性影响明确
* Rollback 路径存在
* 下游采纳影响已说明

---

## 13. Worktree 与分支规则

强制规则：

* `main` 必须受保护
* 禁止直接提交到 `main`
* 禁止直接 push 到 `main`
* 所有实现必须使用独立 `git worktree`
* 每个 worktree 必须绑定 Goal、Issue 或 Task
* 本地 Gates 通过后才能创建 PR
* `make worktree-check` 必须通过

---

## 14. AutoResearch 协议

以下情况必须触发 AutoResearch：

* 外部 API 行为不确定
* 依赖版本可能变化
* Issue 描述不完整
* 架构规则冲突
* 文档与代码不一致
* 测试失败原因不明确
* 发布影响不明确
* 安全或合规假设不明确

AutoResearch 输出必须包含：

* question
* source
* evidence
* confidence
* decision
* impact
* follow-up patch

---

## 15. Compound Engineering 协议

重复出现的问题禁止长期停留在手工处理状态。

以下内容一旦重复出现，应当沉淀为复用资产：

* scripts
* templates
* review checklists
* CI gates
* test fixtures
* documentation patterns
* migration procedures
* troubleshooting playbooks

重复人工动作应当升级为：

* Make target
* script
* Harness Gate
* template
* rule
* reusable library component

---

## 16. Self-improving 协议

每次重要执行后，应当产生 Retrospective。

Retrospective 可以生成：

* Prompt Patch
* Harness Patch
* Rule Patch
* CI Gate Suggestion
* Documentation Patch
* New Issue Candidate
* Risk Register Update

系统只有在经验被转化为可复用、可检查、可执行的规则后，才算真正改进。

---

## 17. 下游采纳规则

采用本标准的下游仓库必须保留：

* 分层边界
* 必要文件结构
* Harness Gates
* Evidence 协议
* Release Manifest 协议
* Worktree-only 开发
* 密钥处理规则
* Self-improving 闭环

下游仓库可以扩展实现细节，但禁止削弱宪法级规则。

---

## 18. 安全规则

以下行为禁止：

* 硬编码密钥
* 密钥进入测试日志
* 密钥进入 Release Manifest
* 密钥进入 PR 描述
* 密钥进入 README
* 密钥进入 Evidence
* 使用隐藏全局配置掩盖密钥来源

所有密钥来源必须显式、可审查、可替换。

---

## 19. Release 规则

每次 Release 必须包含：

* version
* change summary
* compatibility impact
* migration notes
* evidence links
* test results
* known risks
* rollback plan
* downstream adoption notes

Release Manifest 必须存储在：

```text
release/manifest/
```

---

## 20. Change Propagation Matrix

变更必须同步影响对象。

| 变更类型           | 必须同步更新                                       |
| -------------- | -------------------------------------------- |
| Goal 变更        | Spec / Plan / Tasks / Registry / Issue       |
| Spec 变更        | Design / Plan / Tasks / Tests / Traceability |
| Requirement 变更 | AC / Tasks / Tests / Evidence                |
| Design 变更      | ADR / Plan / Tasks / Risk / Docs             |
| Task 变更        | Evidence / Registry / Issue / PR             |
| Public API 变更  | Contracts / Examples / Docs / Release        |
| Storage 变更     | Migration / Tests / Rollback / Docs          |
| Config 变更      | Defaults / Secret Policy / Docs / Tests      |
| CI 变更          | Harness / Evidence / Release                 |
| Risk 变更        | Risk Register / Gate / Review                |

---

## 21. Traceability Matrix

所有非平凡 Goal 必须维护追踪矩阵：

| Requirement | Acceptance Criteria | Design   | Task   | Test   | Evidence | Status       |
| ----------- | ------------------- | -------- | ------ | ------ | -------- | ------------ |
| REQ-*       | AC-*                | DESIGN-* | TASK-* | TEST-* | EVID-*   | pending/done |

没有 Evidence 的 Requirement 禁止进入 Release。

---

## 22. 冲突裁决

当文档、代码、Issue、PR、Agent 输出互相冲突时，按以下顺序裁决：

1. `CONSTITUTION.md`
2. `.agent/rules/`
3. `.agent/harness/`
4. `contracts/`
5. `docs/architecture/`
6. `AGENTS.md`
7. `README.md`
8. Issue / PR 描述
9. 临时生成文档

当代码与文档冲突时，必须通过 Evidence、Review 和 Patch 解决，禁止默认相信任意一方。

---

## 23. 宪法修订流程

本宪法可以修订，但必须通过显式 Amendment。

每次 Amendment 必须包含：

* 修改原因
* 影响规则
* 迁移影响
* 下游影响
* 需要同步更新的 Gates
* 审批 Evidence
* 版本号更新

禁止静默削弱安全、Evidence、分层边界、Release 纪律或 Harness 门禁。

---

## 24. 最低 Make Targets

合规仓库应当提供：

```makefile
make check
make test
make lint
make boundary-check
make worktree-check
make evidence-check
make harness-check
make release-check
```

标准源仓库还应当提供：

```makefile
make constitution-check
make downstream-check
make self-improve-check
```

---

## 25. 不可接受条件

出现以下任意情况，变更不得接受：

* 绕过 Harness Gates
* 缺少 Evidence
* 违反分层边界
* 直接在 main 开发
* 引入隐藏全局状态
* 泄露密钥
* 削弱 Release 纪律
* Public API 变更但未更新文档和契约
* 没有测试或证明却标记完成
* 应沉淀为工具的重复工作仍以一次性脚本处理

---

## 26. 最终规则

健康的工程系统必须让以下链路可执行、可复现、可审查、可持续改进：

```text
Goal → Worktree → PR → Harness → Evidence → Release → Retrospective → Patch
```

本宪法的目的不是写规范，而是让工程质量可执行、可验证、可演进。

最关键的格式原则是：

`CONSTITUTION.md` 只写**最高规则**；
`.agent/rules/` 写**可机器裁决的规则细则**；
`.agent/harness/`、`scripts/`、`Makefile` 负责把规则变成**可执行门禁**。
