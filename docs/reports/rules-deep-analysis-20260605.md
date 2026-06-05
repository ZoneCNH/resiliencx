# `.agent/rules/` 深度分析报告（事实核对修订版）

> 修订日期: 2026-06-05  
> 分析范围: `.agent/rules/` 当前内容；仅修订本报告，不修改 `.agent/rules/` 规则源文件。  
> 核对命令: `find .agent/rules -maxdepth 1 -type f | sort`、`wc -l .agent/rules/*`、`python3 scripts/verify_rules.py`、`rg`。

分析范围：`.agent/rules/` 当前版本、与规则生成/校验强相关的 `README.md`、`.agent/registries/generated-artifacts.yaml`、`scripts/verify_rules.py`。

## 0. 本次修订结论

原报告中的部分判断已与当前仓库事实不符，需要纠正：

- **纠正**: `.agent/rules/registry.yaml` 当前存在，且包含 `total_rules: 419`、`p0_count: 119`、`p1_count: 300`、`active_count: 363`、`indexed_count: 56`；“registry.yaml 缺失”不是当前问题。
- **纠正**: `.agent/rules/README.md` 当前文件树使用现有文件名（如 `core-rules.md`、`schema-registry-rules.md`、`agent-runtime-rules.md`），未发现原报告所称 `00-index.md` / `01-core-rules.md` 数字前缀推荐段落；“命名体系不一致”不是当前可复现问题。
- **保留但降级**: `goal-rules.md` 与 `iron-rules.md` / `core-rules.md` 存在明显重复，但 `scripts/render_domain_rules.py` 明确允许域文件重复引用 `RULE-CORE-001..006` 作为锚点；因此它更像可维护性债务，不应直接定性为 P0/P1 的规则冲突。
- **保留**: `agent-runtime-rules.md` 的 `RULE-GOALCLI-EXIT-001` 退出码叙述仍与 `iron-rules.md` 标准退出码不一致；该规则元信息为 P1，但因影响 Gate/CI/Agent 串接语义，属于本报告中的最高修复优先级问题之一。
- **保留**: `.agent/registries/generated-artifacts.yaml` 将 `registry.yaml` 与三个机器渲染 Markdown 的 `generated_by` 都标为 `goalcli rules-verify`，但当前 README 与文件头显示实际生成入口分别是 `scripts/extract_rules.py` / `scripts/render_domain_rules.py`，属于生成物清单元信息漂移。
- **保留**: 三个机器渲染文件体量大、`.worktree/goal-patch.md` 源引用在当前 worktree 不存在、机器渲染文本仍残留 `07-worktree-rules.md` 历史路径、部分手写规则文件较薄、手写文件间交叉引用不足，均为真实但优先级不同的问题。
- **补充**: `registry.yaml` 不是只存在“文本重复”问题，而是已经在索引层显式记录了重复定义（`duplicate_at`）；这说明重复不是纯叙事噪音，而是可机器化证实的数据结构事实。
- **补充**: `iron-rules.md` 的“已知 P0 Gap”应被单独视为当前体系最高风险债务，因为它明确披露了 `Traceability Gate` 与 `Self-improving Gate` 仍未落地，且 `traceability-check` 尚未实现。

- “`.agent/rules/registry.yaml` 缺失 / 治理索引不存在”是错误结论。当前目录存在 `.agent/rules/registry.yaml`，且 `python3 scripts/verify_rules.py` 可读取并通过校验。
- “需要恢复 `00-index.md`、`01-core-rules.md` 等数字前缀文件名”是错误方向。当前 README 已声明真实树为 `README.md`、`registry.yaml`、`core-rules.md`、`schema-registry-rules.md`、`agent-runtime-rules.md` 等非数字前缀文件。

## 1. 当前文件清单与规模

`.agent/rules/` 当前包含 16 个 Markdown 规则/索引文件，合计 4818 行；另有 2 个 YAML 机器化数据文件，合计 3869 行。

### 1.1 Markdown 文件

