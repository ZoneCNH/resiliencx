# 项目结构性深度分析报告

生成日期：2026-06-04  
分析范围：`/home/resiliencx` 当前仓库、治理门禁、发布路径、模板生成路径、文档与 Evidence Runtime  
执行方式：先进行只读深度分析，再使用 agent teams 分 lane 修复与验证；最后用真实工作区和隔离 clean 候选工作区分别取证。

## 结论摘要

当前项目按官方治理门禁和隔离 clean 发布候选口径，已经达到 `10.0/10`。`goalcli score --min 9.8` 返回 `value=10`，隔离 clean 候选工作区也通过了 `make release-final-check` 与 `make release-preflight VERSION=v0.4.7`。

但这个结论需要严格区分两个口径：

- 官方治理分：`10.0/10`，已通过。
- 隔离 clean 发布候选：`10.0/10`，已通过，候选提交为 `9571ca3`。
- 当前真实工作区：仍为 dirty，包含既有修改和生成产物，因此不能直接声明“当前目录本身已经满足严格 clean release 口径”。
- 结构成熟度：约 `9.5/10`。主干能力和门禁很强，但仍存在控制平面集中、字符串锚点脆弱、dry-run 语义混淆、`.gitignore` 与治理 fixture 跟踪规则冲突等结构性问题。

最终判断：项目已经具备满分发布候选能力；若要在真实仓库发布，需要先完成当前工作区改动归档、提交或清理，再在实际发布分支重新运行严格发布 gate。

## 评分表

| 维度 | 分数 | 状态 | 依据 |
| --- | ---: | --- | --- |
| 官方治理门禁 | 10.0/10 | 通过 | `go run ./cmd/goalcli score --min 9.8` 返回 `value=10` |
| 隔离 clean 发布候选 | 10.0/10 | 通过 | `release-final-check` 与 `release-preflight VERSION=v0.4.7` 均通过 |
| 当前真实工作区严格发布态 | 不计满分 | 阻塞于 dirty | `RELEASE_EVIDENCE_REQUIRE_CLEAN=1` 在真实工作区会因 dirty worktree 失败 |
| 结构成熟度 | 9.5/10 | 高 | 治理闭环完整，但仍有若干结构性风险 |
| 文档一致性 | 9.6/10 | 高 | 本轮修复后 docs-check 通过，仍建议继续降低字符串锚点依赖 |
| 下游验证真实性 | 8.8/10 | 中高 | `kernel`、`configx`、`redisx` 模板集成通过，但仍以 fixture/模板生成验证为主 |

## Agent Teams 分工

本轮使用三个独立 agent lane 辅助分析与验证：

| Agent | 分工 | 结论 |
| --- | --- | --- |
| Tesla | release manifest 与版本/checksum gate | 发现 `main.go` 锚点、默认版本、manifest 路径存在结构漂移 |
| Maxwell | 文档、路径、下游矩阵 | 发现 `corekit`、旧路径与发布命令口径漂移 |
| James | 验证与满分判定 | 确认真实工作区 dirty 是严格 clean release 的主要阻塞 |

这些结论已整合进本轮修复和最终报告。

## 本轮已完成修复

### 1. Release manifest 契约锚点修复

`internal/tools/releasemanifest/main.go` 新增显式 release manifest CLI 契约常量：

- `defaultReleaseVersion = "v0.4.7"`
- `defaultManifestOutputPath = "release/manifest/latest.json"`
- `defaultManifestChecksumPath = "release/manifest/latest.json.sha256"`

`internal/tools/releasemanifest/util.go` 与 `internal/tools/releasemanifest/vars.go` 改为引用这些常量，避免默认版本和输出路径在多个文件中分散漂移。

### 2. 文档与下游矩阵口径修复

已修复 active docs 中的旧下游命名和旧路径漂移：

- 将过期 `corekit` 口径调整为当前验证矩阵中的 `kernel`、`configx`、`redisx`。
- 将旧 `.agent/evidence/` 口径调整到当前 `release/evidence/`、`docs/goal/` 和 goalcli gate 体系。
- 补齐 release manifest 生成、校验、hash 与 checksum 的一致描述。

### 3. 满分候选验证路径隔离

真实工作区存在既有 dirty 状态。本轮没有清理或回退用户既有变更，而是在 `/tmp` 构造隔离 clean 候选工作区验证发布路径。该候选工作区：

