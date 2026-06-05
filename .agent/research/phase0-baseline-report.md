# Phase 0 基线分析报告

> Issue #59 — .agent 分类体系重构
> 分析日期: 2026-06-04
> 分析员: worker-1

---

## 1. 文件清单与统计

### 1.1 总览

| 指标 | 值 |
|------|-----|
| `.agent` 目录文件总数（磁盘） | 134 |
| `index.yaml` 覆盖文件数 | 133 |
| 未覆盖文件数 | 1 |
| 覆盖率 | 99.3% |

**唯一未覆盖文件:** `.agent/rules/enforcement-normalization.yaml`（新增文件，尚未注册）

### 1.2 目录分布

| 目录 | 文件数 |
|------|--------|
| `.agent/policies/` | 21 |
| `.agent/rules/` | 19 |
| `.agent/docs/` | 16 |
| `.agent/registries/` | 15 |
| `.agent/harness/` | 14 |
| `.agent/evidence/` | 12 |
| `.agent/contracts/` | 12 |
| `.agent/traceability/` | 6 |
| `.agent/runtime/` | 6 |
| `.agent/archive/` | 5 |
| `.agent/release/` | 4 |
| `.agent/index.yaml` | 1 |
| `.agent/research/` | 1 |

---

## 2. Layer / Authority / Mutability 分布

### 2.1 Layer 分布

| Layer | 文件数 | 占比 |
|-------|--------|------|
| policy | 34 | 25.6% |
| machine_contract | 28 | 21.1% |
| documentation | 25 | 18.8% |
| registry | 16 | 12.0% |
| evidence | 12 | 9.0% |
| runtime_contract | 7 | 5.3% |
| traceability | 6 | 4.5% |
| archive | 5 | 3.8% |

### 2.2 Authority 分布

| Authority | 文件数 | 占比 |
|-----------|--------|------|
| source_of_truth | 112 | 84.2% |
| validated_mirror | 12 | 9.0% |
| historical_snapshot | 9 | 6.8% |

### 2.3 Mutability 分布

| Mutability | 文件数 | 占比 |
|------------|--------|------|
| hand_written | 115 | 86.5% |
| append_only | 13 | 9.8% |
| generated | 5 | 3.8% |

### 2.4 Owner 分布

| Owner | 文件数 | 占比 |
|-------|--------|------|
| governance | 69 | 51.9% |
| release | 23 | 17.3% |
| gate-runtime | 21 | 15.8% |
| runtime | 6 | 4.5% |
| github | 5 | 3.8% |
| downstream | 4 | 3.0% |
| ci | 3 | 2.3% |
| security | 2 | 1.5% |

---

## 3. 基线检查结果

### 3.1 编译检查

```
GOWORK=off go build ./cmd/goalcli → ✅ 通过（exit 0）
```

### 3.2 治理检查

| 命令 | 状态 | 详情 |
|------|------|------|
| `command-registry` | ❌ 失败 | 38 个 active rule 缺少 `enforced_by` |
| `issue-registry` | ✅ 通过 | entries are implemented, unique, and contiguous |
| `makefile-baseline` | ✅ 通过 | registry contract satisfied |
| `rules-verify` | ❌ 崩溃 | `enforced_by.split()` — 期望 string，实际收到 dict |
| `governance-check` | ❌ 失败 | `unknown command` — CLI 未注册此子命令 |

### 3.3 关键问题详情

#### 问题 A: `command-registry` — 38 个 active rule 缺少 enforced_by

受影响的规则 ID（部分）：
- `RULE-TASK-001` ~ `RULE-TASK-004`
- `RULE-TEMPLATE-001`, `RULE-TEMPLATE-002`
- `RULE-VERSION-001`, `RULE-VERSION-002`
- `RULE-VIOLATION-FIXTURE-001`, `RULE-VIOLATION-FIXTURE-002`
- `RULE-WT-GC-001`, `RULE-WT-GC-002`
- `RULE-XGO-001`
- `RULE-XSTACK-001` ~ `RULE-XSTACK-003`
- `RULE-XSTACK-ADMISSION-001`, `RULE-XSTACK-ADMISSION-002`

**根因:** `.agent/rules/registry.yaml` 中这些规则标记为 `status: active` 但缺少 `enforced_by` 字段，与 schema 定义矛盾。

#### 问题 B: `rules-verify` — enforced_by 类型不一致

```python
AttributeError: 'dict' object has no attribute 'split'
```

`scripts/verify_rules.py` 第 59 行执行 `enforced_by.split()`，但 `registry.yaml` 中部分规则的 `enforced_by` 字段使用了 dict 结构 `{command, args, context}` 而非字符串。代码与 schema 不匹配。

