---
claude_adapter_version: v1.0.0
status: normative
scope:
  - Claude Code
  - local repository execution
owner: ZoneCNH
inherits:
  - ./CONSTITUTION.md
  - ./AGENTS.md
runtime_control_plane: ./.agent
---

# CLAUDE.md

## 0. Purpose

本文件是 Claude Code 在本仓库中的专用执行适配器。

本文件只定义 Claude Code 的本地工作方式，不重复 `CONSTITUTION.md` 与 `AGENTS.md` 的全部规则。

优先级：

```text
CONSTITUTION.md
> AGENTS.md
> CLAUDE.md
> README.md
> Issue / PR / temporary notes
```

如果本文件与 `CONSTITUTION.md` 或 `AGENTS.md` 冲突，必须以上层文件为准。

---

## 1. Claude Code Role

Claude Code 在本仓库中必须作为工程执行 Agent，而不是普通代码补全工具。

Claude 必须：

* 先恢复上下文，再修改文件
* 先判断任务模式，再制定计划
* 先保护分层边界，再实现功能
* 先验证，再声明完成
* 先生成 Evidence，再输出 DONE
* 遇到未知问题时触发 AutoResearch 或明确标记不确定性

---

## 2. Startup Protocol

每次进入仓库后，Claude 必须先执行或确认以下检查。

### 2.1 Repository Check

```bash
pwd
git status --short
git branch --show-current
git rev-parse --show-toplevel
```

必须确认：

* 当前是否在正确仓库
* 当前是否在 worktree
* 当前是否不在 `main`
* 工作区是否已有未提交改动
* 未提交改动是否来自用户或其他 Agent

### 2.2 Main Branch Protection

如果当前分支是 `main`，Claude 禁止直接修改文件。

必须提示或执行 worktree 创建流程：

```bash
git fetch origin
git worktree list
git worktree add ../<repo>-wt-<goal-or-issue> -b work/<goal-or-issue> origin/main
```

进入 worktree 后才能修改。

### 2.3 Context Loading

Claude 必须按顺序读取：

```text
CONSTITUTION.md
AGENTS.md
.agent/INDEX.md
.agent/context/
.agent/rules/
.agent/harness/
当前 Goal / Issue / Task
相关 contracts/
相关 docs/architecture/
相关代码文件
```

禁止只读 README 后直接实现。

---

## 3. Task Classification

Claude 必须先判断任务属于哪类。

### Lite Task

适用于：

* 小文档修正
* 小脚本修正
* 单点 bugfix
* 明确的测试补充

允许简化流程，但仍需要 Evidence。

### Standard Task

适用于：

* 普通功能
* 普通 Issue
* 局部重构
* 新增 Gate
* 修改测试体系
* 修改公共行为

必须有：

* Goal
* Plan
* Task
* Test
* Evidence

### Full Task

适用于：

* 架构调整
* 标准变更
* 分层变更
* Release
* 下游采纳
* 公共 API 变更
* 大型迁移

必须有：

* Goal
* Spec
* Design
* ADR
* Plan
* Tasks
* Tests
* Evidence
* Review
* Release
* Retrospective

---

## 4. Editing Protocol

Claude 修改文件时必须遵守：

* 使用最小必要 diff
* 不做无关格式化
* 不重排无关代码
* 不批量改名，除非任务明确要求
* 不移动公共 API，除非有 Design / ADR
* 不创建新的上帝模块
* 不把业务逻辑写入 L0/L1/L2
* 不绕过 contracts
* 不引入隐藏全局状态
* 不硬编码 secret、路径、环境依赖
* 不删除测试来让 CI 通过
* 不降低 Harness 严格度来通过 Gate

---

## 5. Search and Inspection Rules

Claude 应优先使用快速、可审查的本地搜索。

推荐命令：

```bash
rg "<keyword>"
rg --files
find . -maxdepth 3 -type f
git diff --stat
git diff
git log --oneline -20
```

禁止：

* 盲目全仓重写
* 未理解依赖关系就修改接口
* 未检查调用点就修改 public function
* 未检查 tests 就修改核心逻辑
* 未检查 `.agent/rules/` 就修改治理规则

---

## 6. Planning Rules

Claude 对非 Lite 任务必须先给出简短 Plan。

Plan 至少包含：

```text
Goal:
Mode:
Scope:
Files likely affected:
Risks:
Validation:
Evidence:
```

当用户明确要求直接执行时，Claude 可以压缩 Plan，但不得跳过验证和 Evidence。

---

## 7. Layer Boundary Rules

Claude 必须保护以下分层：

```text
resiliencx
    ↓
L0: kernel
    ↓
L1: configx / observex / testkitx / resiliencx / schedulex
    ↓
L2: redisx / kafkax / postgresx / taosx / ossx / clickhousex / natsx
    ↓
L3+: business systems
```

### 禁止行为

* L0 依赖 L1/L2/L3
* L1 依赖 L2/L3
* L2 依赖其他 L2
* L2 依赖 L3
* L3 依赖上游 internal
* 下游复制上游内部实现
* 公共 contracts 绕过版本管理
* infra adapter 引入业务领域逻辑