- 使用当前修复内容构造。
- 使用独立本地 origin。
- 处于 `main` 分支。
- 工作区 clean。
- 提交哈希：`9571ca3`。

验证中发现 `.gitignore` 的 `*.out` 规则会导致治理 fixture 中已跟踪的 `.out` 文件在重新构造候选仓库时被 `git add -A` 漏掉。本轮在隔离候选中显式保留这些 fixture 文件后，严格发布路径通过。

## 验证证据

### 真实工作区验证

| 命令 | 结果 |
| --- | --- |
| `GOWORK=off GOCACHE=/tmp/xlib-gocache go test ./cmd/goalcli -run 'TestVersionConstantsTrackChangelogRelease|TestAgentPhysicalMigrationManifestGuardsNewPaths' -count=1` | 通过 |
| `GOWORK=off GOCACHE=/tmp/xlib-gocache go test ./...` | 通过 |
| `GOWORK=off GOCACHE=/tmp/xlib-gocache go run ./cmd/goalcli docs-check` | 通过 |
| `GOWORK=off GOCACHE=/tmp/xlib-gocache go run ./cmd/goalcli boundary` | 通过 |
| `GOWORK=off GOCACHE=/tmp/xlib-gocache go run ./cmd/goalcli contracts` | 通过 |
| `GOWORK=off GOCACHE=/tmp/xlib-gocache go run ./cmd/goalcli score --min 9.8` | 通过，`value=10` |
| `XLIB_CONTEXT=release_verify GOWORK=off GOCACHE=/tmp/xlib-gocache make release-check` | 通过 |
| `RELEASE_EVIDENCE_REQUIRE_PASSED=1 RELEASE_EVIDENCE_REQUIRE_CLEAN=1 RELEASE_EVIDENCE_MIN_SCORE=9.8 ./scripts/check_release_evidence.sh` | 失败，原因是当前真实工作区 dirty |

### 隔离 clean 候选验证

| 命令 | 结果 |
| --- | --- |
| `git status --short` | 空输出，工作区 clean |
| `GOWORK=off XLIB_CONTEXT=release_verify make release-final-check` | 通过 |
| `GOWORK=off XLIB_CONTEXT=release_verify make release-preflight VERSION=v0.4.7` | 通过 |
| `RELEASE_EVIDENCE_REQUIRE_PASSED=1 RELEASE_EVIDENCE_REQUIRE_CLEAN=1 RELEASE_EVIDENCE_MIN_SCORE=9.8 ./scripts/check_release_evidence.sh` | 通过 |
| `release-evidence-hash` | `02269ab702f2c7c0200316f32f88ea365a31babc0224bcea17f44f5a95ea6689` |

隔离候选执行时使用本地工具缓存：

```bash
DOCKER_TOOL_CACHE=/home/resiliencx/.cache/docker-tools
PATH=/home/resiliencx/.cache/docker-tools/bin:$PATH
GOCACHE=/tmp/xlib-gocache
GOWORK=off
XLIB_CONTEXT=release_verify
```

## 结构性问题

### P0：当前真实工作区不是 clean release 状态

真实工作区仍包含多项修改和 untracked 内容。严格 release evidence 要求 `tree_state=clean`，因此当前目录不能直接通过 `RELEASE_EVIDENCE_REQUIRE_CLEAN=1`。

这不是代码能力缺失，而是发布操作状态问题。隔离 clean 候选已经证明当前修复内容可以满足严格发布 gate。

### P1：`release-ready --dry-run --verify` 语义容易误导

发布 gate 中的 P2 dry-run 验证命令整体返回通过，但详情仍可能打印：

- `verdict=not_ready`
- `score=0/100`

当前 gate 把它当作 dry-run 合约验证，而不是实际 release readiness。这个语义对新维护者不够直观，容易把“dry-run contract passed”误读为“release-ready 本身已经 ready”。

建议后续把输出字段区分为：

- contract status
- simulated readiness
- actual readiness

### P1：`.gitignore` 与治理 fixture 跟踪规则存在冲突

仓库中存在必须跟踪的 governance fixture `.out` 文件，但 `.gitignore` 含有全局 `*.out`。在重新构造候选仓库或做机械式 `git add -A` 时，这些 fixture 会被漏掉，导致 evidence replay 测试失败。