| 文件 | 行数 | 类型 | 说明 |
| --- | ---: | --- | --- |
| `README.md` | 77 | 索引 | 权威顺序、生成流程、覆盖率说明 |
| `iron-rules.md` | 49 | **SSOT** | 七律、标准退出码、P0 Gap 说明 |
| `core-rules.md` | 1008 | 机器渲染 | Core/Context/State/SSOT/ID/Mode 等 |
| `schema-registry-rules.md` | 1118 | 机器渲染 | Schema/Registry/Goal Pack/Migration 等 |
| `agent-runtime-rules.md` | 1640 | 机器渲染 | Agent 运行时、goalcli、治理/度量等 |
| `goal-rules.md` | 452 | 手写 | Goal 对象模型全生命周期 |
| `worktree-rules.md` | 45 | 手写 | Worktree |
| `commit-rules.md` | 43 | 手写 | Commit |
| `pr-rules.md` | 43 | 手写 | PR |
| `evidence-rules.md` | 49 | 手写 | Evidence |
| `release-rules.md` | 60 | 手写 | Release |
| `harness-rules.md` | 57 | 手写 | Harness |
| `security-rules.md` | 39 | 手写 | Security |
| `issue-rules.md` | 36 | 手写 | Issue |
| `risk-decision-rules.md` | 51 | 手写 | Risk/Decision/Rollback |
| `self-improving-rules.md` | 51 | 手写 | Retrospective/Self-improving |

**规模分布**: 3 个机器渲染 Markdown 文件占 3766 行（约 78%）；11 个手写 Markdown 文件占 935 行（约 19%）；`README.md` 占 77 行（约 2%）。

### 1.2 YAML 文件

| 文件 | 行数 | 类型 | 当前事实 |
| --- | ---: | --- | --- |
| `registry.yaml` | 3798 | 机器化索引 | 存在；`total_rules: 419`、`p0_count: 119`、`p1_count: 300`、`active_count: 363`、`indexed_count: 56` |
| `enforcement-normalization.yaml` | 71 | 归一化数据 | 存在；用于 enforcement/规则机器化辅助 |

## 2. 真实问题与证据

### 2.1 [P1] `RULE-GOALCLI-EXIT-001` 退出码叙述与 `iron-rules.md` 标准退出码冲突

`iron-rules.md` 是当前权威顺序中的第一位，并明确给出 `goalcli` 与所有 Gate 命令统一遵守的标准退出码：

| 退出码 | `iron-rules.md` 含义 |
| ---: | --- |
| 0 | OK |
| 1 | 通用失败 |
| 2 | 参数错误 |
| 5 | worktree / main 违规 |
| 6 | schema 校验失败 |
| 7 | secret / 凭据泄漏 |
| 8 | Evidence 缺失或伪造 |
| 9 | Traceability 断链 |
| 10 | Release 不完整 |

但 `agent-runtime-rules.md` §84 的 `RULE-GOALCLI-EXIT-001` 文本列出另一套映射：

```text
0  = PASS
1  = GENERAL_FAILURE
2  = POLICY_VIOLATION
3  = SCHEMA_INVALID
4  = EVIDENCE_MISSING
5  = TRACEABILITY_BROKEN
6  = WORKTREE_INVALID
7  = SECRET_DETECTED
8  = RELEASE_BLOCKED
9  = NEEDS_HUMAN_APPROVAL
10 = INCONSISTENT_STATE
```

**冲突点**: 退出码 `2`、`5`、`6`、`8`、`9`、`10` 的语义与 `iron-rules.md` 不一致。例如 `iron-rules.md` 中 `5 = worktree / main 违规`，而 `RULE-GOALCLI-EXIT-001` 文本中 `5 = TRACEABILITY_BROKEN`。

**影响**: `agent-runtime-rules.md` 是机器渲染规则文件，读者或实现者可能按 §84 文本实现/判断退出码，导致 Makefile、Hooks、CI、Agent 串接语义与 SSOT 不一致。

**建议**:

