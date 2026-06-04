# 当前项目深度结构分析报告

生成日期：2026-06-04

审计对象：`github.com/ZoneCNH/xlib-standard`

审计版本：`8ebb547`

审计方式：静态结构审计、权威文档复核、核心 gate 抽样验证、历史分析差异复核。

## 结论

当前项目综合工程评分为 **8.6/10**。

这个分数高于普通基础库项目，原因是核心库边界清晰、Go module 无外部依赖、测试覆盖和治理 gate 很强；但它没有达到 9 分以上，主要因为治理运行时、发布门禁、文档检查和 downstream 证据被压缩到一个很大的结构面里，长期维护成本和误判风险仍然明显。

需要区分两个分数：

| 分数类型 | 当前结果 | 含义 |
| --- | ---: | --- |
| `goalcli score --min 9.8` | 10.0/10 | 发布治理完整性分，已通过机器 gate。 |
| 本报告综合工程评分 | 8.6/10 | 代码、测试、架构、治理、downstream、维护成本的综合判断。 |
| 结构健康分 | 7.4/10 | 只看模块边界、复杂度、gate 可维护性和证据质量后的判断。 |

`goalcli score` 的 10 分不能直接等价为项目整体 10 分。`internal/releasequality/score.go:34` 定义的维度主要是文件存在、文档 needle、score gate、release evidence 约束等发布治理项；`docs/scorecard.md` 也说明它不能替代 `make ci`、security、release-final-check、race、vuln scan、secret scan、integration 和 release evidence。

## 评分模型

| 维度 | 权重 | 分数 | 加权分 | 判断 |
| --- | ---: | ---: | ---: | --- |
| 公共 API 与模板核心 | 15% | 9.1 | 1.37 | `pkg/templatex` 职责集中，标准库依赖，模板目标明确。 |
| 测试与回归保护 | 15% | 9.0 | 1.35 | 当前 `go test ./...` 通过，测试行数高于生产 Go 代码行数。 |
| 治理与发布 gate | 20% | 9.4 | 1.88 | `score`、registry、context gate 当前通过，release 流程完整。 |
| 文档与标准一致性 | 10% | 8.8 | 0.88 | 文档体系完整，但依赖大量固定文本检查。 |
| 安全与供应链 | 10% | 9.2 | 0.92 | secrets、govulncheck、release evidence 约束清楚；本轮未跑完整 security。 |
| Generator 与 downstream 验证 | 10% | 7.8 | 0.78 | integration 覆盖代表样本，但 downstream matrix 仍显示未采纳状态。 |
| 可维护性与结构复杂度 | 15% | 7.1 | 1.07 | `goalcli`、`.agent`、文档 gate 的复杂度集中。 |
| 反馈成本与运行复杂度 | 5% | 7.3 | 0.37 | release profile 完整但昂贵，本地迭代心智负担较高。 |
| 合计 | 100% | 8.65 | 8.65 | 四舍五入为 **8.6/10**。 |

## 已验证事实

本轮实际执行并通过的命令：

- `GOWORK=off GOCACHE=/tmp/xlib-analysis-gocache go test ./...`
- `GOWORK=off GOCACHE=/tmp/xlib-analysis-gocache go run ./cmd/goalcli score --min 9.8`
- `GOWORK=off GOCACHE=/tmp/xlib-analysis-gocache go run ./cmd/goalcli issue-registry`
- `GOWORK=off GOCACHE=/tmp/xlib-analysis-gocache go run ./cmd/goalcli agent-team-contract --dry-run --verify`
- `GOWORK=off GOCACHE=/tmp/xlib-analysis-gocache go run ./cmd/goalcli context-profile-check`

当前结构度量：

| 项 | 数值 |
| --- | ---: |
| `.agent` 文件数 | 130 |
| `.agent` 总大小 | 553088 bytes |
| `docs` 文件数 | 60 |
| `docs` 总大小 | 323842 bytes |
| 非测试 Go 代码行数 | 11462 |
| 测试 Go 代码行数 | 14326 |
| `cmd/goalcli/governance.go` 行数 | 1569 |
| `cmd/goalcli/main_test.go` 行数 | 2060 |
| `scripts/check_docs.sh` 行数 | 470 |

## 当前强项

