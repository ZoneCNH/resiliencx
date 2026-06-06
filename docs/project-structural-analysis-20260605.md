# 项目结构分析报告（2026-06-05）

## 结论

当前项目综合评分：**8.6 / 10**。

该评分基于本次同步后的结构状态：Standard、Go Reference Template、Generator、Harness 和 Evidence Runtime 已形成可执行闭环；`govulncheck` 已从“启用即每次运行”改为一周窗口，默认 gate 不再依赖外部漏洞库。但在本次提交推送和远端自动发布 workflow 完成前，发布证据仍缺少端到端闭环，因此不按 release gate 的 `9.8` 阈值给分。

若 `release-check`、`release-final-check`、`release-preflight VERSION=v0.4.13` 与 GitHub Release 对象全部通过并可追溯，本项目可上调到 **9.2 - 9.5**。距离 **9.8** 的主要差距不是单点功能，而是多源事实、发布 Evidence 操作成本和 workflow 远端证据仍偏重。

## 评分分解

| 维度 | 评分 | 说明 |
| --- | ---: | --- |
| 标准与实现一致性 | 9.0 | `cmd/goalcli`、Makefile、workflow、README、标准文档和 `.agent` 工件已对齐到 `v0.4.13` 与每周漏洞扫描窗口。 |
| Gate 可执行性 | 8.8 | `make security` 默认只运行 secret scan；仅显式启用且一周窗口到期、状态缺失或强制时运行 `govulncheck`，降低外部漏洞库对日常 gate 的阻塞。 |
| 发布 Evidence | 8.4 | manifest schema、hash、score、workflow artifact 规则完整；仍需要本次提交后的 clean workspace、远端 workflow 与 release 对象证据。 |
| 安全门禁 | 8.6 | secret scan 保持强制，漏洞扫描改为 weekly/force；Security workflow 每周强制扫描，默认 CI/Release/Auto Patch/Docker Contract 不访问漏洞库。 |
| 文档可追溯 | 8.7 | README、`docs/release.md`、标准文档、`.agent` 发布和证据工件已同步；历史审计和旧结构报告保留旧事实，需要读者按日期区分。 |
| 维护复杂度 | 7.6 | 版本号、安全工具版本、workflow policy 和 Evidence 描述仍分散在代码、脚本、文档与 `.agent` 多处，漂移成本高。 |

## 证据口径

- Evidence：`cmd/goalcli/main.go` 定义 `XLIB_ENABLE_VULNCHECK`、`XLIB_FORCE_VULNCHECK`、`XLIB_VULNCHECK_INTERVAL_HOURS` 和 `XLIB_VULNCHECK_STATE`，并在 `security` 子命令中按 weekly/force 判断是否运行 `govulncheck`。
- Evidence：`.github/workflows/ci.yml`、`.github/workflows/release.yml`、`.github/workflows/release-auto-patch.yml` 和 `.github/workflows/docker-contract.yml` 默认设置 `XLIB_ENABLE_VULNCHECK=0`。
- Evidence：`.github/workflows/security.yml` 增加每周一 03:17 UTC 定时任务，并在定时任务中设置 `XLIB_FORCE_VULNCHECK`。
- Evidence：`scripts/check_release_preflight.sh` 仅在漏洞扫描到期、状态缺失或强制时要求本地存在 `govulncheck`。
- Evidence：`README.md`、`docs/release.md`、`docs/standard/security-and-secret-policy.md`、`docs/standard/harness-gates.md`、`docs/supply-chain.md` 与 `.agent` 工件已同步 weekly/force 口径。
- Evidence：`cmd/goalcli/governance.go`、`pkg/resiliencx/version.go`、`internal/tools/releasemanifest/main.go` 和 `release/manifest/template.json` 已同步到 `v0.4.13`。
- Inference：8.6 分是结构和本地待验证状态评分，不等同于 `goalcli score` 的发布门禁结果；远端发布完成后才可按 release evidence 重新计分。
- Unknown：在推送前尚无当前 HEAD 的 GitHub Actions run、tag 和 GitHub Release 对象证据。

## 结构性问题

1. **权威事实仍然分散**

   版本号、`govulncheck` 固定版本、workflow 默认环境变量、发布命令示例和文档规则散落在 Go 常量、shell 脚本、workflow YAML、README、标准文档、`.agent` policy 和 harness 文件中。当前已经同步，但后续版本升级仍容易出现局部更新遗漏。

2. **发布证据链强，但操作面偏大**

   `release-final-check`、manifest、hash、score、contracts 和 workflow artifact 构成较完整的发布证据链；代价是本地 clean workspace、origin 同步、tag 不存在、工具存在性和 manifest 生成状态都必须同时满足。对于频繁 patch release，维护者需要明确区分本地开发 gate、release gate 和远端发布 gate。

3. **安全门禁已经降耦合，但仍需定时证据**

   本次调整把漏洞库访问从日常 CI/Release gate 移出，解决“每次都执行”的成本问题。新的风险转为：Security workflow 每周定时扫描必须可观测；若长期失败或被禁用，默认 gate 通过并不代表漏洞扫描证据仍新鲜。

4. **历史报告与当前口径并存**

   `docs/project-structural-analysis-20260604.md`、`docs/independent-audit-20260602.md`、`docs/project-analysis-20260602.md` 和历史 changelog 保留当时事实，包括旧版本号和旧 `govulncheck` 口径。这是正确的审计时间线，但搜索结果会混合当前事实与历史事实。

5. **`.agent` 工件信息密度高**

   `.agent` 目录覆盖 runtime、harness、policy、evidence、release 和 retrospective，适合作为机器可执行治理层；但人工阅读成本高。当前需要 `CONSTITUTION.md`、`AGENTS.md` 和 `docs/standard/` 共同解释权威顺序，否则新维护者容易把历史或辅助工件误认为唯一权威。

## 本次对齐结果

- `goalcli security` 默认只运行 secret scan。
- `XLIB_ENABLE_VULNCHECK=1` 时仅在一周窗口到期或状态缺失时运行 `govulncheck`。
- `XLIB_FORCE_VULNCHECK=1` 可强制执行漏洞扫描。
- CI、Release Check、Auto Patch 和 Docker Contract workflow 默认不安装或访问漏洞库。
- Security workflow 每周定时强制执行固定版本 `golang.org/x/vuln/cmd/govulncheck@v1.1.4`。
- 发布版本元数据、manifest 模板、README、标准文档和 `.agent` 工件同步到 `v0.4.13`。

## 改进建议

1. 建立单一版本事实源，或用 `goalcli` 校验所有当前版本引用，减少手工同步。
2. 将 `scripts/check_docs.sh` 中的脆弱文本断言逐步改为结构化 YAML/Markdown 检查。
3. 为 Security weekly workflow 增加失败告警或 dashboard 摘要，避免漏洞扫描证据悄然过期。
4. 将历史报告集中加上“历史快照”索引，降低搜索时误读旧事实的概率。
5. 继续保持 `latest.json` 和 `.sha256` 为生成产物，不提交到源码历史。

## 发布前停止条件

本次变更在以下条件都满足后才可按发布完成声明：

- 本地 Go 测试、文档检查、release check 和 release final check 通过。
- 工作区 clean，`main` 已推送到 `origin/main`。
- `release-preflight VERSION=v0.4.13` 在推送后通过。
- `.github/workflows/release-auto-patch.yml` 对当前 HEAD 成功完成，或当前 HEAD 已存在稳定 release tag。
- GitHub Release 对象存在，且不是 draft / prerelease。