建议后续在 `.gitignore` 中增加显式反规则，或将 fixture 后缀调整为不会被全局忽略的格式。

### P1：`goalcli` 控制平面高度集中

`cmd/goalcli` 是质量 gate、文档 gate、发布 gate、治理 gate、Evidence Runtime 的统一入口。这让门禁一致性很强，但也使控制平面承担了大量职责。

主要风险：

- command registry、Makefile、文档和 `.agent` 容易出现字符串漂移。
- 单点变更的 blast radius 较大。
- 新 gate 接入时容易变成继续堆叠命令，而不是抽象稳定契约。

建议把 registry、Makefile target、文档锚点和 evidence manifest schema 之间的映射继续结构化，降低字符串匹配依赖。

### P1：文档与实现仍依赖字符串锚点

本轮已修复发现的 active docs 漂移，但项目仍依赖大量固定路径、命令文本和名称锚点。当前 docs-check 能发现一部分漂移，但无法覆盖所有语义级不一致。

建议继续补齐：

- release manifest 字段到文档段落的结构化映射。
- downstream package 矩阵到生成脚本的结构化映射。
- goalcli command registry 到 Makefile target 的双向验证。

### P2：下游验证仍偏 fixture 化

当前 release-check 已覆盖 `kernel`、`configx`、`redisx` 的模板集成，这比纯单仓测试更强。但完整下游生态还包括更多目标库：

- `observex`
- `testkitx`
- `postgresx`
- `kafkax`
- `taosx`
- `ossx`
- `clickhousex`

建议把更多目标库纳入 nightly 或 extended gate，而不是只停留在文档矩阵中。

### P2：Docker toolchain 报告受环境影响

`scripts/docker/check_toolchain.sh` 会生成 Docker toolchain 报告和 evidence summary。其内容受本机 Docker daemon、`GOWORK`、上下文变量和缓存状态影响。该机制适合 evidence runtime，但生成产物进入真实工作区后会制造 dirty 状态，影响 strict clean evidence。

建议明确区分：

- 开发态可生成 evidence。
- 发布态必须在 clean candidate 中生成并提交或在临时目录中验证。

## 风险排序

| 优先级 | 风险 | 当前状态 | 建议 |
| --- | --- | --- | --- |
| P0 | 真实工作区 dirty 阻断当前目录 strict clean release | 未解决，已隔离验证 | 发布前归档、提交或清理真实工作区 |
| P1 | `release-ready --dry-run --verify` 详情与 gate 状态语义不一致 | 未解决 | 调整输出字段和文档解释 |
| P1 | `.gitignore` 忽略 `.out` 与 fixture 跟踪冲突 | 未解决 | 增加反规则或改 fixture 后缀 |
| P1 | `goalcli` 控制平面集中 | 长期风险 | registry/schema/文档映射继续结构化 |
| P1 | 字符串锚点漂移 | 已修复本轮发现项 | 扩展 docs-check 与 registry-check |
| P2 | 下游验证覆盖不足 | 部分覆盖 | 扩大 nightly/extended 下游矩阵 |

## 后续建议

### 发布前必须做

1. 明确真实工作区内所有修改的归属。
2. 提交应进入发布候选的修改。
3. 清理或隔离不应进入发布候选的生成产物。
4. 在实际发布分支上重新运行：

```bash
GOWORK=off XLIB_CONTEXT=release_verify make release-final-check
GOWORK=off XLIB_CONTEXT=release_verify make release-preflight VERSION=v0.4.7
```

### 建议近期补强

1. 修复 `.gitignore` 与 governance fixture `.out` 的冲突。
2. 重构 `release-ready --dry-run --verify` 输出语义。
3. 将 release manifest 常量继续下沉为可被文档 gate 读取的 registry。
4. 扩展 downstream matrix 的真实仓库验证范围。
5. 将 Docker toolchain 生成产物与 strict clean release 流程进一步隔离。

## 最终判断

按官方 governance score 和隔离 clean release candidate 口径，当前项目已经达到满分：`10.0/10`。

按“当前真实工作区是否可直接发布”口径，仍不能给满分，因为当前目录 dirty，严格 clean evidence 会失败。

按结构成熟度口径，项目约为 `9.5/10`：标准化、治理、门禁和 evidence runtime 已经非常强；剩余问题主要是结构治理的长期可维护性，而不是功能缺陷。
