# 快速入门

本指南帮助新协作者在本地搭建 xlib-standard 开发环境，完成首次构建、测试与门禁验证。

> 本仓库的通过状态仅代表 xlib-standard 自身；下游 kernel/configx 的通过状态必须来自真实下游运行结果。

## 环境要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.23.x（遵循 `.tool-versions`） | 编译与测试 |
| Git | 2.30+ | 版本管理与 hooks |
| golangci-lint | 最新稳定版 | 可选，`make lint` 需要 |
| govulncheck | v1.3.0 | 可选，`make security` 需要 |

> 缺少 `golangci-lint` 或 `govulncheck` 时，对应的 `make lint` / `make security` 会直接失败，不允许跳过。

## 快速开始

### 1. 克隆仓库

```bash
git clone https://github.com/ZoneCNH/xlib-standard.git
cd xlib-standard
```

### 2. 安装 Git Hooks

```bash
make install-hooks
make doctor-hooks
```

`make install-hooks` 将 `git config core.hooksPath` 指向仓库内的 `.githooks/` 目录，启用本地 P0 防线：

- **pre-commit**：禁止在 `main` 分支上直接提交 + secret 扫描（拦截硬编码密钥）
- **pre-push**：禁止直接 push 到 `main` 分支

`make doctor-hooks` 验证 hooks 配置是否生效。未启用 hooks 时，上述防线形同虚设。

### 3. 同步 main 分支

```bash
make sync-main
```

拉取远端 `main` 并 fast-forward 本地分支，确保后续工作基于最新代码。

### 4. 构建与测试

```bash
make build          # 编译所有包
make test           # 运行全部单元测试
make race           # 使用 race detector 运行测试
make lint           # 代码检查（需要 golangci-lint）
make security       # 漏洞扫描 + secret 检查（需要 govulncheck）
```

运行单个测试：

```bash
go test ./pkg/templatex/ -run TestConfigValidate
go test ./internal/sanitize/ -run TestSanitize
```

### 5. 运行 CI Gate

```bash
make ci             # 完整 CI 链：fmt → vet → lint → test → race → boundary → security → contracts → governance → score
make ci-extended    # 扩展 CI：ci + property + golden + fuzz-smoke + docs-drift
```

### 6. 治理检查

```bash
make governance-check         # P0 治理全量检查
make p1-governance-check      # P1 本地 dry-run 治理契约
make p2-runtime-check         # P2 运行时/downstream dry-run 契约
```

### 7. 发布流程

所有发布和验证命令必须使用 `GOWORK=off`，避免本地 `go.work` 改写 module 解析：

```bash
GOWORK=off make release-check                                        # 发布检查
XLIB_CONTEXT=release_verify GOWORK=off make release-final-check      # 发布终检（含工作区 clean 要求）
XLIB_CONTEXT=release_verify GOWORK=off make release-preflight VERSION=vX.Y.Z  # 发布预检
```

`release-check` 会依次执行 CI、集成测试、依赖漂移检查、标准影响报告、文档 gate、分数 gate、治理全量检查，最后生成 release manifest 及其校验和。

### 8. 常用命令速查

| 命令 | 用途 |
|------|------|
| `make build` | 编译所有包 |
| `make fmt` | 格式化代码 |
| `make vet` | 静态分析 |
| `make test` | 运行全部测试 |
| `make race` | 竞态检测 |
| `make lint` | golangci-lint 检查 |
| `make security` | govulncheck + secret 扫描 |
| `make boundary` | 模块边界检查 |
| `make contracts` | JSON schema 契约检查 |
| `make property` | 属性/不变量测试 |
| `make golden` | Golden/快照测试 |
| `make fuzz-smoke` | Fuzz smoke（默认 10s/target） |
| `make integration` | 集成测试 |
| `make ci` | 完整 CI 链 |
| `make ci-extended` | 扩展 CI 链 |
| `make docs-check` | 文档结构 gate |
| `make dependency-check` | 依赖漂移检查 |
| `make standard-impact-check` | 标准影响报告 |
| `make score` | Release score gate（≥9.8） |
| `make governance-check` | P0 治理全量检查 |
| `make evidence` | 生成 release manifest |
| `make doctor` | 环境自检 |
| `make install-hooks` | 启用 git hooks |
| `make doctor-hooks` | 验证 hooks 配置 |
| `make sync-main` | 同步远端 main |

### 9. 下一步

- 阅读 [基础库标准索引](standard/README.md) 了解标准体系
- 阅读 [AGENTS.md](../AGENTS.md) 了解贡献规范与编码约定
- 阅读 [发布指南](release.md) 了解完整发布流程与 Evidence 要求
- 阅读 [Harness Gate](standard/harness-gates.md) 了解所有门禁命令
- 阅读 [Evidence 协议](standard/evidence-protocol.md) 了解完成声明规范
