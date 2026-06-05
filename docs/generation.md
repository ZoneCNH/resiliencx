# 生成模板

## 用途

`scripts/render_template.sh` 用于把 `resiliencx` 参考模板渲染为具体基础库，例如 `kernel`。标准源仓库是 [`resiliencx`](https://github.com/ZoneCNH/resiliencx)，并同时承载模板、generator、Harness 和 Evidence 实现。旧 `resiliencx` / `foundationx` 名称只作为迁移文档语境保留。脚本负责同步替换 module name、module path、package name、`pkg/` 目录名、imports、文档占位符和脚本中的模板名称。

## 示例

```bash
scripts/render_template.sh \
  --module-name kernel \
  --module-path github.com/ZoneCNH/kernel \
  --package-name kernel \
  --out ../kernel
```

`--out` 必须指向不存在或为空的目录，避免覆盖已有仓库内容。

## Repository Governance Pack

需要把标准治理面落地到下游仓库时，渲染命令必须启用 governance pack：

```bash
scripts/render_template.sh \
  --module-name kernel \
  --module-path github.com/ZoneCNH/kernel \
  --package-name kernel \
  --layer L0 \
  --enable-governance \
  --standard-version v0.5.0 \
  --standard-commit "$(git rev-parse HEAD)" \
  --out ../kernel
```

`--enable-governance` 必须和 `--layer`、`--standard-version`、`--standard-commit` 同时使用；缺少任一 provenance 字段都会中止渲染。启用后脚本会写入 `xlib-standard.lock`，并在渲染结束前验证 `.githooks/pre-commit`、`.githooks/pre-push`、`.github/workflows/adoption-check.yml`、`.github/rulesets/protect-main.json`、`mk/governance.mk`、`.agent/harness/harness.yaml` 已随下游控制面一起落地。下游仓库必须运行 `GOWORK=off make adoption-check`，证明 Repository Governance Pack、lock、workflow、main ruleset、registry、Makefile target 和 harness gate 没有被裁剪；`mk/governance.mk` 自身也会阻断未设置 `GOWORK=off` 的本地执行。main ruleset 必须禁止 bypass，并要求 `adoption-check`、`governance-check` 和 `release-check`。标准源仓库没有 downstream lock，因此 source-template 的 GitHub workflow 会显式跳过这个 downstream-only gate，渲染后的下游仓库仍会执行 `GOWORK=off make adoption-check`。

## 渲染范围

- `resiliencx` 替换为 `--module-name`。
- `github.com/ZoneCNH/resiliencx`、`github.com/ZoneCNH/resiliencx` 和迁移兼容的 `github.com/ZoneCNH/resiliencx` 替换为 `--module-path`。
- `resiliencx` 和迁移兼容的 `resiliencx` 替换为 `--module-name`。
- `resiliencx`、`pkg/resiliencx` 和 `resiliencx` imports 替换为 `--package-name`。
- 文档、Go 代码、JSON contract、shell 脚本、Makefile 和 CI 配置同步更新；标准源仓库仍是 [`https://github.com/ZoneCNH/resiliencx`](https://github.com/ZoneCNH/resiliencx)，渲染产物中的源身份会改写为下游 module identity，避免残留模板仓库名称。
- `cmd/goalcli/`、`internal/goalcli/README.md`、`Makefile`、`.agent/harness/`、`.agent/registries/`、`docs/standard/goalcli-cli-contract.md` 和 `contracts/goalcli-report.schema.json` 作为下游治理控制面同步。命令实现、registry、harness、schema 和文档变更必须同批进入模板。

脚本不会复制 `.git`、`.omc`、`.omx`、`.worktree`、`.agent/inbox`、`docs/adr`、旧迁移单文件目标文档、`release/manifest/latest.json`、`release/manifest/latest.json.sha256`、`release/standard-impact/latest.md`、`release/downstream-sync/latest.md`、`release/debt/latest.json`、`release/debt/latest.md`、`release/debt/latest.json.sha256`、临时文件、缓存、coverage 输出和构建目录。生成后的库必须自己运行 release gate 生成新的 Evidence artifact；当前权威目标文档目录 `docs/goal/` 会随下游治理控制面同步。

## 验证

生成后至少运行：

```bash
GOWORK=off make release-check
GOWORK=off make adoption-check # 仅适用于启用 --enable-governance 的渲染 downstream 仓库
```

模板自身的 `make integration` 会渲染三个临时下游库：

- `kernel`：目标仓库路径 `github.com/ZoneCNH/kernel`，用于证明默认 L0 下游目标仍可生成。
- `configx`：目标仓库路径 `github.com/ZoneCNH/configx`，用于证明 L1 配置基础库形态仍可生成。
- `redisx`：目标仓库路径 `github.com/ZoneCNH/redisx`，用于证明 L2 profile 基础设施形态仍可生成。

每个临时库都会运行以下验证：

- `scripts/check_rendered_template.sh`：确认 `go.mod` module path、`pkg/<package>` 目录、旧模板目录、旧 module path、占位符和 `resiliencx` 标识。
- `GOWORK=off go mod tidy` 后检查 `go.mod` / `go.sum` 没有未提交差异。
- `GOWORK=off make docker-toolchain-check`
- `GOWORK=off go test ./...`
- `GOWORK=off make contracts`
- `GOWORK=off make boundary`
- `GOWORK=off make standard-impact-check`
- `GOWORK=off make debt`
- `GOWORK=off make debt-evidence`
- `GOWORK=off make debt-evidence-checksum-check`
- `CHECK_STATUS=passed GOWORK=off make evidence`
- `RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check`

这组验证用于防止生成脚本、包路径、imports、contract gate、boundary gate 和生成后 Evidence 回归。

## 生成后 Release Evidence

生成后的库会继承 `internal/tools/releasemanifest`。该工具会生成并校验 `release/manifest/latest.json`，其中包括当前 HEAD、tree SHA、源码摘要、contract SHA256、依赖清单和工具版本。发布前应使用：

```bash
GOWORK=off make release-final-check
```

`release-final-check` 要求所有 gate 状态为 `passed`，并要求 git 工作区为 `clean`。如果只是开发中自测，`make release-check` 已足够；它允许工作区显示 `dirty`，但仍会验证 manifest 和当前源码内容一致。

## 边界

生成后的基础库仍必须保持独立，不能依赖 `github.com/bytechainx/x.go`、`github.com/ZoneCNH/x.go` 或任何 `x.go/internal/*` 包；标准规则继续引用独立仓库 [`resiliencx`](https://github.com/ZoneCNH/resiliencx)。

## Docker Toolchain Runtime 模板继承

生成后的下游库必须继承 Docker Toolchain Runtime contract：`Dockerfile`、`docker-compose.yml`、`.dockerignore`、`.devcontainer/devcontainer.json`、`scripts/docker/check_toolchain.sh`、`scripts/docker/docker_gate.sh`，以及 `docker-toolchain-check`、`docker-build`、`docker-build-check`、`docker-shell`、`docker-ci`、`docker-release-check`、`docker-release-final-check`、`docker-goalcli`、`docker-goalcli-image`、`docker-goalcli-version`、`docker-runtime-check`、`docker-drift-check`、`docker-contract`。`scripts/check_rendered_template.sh` 会扫描这些文件和 targets，防止下游 Docker contract 漂移。