### 1. 核心库边界干净

`go.mod` 只有 module path 和 Go 版本，没有外部 Go module 依赖。这对基础库标准项目是重要优势：供应链面小，模板渲染后的 downstream 起点也更可控。

### 2. 测试保护强

本轮 `go test ./...` 全量通过，测试代码 14326 行，高于非测试 Go 代码 11462 行。对于一个标准库模板和治理工具混合仓库，这说明回归保护投入充分。

### 3. 发布治理证据完整

`goalcli score --min 9.8` 返回 10.0 且通过；`issue-registry`、`agent-team-contract --dry-run --verify`、`context-profile-check` 均通过。历史报告中提到的 Issue Registry 硬编码计数问题已经修复为动态解析；planned command 空文件和 JSON 基础校验也已经补上。

### 4. release profile 有明确 DAG

`Makefile:272` 到 `Makefile:323` 将 `context-lite`、`context-standard`、`context-full`、`context-release`、`release-check` 和 `release-final-check` 串起来，并要求 `GOWORK=off`。这比散落脚本更容易形成发布证据链。

### 5. downstream integration 不是空检查

`scripts/run_integration.sh:7` 到 `scripts/run_integration.sh:44` 会渲染代表 module，执行 `go mod tidy`、`go test ./...`、contracts、boundary、standard-impact、debt、evidence 和 release-evidence-check。虽然覆盖范围有限，但不是简单的文件存在检查。

## 结构性问题排序

### P1. `cmd/goalcli` 仍是治理单体和路由中枢

严重度：高

证据：

- `cmd/goalcli/main.go:20` 到 `cmd/goalcli/main.go:130` 是大型 switch，集中路由版本、治理、文档、security、release、planned command 和外部脚本。
- `cmd/goalcli/governance.go` 当前 1569 行，同时承载 registry、context profile、planned command、版本常量和多类治理检查。
- `cmd/goalcli/main_test.go` 当前 2060 行，说明测试也跟随单体入口膨胀。

影响：

治理职责被集中在少数文件里，新 gate 很容易继续追加到 switch 和大文件中。短期可控，长期会增加改动冲突、测试定位成本和 gate 语义漂移风险。

建议：

把 `goalcli` 拆为声明式 command registry 加 typed runner。每个 command 至少声明 `name`、`category`、`runner_kind`、`accepted_flags`、`required_artifacts`、`validator` 和 `evidence_contract`。主入口只做解析和 dispatch，不继续承载治理语义。

### P2. planned command 已修补基础校验，但语义覆盖仍不均匀

严重度：高

证据：

- `cmd/goalcli/governance.go:860` 到 `cmd/goalcli/governance.go:901` 定义了大量 planned command 到文件的映射。
- `cmd/goalcli/governance.go:903` 到 `cmd/goalcli/governance.go:937` 只为部分命令定义 semantic markers。
- `cmd/goalcli/governance.go:939` 到 `cmd/goalcli/governance.go:1017` 已经拒绝缺失文件、目录、空文件、无效 JSON 和缺失 marker，这是历史问题的改进。

影响：

当前实现已经不是纯占位检查，但很多 planned command 仍主要证明“有文件、有少量标记”，不能证明该 gate 的业务语义真的被执行。`--dry-run --verify` 容易被误读为完整行为验证。

建议：

按 command 增加 typed validator。对 YAML/JSON 工件读取结构化字段，验证 schema version、必需节点、枚举值、cross-reference 和对应 Makefile/contract 是否一致。对尚未具备真实执行能力的 command，输出状态应显式区分 `manifest_verified`、`semantic_verified` 和 `behavior_verified`。

### P3. 多个关键 gate 仍依赖字符串 needle 和手写解析

严重度：中高

证据：

- `internal/releasequality/score.go:34` 到 `internal/releasequality/score.go:44` 的 score 维度主要是文件存在和文本包含。
- `internal/releasequality/score.go:87` 到 `internal/releasequality/score.go:103` 的 `textDimension` 使用 `strings.Contains`。
- `scripts/check_docs.sh:43` 到 `scripts/check_docs.sh:51` 的 `require_text` 是固定文本检查，并被大量复用。
- `cmd/goalcli/governance.go` 中 context profile 和 registry 校验包含手写解析逻辑。