1. 以 `iron-rules.md` 的“标准退出码”为唯一权威。
2. 修正 `RULE-GOALCLI-EXIT-001` 的源数据或 `scripts/render_domain_rules.py` 渲染输入，使其文本与 `iron-rules.md` 对齐。
3. 重跑 `python3 scripts/extract_rules.py` 与 `python3 scripts/render_domain_rules.py`，再运行 `python3 scripts/verify_rules.py` / `make rules-verify`。

**剩余风险**: `registry.yaml` 是大型生成索引，应继续用 `python3 scripts/verify_rules.py`、`make rules-verify` 或 `go run ./cmd/goalcli rules-consistency-check` 验证新鲜度与一致性。

### 2.2 [P1] `generated-artifacts.yaml` 的生成入口记录与实际脚本不一致

`.agent/registries/generated-artifacts.yaml` 当前把以下规则生成物的 `generated_by` 都记录为 `goalcli rules-verify`：

- `.agent/rules/registry.yaml`
- `.agent/rules/core-rules.md`
- `.agent/rules/schema-registry-rules.md`
- `.agent/rules/agent-runtime-rules.md`

但当前文件头与 README 显示：

- `.agent/rules/registry.yaml` 的头部写明重新生成命令是 `python3 scripts/extract_rules.py`。
- 三个机器渲染 Markdown 的头部写明由 `scripts/render_domain_rules.py` 从 `registry.yaml` 生成。
- `.agent/rules/README.md` 也把生成流程写为 `python3 scripts/extract_rules.py` 与 `python3 scripts/render_domain_rules.py`。

**真实风险**:

- `goalcli rules-verify` 更像校验入口，不是当前文本证据中的直接生成入口。
- 生成物清单若被 Agent、CI 或审计脚本当成“如何重建”的依据，可能导致只验证不重建。
- 后续规则变更时，生成链路和证明链路容易混淆。

**建议**:

1. 区分 `generated_by` 与 `verified_by`，或把生成入口改为当前实际脚本。
2. 在 README 中把“人改哪里 / 机器生成哪里 / 如何重建 / 如何验证”写成明确流程。
3. 修订后用 `rg` 固化 `generated_by`、`scripts/extract_rules.py`、`scripts/render_domain_rules.py`、`goalcli rules-verify` 的一致性证据。

### 2.3 [P1] `goal-rules.md` 与铁律/核心域规则存在大量重复，但需按“维护性债务”处理

`goal-rules.md` 的“第一性原理铁律”章节重复列出 `RULE-CORE-001` 到 `RULE-CORE-006`；同时还包含 ID、状态机、上下文恢复等与 `core-rules.md` 重叠的内容，例如：

- `RULE-ID-001`、`RULE-ID-002`
- `RULE-STATE-001`、`RULE-STATE-002`、`RULE-STATE-003`
- `RULE-CONTEXT-001`、`RULE-CONTEXT-002`、`RULE-CONTEXT-003`

**需修正原报告判断**: 这类重复不应直接等同为“SSOT 违规”。`scripts/render_domain_rules.py` 文件头说明“不重写 `iron-rules.md` 已覆盖的 7 条铁律, 但允许域文件中重复引用 `RULE-CORE-001..006` 作为锚点（`goal-rules.md` 中已有先例）”。这说明部分重复是现有设计承认的导航/锚点重复。

**真实风险**:

- 人工维护 `goal-rules.md` 时容易与机器渲染域文件产生叙述漂移。
- 读者可能分不清“锚点重复”与“独立权威定义”。
- `goal-rules.md` 452 行中相当一部分用于跨域汇总，导航价值与维护成本需要重新权衡。

**建议**:

1. 保留 `goal-rules.md` 的 Goal 生命周期导航定位。
2. 对重复章节增加明确注记：权威以 `iron-rules.md` / `registry.yaml` / 机器渲染域文件为准，此处为导航锚点。
3. 若后续重构，优先把重复正文压缩为链接，不直接删除独有内容。

### 2.4 [P1] 机器渲染文件体量过大，影响可读性与审阅粒度

三个机器渲染 Markdown 文件合计 3766 行：