#### 问题 C: `governance-check` — CLI 未注册

`go run ./cmd/goalcli governance-check` 返回 `unknown command "governance-check"`。该命令在 `command-registry.yaml` 中未注册，但在 `harness.yaml` 的 `required_gates` 中被引用。

---

## 4. 核心文件分析

### 4.1 `index.yaml` 结构

- **schema_version:** 1.0
- **模块:** resiliencx
- **SSOT 边界定义:** 6 个（command-registry, harness, rules/registry, issue-registry, release-required-gates, generated-artifacts）
- **物理迁移标记:** `physical_migration: true`
- **文件条目:** 133 条

### 4.2 `command-registry.yaml`

- **schema_version:** 2.9.3
- **总命令数:** 91
- **Phase 分布:**
  - P0: 47 命令
  - CTX: 10 命令
  - P1: 22 命令
  - P2: 12 命令

### 4.3 `harness.yaml`

- **schema_version:** 3.1
- **必需 gate 数:** 26（required_gates）
- **扩展 gate 数:** 6（extended_gates）
- **最终 gate 数:** 6（final_gates）
- **证据路径:** 5 个（manifest, checksum, debt_manifest, debt_markdown, debt_checksum）
- **MVA gate 数:** 6（G12-G16）

### 4.4 `rules/registry.yaml`

- **version:** 1
- **总规则数:** 419
- **P0 规则:** 119
- **P1 规则:** 300
- **active 规则:** 363
- **indexed 规则:** 56
- **enforced_by 结构:** 混合使用 string 和 dict 格式（不一致）

### 4.5 `generated-artifacts.yaml`

- **schema_version:** 1.0
- **生成 artifact 数:** 9
- **validated_mirror 数:** 8
- **documentation_mirror 数:** 2

---

## 5. 发现的问题与不一致

### 5.1 高优先级

| # | 问题 | 影响 | 建议 |
|---|------|------|------|
| 1 | `rules-verify` 崩溃（enforced_by 类型不一致） | 规则验证流水线完全不可用 | 统一 enforced_by 为 dict 结构，修复 verify_rules.py |
| 2 | 38 个 active rule 缺少 enforced_by | command-registry 检查失败 | 将这些规则降级为 `indexed` 或补充 enforced_by |
| 3 | `governance-check` CLI 子命令未注册 | 无法通过 goalcli 执行治理检查 | 在 command-registry.yaml 中注册，或确认已被其他命令替代 |

### 5.2 中优先级

| # | 问题 | 影响 | 建议 |
|---|------|------|------|
| 4 | `enforcement-normalization.yaml` 未在 index.yaml 注册 | 新文件脱离控制面 | Phase 1 中注册 |
| 5 | `rules/registry.yaml` 标记为 `generated` 但实际手写 | authority 语义混乱 | 重新分类 mutability |
| 6 | `governance` owner 负载过重（51.9%） | ownership 粒度不足 | 考虑细分 owner 角色 |

### 5.3 低优先级

| # | 问题 | 影响 | 建议 |
|---|------|------|------|
| 7 | `physical-migration-manifest.yaml` 存在但被 git 删除 | Phase 3/4 迁移清单可能过期 | 确认迁移状态 |
| 8 | documentation layer 文件过多（25 个） | 文档膨胀风险 | 合并或归档冗余文档 |

---

## 6. Phase 1 建议

1. **修复 enforced_by 一致性:** 统一 `rules/registry.yaml` 中所有 enforced_by 为 dict 格式 `{command, args, context}`，修复 `scripts/verify_rules.py` 的类型断言。

2. **补全 command-registry 注册:** 将 `governance-check` 等缺失命令注册到 `command-registry.yaml`，或将引用它的 gate 重定向到已有命令。

3. **注册新文件:** 将 `.agent/rules/enforcement-normalization.yaml` 加入 `index.yaml`。

4. **规则状态清理:** 将 38 个缺少 enforced_by 的 active rule 降级为 `indexed`，或为它们补充 enforced_by。

5. **index.yaml 分类审查:** 基于 layer/authority/mutability 分布，评估是否需要引入新的分类维度（如 `criticality` 或 `domain`）来支持 Issue #59 的重构目标。

---

## 附录: 基线快照

```
分析时间: 2026-06-04
Git commit: (working tree, dirty)
Branch: main
Build: ✅ 通过
command-registry: ❌ 38 rules missing enforced_by
issue-registry: ✅ 通过
makefile-baseline: ✅ 通过
rules-verify: ❌ 崩溃
governance-check: ❌ 未注册
```