### 修改分层相关文件时

必须同步：

* `docs/architecture/`
* `.agent/rules/`
* `.agent/harness/`
* boundary tests
* Evidence

---

## 8. Command Policy

Claude 可以优先运行只读命令：

```bash
pwd
ls
find
rg
git status
git diff
git log
go test ./...
go test ./... -run <TestName>
make check
make test
make lint
```

涉及破坏性命令时必须极度谨慎。

高风险命令包括：

```bash
rm -rf
git reset --hard
git clean -fd
git push --force
git rebase
git checkout -- .
docker system prune
```

未经明确授权，不得执行高风险命令。

---

## 9. Testing Protocol

Claude 修改代码后必须尽量运行相关测试。

推荐优先级：

```bash
make check
make lint
make test
make boundary-check
make worktree-check
make evidence-check
make harness-check
```

Go 项目可使用：

```bash
go test ./...
go test ./... -race
go vet ./...
gofmt -w <changed-files>
```

无法运行测试时，Claude 必须说明：

* 未运行哪些测试
* 原因
* 风险
* 用户可手动执行的命令

禁止写：

```text
测试应该可以通过
```

必须写：

```text
未运行测试，原因是...
```

或：

```text
已运行测试：...
结果：...
```

---

## 10. Evidence Protocol

Claude 不能只输出"完成了"。

完成前必须生成或说明 Evidence。

Evidence 推荐格式：

```text
Evidence:
- Commands:
- Output summary:
- Changed files:
- Tests:
- Risks:
- Remaining work:
```

可落盘路径：

```text
.agent/runs/<run-id>/
docs/evidence/<goal-id>/
release/manifest/<release-id>/
```

最终完成声明必须使用：

```text
DONE with evidence:
```

---

## 11. Git Discipline

Claude 必须保护用户工作区。

在修改前检查：

```bash
git status --short
```

如果存在未提交变更，必须判断：

* 是否为本次任务相关
* 是否可能是用户改动
* 是否需要避免覆盖

禁止覆盖用户未提交改动。

提交前应检查：

```bash
git diff --stat
git diff
git status --short
```

Commit 推荐格式：

```text
<type>(<scope>): <summary>

Goal: GOAL-YYYYMMDD-NNN
Task: TASK-<goal-id>-NNN
Evidence: EVID-<task-id>-YYYYMMDD-NNN
```

---

## 12. Pull Request Protocol

Claude 创建或准备 PR 时，PR 描述必须包含：

```text
## Summary

## Goal / Issue

## Changes

## Validation

## Evidence

## Risk

## Rollback

## Downstream Impact

## Checklist
- [ ] Not on main
- [ ] Worktree used
- [ ] Tests passed
- [ ] Evidence attached
- [ ] No secrets exposed
- [ ] Layer boundaries respected
- [ ] Docs/contracts updated if needed
```

---

## 13. Documentation Protocol

Claude 修改文档时必须保证：

* 与 `CONSTITUTION.md` 不冲突
* 与 `AGENTS.md` 不冲突
* 文档不是孤立规则，关键规则应进入 `.agent/rules/`
* 可执行规则应进入 `.agent/harness/` 或 `scripts/`
* 结构性决策应进入 ADR
* Release 影响应进入 Release Manifest

禁止把所有规则都堆进 README。

---

## 14. AutoResearch Protocol

Claude 遇到以下情况必须停止猜测，转入 AutoResearch 或明确标记不确定性：

* API 行为不确定
* 依赖版本不确定
* 外部工具行为可能变化
* Issue 要求不完整
* 文档与代码冲突
* 测试失败原因不清楚
* Release 风险不明确
* 安全影响不明确

AutoResearch 输出格式：

```text
Question:
Known facts:
Unknowns:
Research result:
Confidence:
Decision:
Impact:
Follow-up:
```

---

## 15. Self-improving Protocol

当 Claude 发现以下情况时，应建议或生成补丁：

* 重复人工操作
* 规则逃逸
* Harness 漏检
* CI 缺口
* 文档与代码反复不一致
* 分层违规未被机器发现
* Evidence 缺失但流程允许通过
* Agent 容易误执行

补丁类型：

```text
PATCH-PROMPT-YYYYMMDD-NNN
PATCH-HARNESS-YYYYMMDD-NNN
PATCH-RULE-YYYYMMDD-NNN
```

---

## 16. Final Response Format

Claude 完成任务后，最终输出应包含：

```text
DONE with evidence:

Summary:
- ...

Changed files:
- ...

Validation:
- ...

Evidence:
- ...

Risks:
- ...

Follow-up:
- ...
```

如果未完成，输出：

```text
NOT DONE:

Current state:
- ...

Blocked by:
- ...

Verified facts:
- ...

Remaining work:
- ...

Recommended next step:
- ...
```

---

## 17. Final Rule

Claude Code 的核心目标不是快速改文件，而是让每次修改都满足：

```text
Context-aware
→ Layer-safe
→ Evidence-backed
→ Harness-verified
→ Reviewable
→ Self-improving
```

任何不能被验证的完成，都不是完成。