影响：

needle 检查适合防止关键文字被误删，但不适合证明结构关系正确。它会带来两类风险：文本存在但语义错误的假阳性，以及措辞调整导致 gate 失败的假阴性。

建议：

保留少量关键 needle 作为兼容 guard，同时把 schema、manifest、registry、profile、release evidence 迁移到结构化解析。文档类 gate 可改为“元数据块 + 链接/标题/命令结构校验”，减少对正文措辞的绑定。

### P4. Go 与 shell gate 边界仍然分裂

严重度：中

证据：

- `cmd/goalcli/main.go:86` 到 `cmd/goalcli/main.go:128` 将 boundary、contracts、dependency-check、docs-check、integration、release evidence、rules、security、standard-impact 等委托给外部脚本或工具。
- `runExternal` 当前主要传递进程退出码和输出，不聚合结构化失败原因。

影响：

外部脚本本身不是问题，问题在于 evidence contract 分散。失败时，调用者很难统一知道失败属于依赖缺失、输入缺失、语义不一致、环境问题还是真实质量问题。

建议：

为脚本输出定义最小 JSON report：`command`、`status`、`category`、`checked_files`、`gaps`、`warnings`、`duration_ms`。`goalcli` 保留 shell 执行能力，但统一吸收结构化结果。

### P5. `.agent` 和文档面过大，认知负担高

严重度：中

证据：

- `.agent` 当前 130 个文件，约 553 KB。
- `docs` 当前 60 个文件，约 324 KB。
- 多个 `.agent` YAML 是小型 marker 或单一契约文件。

影响：

治理资料很完整，但维护者需要同时理解大量文件、Makefile profile、command registry、release evidence 和标准文档。随着版本推进，漂移风险会上升，尤其是“文档更新了但 registry/gate 未更新”或“gate 更新了但标准文档未同步”。

建议：

把高频变动的 `.agent` 小文件合并为少数 schema-backed pack，例如 `governance-pack.yaml`、`runtime-pack.yaml`、`release-pack.yaml`。保留对外文档的可读性，但让机器 gate 读取更少、更稳定的结构化入口。

### P6. release/context profile 完整但运行链路昂贵

严重度：中

证据：

- `Makefile:272` 到 `Makefile:286` 定义 context profile 层级。
- `Makefile:317` 到 `Makefile:323` 的 `release-final-check` 进入 `context-release`，再执行 evidence checksum、score、严格 release evidence 检查。
- 本轮 `context-profile-check` 已通过，说明自递归和 profile DAG 的历史风险已被 gate 覆盖。

影响：

完整 release gate 是优势，但本地反馈成本高，容易导致开发者只跑局部 gate。只要文档没有明确每个 profile 的目标、耗时和适用场景，使用者就容易误选过轻或过重的检查。

建议：

为每个 profile 固定输出预期：目标、适用阶段、覆盖 gate、平均耗时、必须前置条件、不能证明什么。让 `goalcli context-profile --profile ...` 输出同一份机器可读说明。

### P7. downstream 真实采纳仍是主要外部风险

严重度：中

证据：

- `docs/downstream-matrix.md:9` 到 `docs/downstream-matrix.md:20` 登记了 10 个 standard target libraries，当前均为 `not_adopted` / `not_run`。
- `docs/downstream-matrix.md:25` 说明完整 release Evidence 应覆盖矩阵目标库或记录未覆盖原因。
- `scripts/run_integration.sh:7` 到 `scripts/run_integration.sh:10` 当前只覆盖 `kernel`、`configx`、`redisx` 代表样本。

影响：

模板自测可以证明“生成样本可跑”，不能证明全部目标库的真实迁移成本、外部依赖组合、owner 接受度和长期升级路径。

建议：

将 downstream 分为 rings：R0 `kernel`、R1 `configx`/`observex`/`testkitx`、R2 存储与消息类库。release evidence 必须说明每个 ring 的状态：`verified`、`blocked`、`not_applicable` 或 `not_run_with_reason`。如果目标是 9 分以上，至少应让 R0 和 R1 从样本验证进入真实仓库验证。

### P8. 测试体量大是优势，也是维护压力

严重度：中低

证据：