| 文件 | 行数 | 规则范围 |
| --- | ---: | --- |
| `agent-runtime-rules.md` | 1640 | Agent 协议、运行时、goalcli、Lease/Heartbeat、治理/度量等 |
| `schema-registry-rules.md` | 1118 | Schema、Registry、Goal Pack、Migration 等 |
| `core-rules.md` | 1008 | Core、Context、State、SSOT、ID、Mode 等 |

**真实风险**:

- `agent-runtime-rules.md` 覆盖子域过多，单次审阅成本高。
- 机器渲染文件由 `scripts/render_domain_rules.py` 生成，拆分必须同步调整生成逻辑，不能手工拆文件。
- 大文件使局部变更 diff 不易审阅，但并不表示当前规则内容错误。

**建议**:

1. 保持当前文件不手工拆分。
2. 如需拆分，先调整 `scripts/render_domain_rules.py` 的分区规则和 README 文件树。
3. 候选拆分方向：Agent 协议、goalcli 命令契约、治理/度量/Doctor/Repair。
4. 对 generated Markdown 增加更醒目的头部约束：不要手改，改源数据后重新生成。

### 2.5 [P2] `.worktree/goal-patch.md` 源引用在当前 worktree 不存在，影响考古与再生成可解释性

`core-rules.md`、`schema-registry-rules.md`、`agent-runtime-rules.md` 的规则元信息包含 `source: §N Lxxxx`，README 也说明这些 derived artifacts 来自 `.worktree/goal-patch.md`。

当前 worktree 中未找到 `.worktree/goal-patch.md`。

**需修正原报告判断**: README 已明确 `.worktree/goal-patch.md` 是“历史推导，**仅供考古，不可作为依据**”。因此它缺失不直接削弱当前规则权威；当前权威仍是 `iron-rules.md` 与 `registry.yaml`。

**真实风险**:

- 后续重跑 `scripts/extract_rules.py` / `scripts/render_domain_rules.py` 时，若缺少源文档，生成链路可复现性不足。
- `source: §N Lxxxx` 的行号证据只能作为历史线索，不能作为当前审计证据。

**建议**:

1. README 中保留“仅供考古，不可作为依据”的提示。
2. 若项目仍要求再生成能力，应补齐源文件或在 README 中记录新的生成源位置。
3. 审计报告引用规则事实时，优先引用当前 `.agent/rules/*.md` 与 `registry.yaml`，不要依赖 `.worktree/goal-patch.md` 行号。

### 2.6 [P2] 机器渲染文本残留 `07-worktree-rules.md` 历史路径

当前 README、`iron-rules.md`、`goal-rules.md` 均使用现有文件名 `worktree-rules.md`，但机器渲染文本中仍出现历史路径：

- `core-rules.md`: `详见 .agent/rules/07-worktree-rules.md`
- `agent-runtime-rules.md`: 文件树片段仍列出 `07-worktree-rules.md`
- `schema-registry-rules.md`: 多处引用 `.agent/rules/07-worktree-rules.md`

**需修正原报告判断**: 这不支持“恢复数字前缀命名”的结论；它只说明生成文本中残留旧路径，应修生成源或渲染输入。

**真实风险**:

- 读者按旧路径跳转会失败。
- 机器渲染文件会把历史命名误传给后续 Agent 或审计脚本。
- 若生成源继续保留旧路径，重跑渲染会反复带回同一漂移。

**建议**:

1. 找到 `07-worktree-rules.md` 文本来源，优先修源数据或渲染输入。
2. 重跑 `python3 scripts/render_domain_rules.py` 并验证所有引用改为 `worktree-rules.md`。
3. 保留 `rg -n "07-worktree-rules|worktree-rules.md" .agent/rules .agent/registries/generated-artifacts.yaml` 作为回归证据。

### 2.7 [P2] 手写规则文件较薄，且交叉引用不足

多个手写文件只有 3-5 条规则、36-60 行，例如：

- `issue-rules.md`: 36 行
- `security-rules.md`: 39 行
- `commit-rules.md`: 43 行
- `pr-rules.md`: 43 行

**真实风险**:

