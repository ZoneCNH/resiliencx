# Template Generation Contract

`scripts/render_template.sh` 是从 `resiliencx` 生成具体基础库的唯一标准入口。旧 `resiliencx` module path 只作为迁移扫描项和兼容说明保留，不能作为新主身份。

## 默认示例

```bash
scripts/render_template.sh \
  --module-name kernel \
  --module-path github.com/ZoneCNH/kernel \
  --package-name kernel \
  --out ../kernel
```

## 必须替换

- Module name、module path 和 package name 占位符必须替换为目标库值。
- `github.com/ZoneCNH/resiliencx` 模板自身 import 到目标 module path。
- 旧迁移扫描项：`github.com/ZoneCNH/resiliencx`、`resiliencx`、`foundationx`。
- `pkg/resiliencx` 目录名到 `pkg/<package-name>`。
- README、docs、contracts、examples、scripts 和 manifest 中的模板占位。

## 不变量

- `--out` 不得指向 `resiliencx` 仓库根目录，也不得位于仓库内部。
- 生成库不得依赖 `x.go` 或业务仓库。
- 生成库不得读取 `/home/k8s/secrets/env/*`；该路径只属于调用方部署配置。
- 生成后的 module 必须在 `GOWORK=off` 下运行测试、contracts、boundary 和 release Evidence gate。
- 旧名只可在迁移文档或兼容说明中出现，不得作为生成库主标题、module name、package name 或 release 主体。

## GoalCLI 控制面同步

生成库必须保留完整 `goalcli` 治理控制面，至少包括：

- `cmd/goalcli/**`
- `internal/goalcli/README.md`
- `Makefile` 中的 `GOALCLI ?= go run ./cmd/goalcli`
- `.agent/index.yaml`
- `.agent/harness/harness.yaml`
- `.agent/harness/gates.md`
- `.agent/registries/command-registry.yaml`
- `.agent/registries/command-implementation-status.yaml`
- `.agent/registries/makefile-baseline.yaml`
- `.agent/registries/makefile-target-registry.yaml`
- `contracts/goalcli-report.schema.json`
- `docs/standard/goalcli-cli-contract.md`
- `docs/standard/goalcli-runtime.md`

`goalcli` 命令实现、usage、Makefile 目标、registry、harness、schema 和文档必须同批同步；不得只同步其中一个 surface。

## Metrics Prefix

Metrics Prefix 必须跟随 package name 替换。模板中的 `resiliencx_` prefix 在 `kernel` 渲染后必须变为 `kernel_`，在 `configx` 渲染后必须变为 `configx_`，在 `redisx` 渲染后必须变为 `redisx_`。metrics contract、README、docs、examples、测试和 snapshot 中不得残留 `resiliencx_`，除非某个文件被明确 allowlist 为模板来源说明。

## 排除规则

generator 不得复制：

- `.git/`
- `.omc/`
- `.omx/`
- `.worktree/`
- `.agent/inbox/`
- `release/manifest/latest.json`
- `release/manifest/latest.json.sha256`
- `release/standard-impact/latest.md`
- `release/downstream-sync/latest.md`
- `release/debt/latest.json`
- `release/debt/latest.md`
- `release/debt/latest.json.sha256`
- `docs/adr/`
- 旧迁移单文件目标文档；当前权威 `docs/goal/` 目录必须作为治理控制面同步。
- 临时文件、缓存、coverage 输出、构建目录、本地 Evidence 输出和 editor 产物。

## 输出不变量

生成结果必须满足：

- `go.mod` module path 正确。
- 公共包目录和 package name 正确。
- README、docs、contracts、Makefile、scripts、CI 和 `.agent/` 模板存在。
- 无 template token 未替换残留。
- 无 generic placeholder、TODO-style template marker 或 `resiliencx_` metrics prefix 残留。
- 无 `resiliencx` module import 残留，除非在文档中作为来源说明出现。
- `GOWORK=off go mod tidy` 后 `go.mod` 和 `go.sum` 保持 clean。

## Scanner

`scripts/check_rendered_template.sh` 必须扫描：

- stale module path：`github.com/ZoneCNH/resiliencx`。
- stale module name：`resiliencx`。
- stale package directory：`pkg/resiliencx`。
- stale package name：`resiliencx`、`Resiliencx`、`RESILIENCX`。
- stale metrics prefix：`resiliencx_`。
- unresolved template token、generic placeholder 和 TODO-style template marker。

扫描 unresolved template token 时，应跳过合法包含表达式语法的 GitHub Actions workflow、检查脚本自身和 Go template probe 脚本，避免把 scanner 规则或 `go list -f` 模板语法误判为未替换占位符。

扫描失败时 integration gate 必须失败。

## Release 验证

任何 generator 修改必须附带 integration Evidence。Release 级验证必须证明渲染出的 `kernel`、`configx` 和 `redisx` 可以独立运行：

```bash
GOWORK=off go mod tidy
GOWORK=off make docker-toolchain-check
GOWORK=off go test ./...
GOWORK=off make contracts
GOWORK=off make boundary
GOWORK=off make standard-impact-check
GOWORK=off make debt
GOWORK=off make debt-evidence
GOWORK=off make debt-evidence-checksum-check
CHECK_STATUS=passed GOWORK=off make evidence
RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check
```

模板仓库侧验证入口：

```bash
GOWORK=off make integration
GOWORK=off make boundary
GOWORK=off make contracts
GOWORK=off make release-check
```

## Docker Toolchain Runtime 输出

模板生成契约包含 Docker Toolchain Runtime surface。渲染产物必须包含 `Dockerfile`、`docker-compose.yml`、`.dockerignore`、`.devcontainer/devcontainer.json`、`scripts/docker/check_toolchain.sh`、`scripts/docker/docker_gate.sh`，并在 Makefile 暴露 `docker-toolchain-check`、`docker-build`、`docker-build-check`、`docker-shell`、`docker-ci`、`docker-release-check`、`docker-release-final-check`、`docker-goalcli`、`docker-goalcli-image`、`docker-goalcli-version`、`docker-runtime-check`、`docker-drift-check`、`docker-contract`。`scripts/docker/docker_gate.sh` 只能调用既有 Makefile gate；Docker 不是第二套 gate。`scripts/check_rendered_template.sh` 必须把 Docker 文件和 targets 纳入 scanner。