- 测试 Go 代码 14326 行，非测试 Go 代码 11462 行。
- `cmd/goalcli/main_test.go` 2060 行。

影响：

高测试投入保护了治理行为，但当测试集中绑定单体入口时，重构成本也会上升。未来拆分 `goalcli` 时，如果先拆实现而不拆测试结构，会产生大量噪声 diff。

建议：

在拆分 `goalcli` 前先拆测试夹具：命令解析测试、registry 测试、planned command validator 测试、external runner contract 测试、release score 测试分别归档。先改测试边界，再改生产代码边界。

### P9. 版本漂移已改善，但仍依赖硬编码常量

严重度：低

证据：

- `cmd/goalcli/governance.go:18` 到 `cmd/goalcli/governance.go:20` 当前硬编码 `projectReleaseVersion = "v0.4.6"` 和 `governanceRuntimeVersion = "v2.9.3"`。

影响：

只要 release 流程严格同步，这不是当前缺陷；但它仍是未来版本漂移的单点来源。

建议：

把版本来源收敛为生成文件或 manifest 字段，由 release preflight 校验 `goalcli version`、release docs、manifest template 和 tag 目标一致。

## 历史问题复核

对比 `docs/project-analysis-20260602.md` 和 `docs/structural-issues-20260602.md`，当前项目已经修复或缓解了几类问题：

- `goalcli score` 当前达到 10.0/10，而历史分析中的 8.2/10 综合状态已不是当前状态。
- Issue Registry 不再依赖固定数量判断，本轮 `goalcli issue-registry` 已通过。
- planned command 已增加空文件、目录、JSON 和部分 semantic marker 校验。
- context profile 自递归和 Makefile dependency 解析问题已纳入 `context-profile-check`，本轮验证通过。

仍然保留的问题是结构性而非单点缺陷：`goalcli` 单体化、字符串式 gate、Go/shell 证据分裂、治理文档面过大、downstream 真实采纳不足。

## 优先修复路线

### 第一阶段：降低误判风险

目标：让 gate 结果更接近真实语义。

1. 为 planned command 建立 typed validator。
2. 把 `goalcli score` 文档和 CLI 输出都明确标注为 release governance score。
3. 给外部脚本定义统一 JSON report contract。
4. 将 docs-check 中最脆弱的正文 needle 改为结构化元数据或标题/命令块校验。

### 第二阶段：降低维护成本

目标：让新增 gate 不再扩大单体文件。

1. 拆分 `cmd/goalcli/governance.go` 为 `registry`、`contextprofile`、`plannedcommand`、`releasegate` 等内部包或文件组。
2. 把 `cmd/goalcli/main.go` 的 switch 替换为 command registry。
3. 拆分 `cmd/goalcli/main_test.go`，先稳定测试边界再移动实现。
4. 合并低信息密度 `.agent` 文件为 schema-backed pack。

### 第三阶段：补齐 downstream 证据

目标：让模板标准从“本仓库可验证”提升到“目标库可采纳”。

1. 建立 downstream rings。
2. 将 R0/R1 真实仓库验证纳入 release evidence。
3. 对 R2 暂未覆盖目标，要求 `blocked owner`、原因、下一次复验日期和最小验证命令。

## 证据与推断边界

本报告中的“已验证事实”来自本地命令输出和当前文件内容。评分权重、严重度排序和 8.6/10 综合分属于工程判断，不是 `goalcli` 的机器输出。

本轮未执行完整 `make ci`、`make race`、`make security`、`make integration` 或 `make release-final-check`。这些命令耗时和外部工具要求更高，若用于发布签核，应单独执行并记录 evidence。

## 最终判断

当前项目可以评为 **8.6/10**：基础能力和治理成熟度强，已经接近可持续交付标准；但结构健康仍低于治理分，主要瓶颈是 `goalcli` 单体、planned command 语义验证不均、字符串式 gate、治理文档面过大和 downstream 真实采纳不足。

如果完成本报告建议的第一阶段和第二阶段，且 R0/R1 downstream 进入真实仓库验证，项目综合分有机会提升到 **9.0 到 9.2**。如果继续扩展 gate 但不拆分 `goalcli` 和证据 contract，分数会维持在 **8.4 到 8.7** 区间，并逐渐转化为维护成本问题。