- 文件粒度较细，导航时需要频繁跳转。
- 跨域关系不够显式，例如 PR、Evidence、Release、Harness、Security 之间存在实际依赖，但手写文件中的相互链接不充分。

**建议**:

1. 不建议仅因文件短而合并；“一个域一个文件”仍有清晰边界价值。
2. 优先增加“相关规则”链接，而不是合并文件。
3. 示例：`pr-rules.md` 链接 `evidence-rules.md` / `harness-rules.md`，`release-rules.md` 链接 `evidence-rules.md` / `risk-decision-rules.md`。

### 2.8 [P1] `registry.yaml` 的 `duplicate_at` 证明重复已进入索引层，不应只按“维护性债务”概括

`registry.yaml` 的字段定义已经明确 `duplicate_at` 语义是“该 id 在源文档中被二次定义的行号 (若有)”。这意味着仓库不仅存在规则正文重复，还存在由索引层直接记录的重复定位事实。

例如当前索引中至少存在如下重复标记：

- `RULE-CORE-004` 的 `duplicate_at` 指向 `3176`
- `RULE-CORE-006` 的 `duplicate_at` 指向 `3186`

**影响**: 这类重复不是单纯的叙事冗余，而是会影响规则索引、生成链路和机器审计的结构性事实。若报告只把它归为一般“维护性债务”，会低估其可机器验证的严重性。

**建议**:

1. 在结论中单列“索引级重复”。
2. 把 `duplicate_at` 的存在与 `goal-rules.md` 的锚点重复区分开。
3. 若后续修复规则源，优先修复源文档或提取器，避免重复继续回流到 `registry.yaml`。

### 2.9 [P1] `iron-rules.md` 的 `已知 P0 Gap` 应提升为当前体系的最高风险债务

`iron-rules.md` 已明确披露两类尚未落地的 P0 gap：

- `Traceability Gate`：要求 `goalcli traceability-check`，并按链路完整性返回退出码 9
- `Self-improving Gate`：要求 Retrospective / Patch 校验命令

这说明问题不是“未来可能需要补齐”，而是当前铁律文本已经承认存在未机器化落地的硬缺口。`traceability-check` 甚至在标准退出码表中被显式标为 `GAP`。

**影响**: 如果不把这部分放进摘要，报告会把一个公开的 P0 债务压低成普通背景信息，削弱深度分析的优先级判断。

**建议**:

1. 在结论中把 `Traceability Gate` 和 `Self-improving Gate` 作为当前最高风险债务之一。
2. 与 `RULE-GOALCLI-EXIT-001` 冲突一起看，因为两者都直接影响 Gate 语义和执行链路。
3. 后续修订报告时，优先把“已知 P0 Gap”单列，而不是只在退出码段落里顺带提及。

---

## 3. 已纠正的原报告错误结论

### 3.1 “registry.yaml 缺失”已证伪

当前 `.agent/rules/registry.yaml` 存在，`wc -l` 为 3798 行；解析得到：

```text
total_rules = 419
p0_count = 119
p1_count = 300
active_count = 363
indexed_count = 56
```

因此原报告中的 P0“确认 registry.yaml 状态 / 如缺失重新生成”应改为：

- **当前无需按缺失处理**。
- 后续如修改规则源，仍应运行 `python3 scripts/extract_rules.py`、`python3 scripts/render_domain_rules.py` 和 `python3 scripts/verify_rules.py`。

### 3.2 “文件命名不一致 / README 推荐数字前缀”未复现

当前 `.agent/rules/README.md` 文件树列出的就是现有命名：

- `core-rules.md`
- `schema-registry-rules.md`
- `agent-runtime-rules.md`
- `goal-rules.md`
- 以及各手写 `*-rules.md`

未在当前 README 中发现原报告所称数字前缀结构。因此原 P2“统一命名体系”建议应删除，避免引入不必要的大规模重命名风险。

补充说明：机器渲染内容中仍可见少量数字前缀或历史路径残留，尤其是 `07-worktree-rules.md`；这只构成第 2.6 节中的 P2 叙述漂移/路径修复问题，不构成重命名当前规则文件的依据。

---

## 4. 修订后优先级排序

| 优先级 | 动作 | 影响 | 工作量 | 状态 |
| --- | --- | --- | --- | --- |
| P1 | 修正 `RULE-GOALCLI-EXIT-001` 退出码叙述，使其与 `iron-rules.md` 对齐 | 消除实现/CI/Agent 退出码歧义 | 低-中；需改源或渲染链路后重跑 | 真实问题 |
| P1 | 修正 `generated-artifacts.yaml` 的生成入口元信息，区分 `generated_by` 与 `verified_by` | 防止只验证不重建，避免 Agent/CI 误读生成链路 | 低-中；需同步 README 与脚本证据 | 真实问题 |
| P1 | 为 `goal-rules.md` 重复章节增加“导航锚点，非独立权威”说明，或压缩为链接 | 降低 SSOT 漂移风险 | 中；只应在规则文档任务中修改 | 真实问题 |
| P1 | 评估机器渲染文件拆分，并先改 `scripts/render_domain_rules.py` 分区逻辑 | 提升审阅性 | 中-高；涉及生成链路 | 真实问题 |
| P2 | 修正机器渲染文本中的 `07-worktree-rules.md` 历史路径残留 | 避免导航失败和命名误导 | 低-中；需改源或渲染输入后重跑 | 真实问题 |
| P2 | 处理 `.worktree/goal-patch.md` 源引用缺失的可复现性说明 | 提升考古/再生成可解释性 | 低-中 | 真实问题 |
| P2 | 为手写规则文件增加跨域“相关规则”链接 | 提升可发现性 | 低 | 真实问题 |
| 删除 | “registry.yaml 缺失” | 当前事实不成立 | — | 已纠正 |
| 删除 | “采用数字前缀重命名规则文件” | 当前 README 未支持，且风险高 | — | 已纠正 |

---

## 5. 验证记录

本报告修订基于以下命令证据：

```bash
find .agent/rules -maxdepth 1 -type f | sort
wc -l .agent/rules/* docs/reports/rules-deep-analysis-20260605.md
python3 - <<'PY'
from pathlib import Path
import yaml
p = Path('.agent/rules/registry.yaml')
data = yaml.safe_load(p.read_text())
print(p.exists())
print(data['total_rules'], data['p0_count'], data['p1_count'], data['active_count'], data['indexed_count'])
PY
rg -n "RULE-GOALCLI-EXIT-001|退出码|POLICY_VIOLATION|WORKTREE_INVALID|SECRET_DETECTED" .agent/rules/agent-runtime-rules.md -C 4
rg -n "标准退出码|worktree / main|schema 校验失败|Traceability 断链|Release 不完整" .agent/rules/iron-rules.md -C 2
rg -n "registry.yaml|419|active|00-index|01-core|权威顺序" .agent/rules/README.md -C 2
rg -n "generated_by|goalcli rules-verify|scripts/extract_rules.py|scripts/render_domain_rules.py" .agent/rules .agent/registries/generated-artifacts.yaml -C 2
rg -n "07-worktree-rules|worktree-rules.md" .agent/rules .agent/registries/generated-artifacts.yaml
python3 scripts/verify_rules.py
```

---

## 6. 结论

`.agent/rules/` 当前规则体系并非缺少 `registry.yaml`，也不需要按数字前缀重命名。当前最高优先级真实问题是两个 P1：`RULE-GOALCLI-EXIT-001` 退出码叙述与 `iron-rules.md` 冲突，以及 `generated-artifacts.yaml` 生成入口元信息与实际脚本不一致；`07-worktree-rules.md` 属于 P2 历史路径残留。其余问题主要属于维护性、可读性、可复现性和导航优化。

建议优先修正退出码冲突和 generated artifacts 清单元信息，再处理 `goal-rules.md` 重复说明、旧路径残留与机器渲染文件拆分策略；不要基于已证伪的“registry.yaml 缺失”或“数字前缀命名要求”发起变更。

---

_报告结束_
