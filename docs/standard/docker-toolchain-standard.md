# Docker Toolchain Runtime 标准

本页是 parent plan #62 的 Docker runtime contract。Docker Toolchain Runtime 是可复现的工具链运行时，不是第二套 gate；它只能把既有 Makefile gate 放进统一容器边界中执行，不能绕过、替换或降低 `make ci`、`make release-check`、Harness、CI 和下游验证要求。

## 角色与 stage

- `make docker-toolchain-check`：检查宿主 Docker CLI、daemon 与 buildx/BuildKit 可用性，并写入 `release/docker/toolchain-check.md`。该 stage 只证明 Docker Toolchain Runtime 可启动，不声称源码 gate 已通过。
- `make docker-build`、`make docker-build-check`、`make docker-shell`：分别构建 toolchain image、在容器内运行 `make build-check`、进入同一 bind-mounted 工作区 shell。
- `make docker-ci`：使用 BuildKit 构建 toolchain image，并在 bind-mounted 工作区中运行既有 `make ci`。它不是新增的第二套 CI 规则。
- `make docker-release-check`、`make docker-release-final-check`：使用同一个运行时边界执行既有 release gate。发布语境仍必须显式带上 `XLIB_CONTEXT=release_verify GOWORK=off`。
- `make docker-goalcli`、`make docker-goalcli-image`、`make docker-goalcli-version`、`make docker-runtime-check`、`make docker-drift-check`、`make docker-contract`：证明 goalcli 容器入口、runtime check、静态漂移检查和 Docker contract 聚合面没有分叉。

CI、Harness、release 和 downstream verification 都必须保持 `GOWORK=off`。本地发布示例：

```bash
XLIB_CONTEXT=release_verify GOWORK=off make docker-release-check
XLIB_CONTEXT=release_verify GOWORK=off make release-final-check
```

下游集成示例：

```bash
GOWORK=off make integration DOWNSTREAM=kernel
```

## Build context 与 `.git` 边界

`.dockerignore` 必须排除 `.git/`、`.omc/`、`.omx/`、`.worktree/`、本地 Evidence、coverage、cache 和构建输出。Docker image build context 不得包含 Git metadata 或 Agent 运行态文件。

Release 与 Evidence 命令允许通过 bind mount 读取工作区中的 Git metadata，因为 release manifest、tree state、commit、tree SHA 和审计锚点需要真实仓库状态。这个例外只属于运行时挂载边界，不能反向扩大 image build context。

## 环境变量 pass-through

Docker Toolchain Runtime 必须显式传递并记录以下环境变量语义：

- `XLIB_CONTEXT`：区分 `ci_pull_request`、`release_verify` 等执行语境。
- `GOWORK`：release、Harness、CI 和 downstream verification 中必须为 `off`。
- `VERSION`：release/preflight 命令使用的版本输入。
- `DOWNSTREAM`：integration 或下游验证选择的目标库。
- `XLIB_ENABLE_VULNCHECK`：仅在显式为 `1` 时启用漏洞扫描工具链。
- `CI`、`GITHUB_ACTIONS`：仅传递执行语境标志，确保 GitHub Actions 内的容器化 `make ci` 仍按 CI 语义跳过本地 hooks 配置检查；本地未设置时保持空值，继续执行 `doctor-hooks-local`。
- `GIT_CONFIG_COUNT=1`、`GIT_CONFIG_KEY_0=safe.directory`、`GIT_CONFIG_VALUE_0=/workspace`：固定注入容器内 Git trust 配置，使 bind-mounted 工作区在 GitHub Actions 和本地 Docker 中都能执行 `git ls-files`、`git archive`、release manifest 和治理检查；该配置只信任 `/workspace`，不得扩大到宿主其他路径。

未列入 contract 的私密变量不得默认传入容器；需要时必须在调用点显式声明并脱敏记录。

## BuildKit、cache 与 volume

CI workflow 必须启用 BuildKit/buildx。Docker Compose 和 gate 脚本应使用独立 Go build cache 与 module cache volume，避免把宿主缓存路径写入 release Evidence。cache 只能改善性能，不能成为 gate 通过的前置条件；cache miss 后仍必须可重建并运行同一组 Makefile gate。

## Runtime image 边界

`Dockerfile` 提供的是 toolchain image，不是生产服务镜像或二进制发布镜像。它必须包含 Go、make、git、jq、curl、证书、`python3-yaml` 和 gate 必需工具；`python3-yaml` 是 `rules-verify` / `render-domain-rules` 读取 YAML registry 的固定依赖。镜像必须通过 `git config --global --add safe.directory /workspace` 固定容器工作区 trust，其中 lint 基线为 `golangci-lint v2.1.6`。`govulncheck v1.1.4` 可以安装在镜像中作为固定工具链事实，但执行仍只能由 `XLIB_ENABLE_VULNCHECK`、一周窗口状态和 `XLIB_FORCE_VULNCHECK` 控制；CI、release 和 Docker Contract 默认不得因此每次运行漏洞扫描。该 image 不得承诺应用运行时 SLA、端口、数据库、队列或生产部署拓扑。

## 下游模板继承

`scripts/render_template.sh` 渲染出的 `kernel`、`configx`、`observex`、`testkitx`、`postgresx`、`redisx`、`kafkax`、`taosx`、`ossx`、`clickhousex` 等目标必须继承 Docker contract 文件和 Makefile 目标：`Dockerfile`、`docker-compose.yml`、`.dockerignore`、`.devcontainer/devcontainer.json`、`scripts/docker/check_toolchain.sh`、`scripts/docker/docker_gate.sh`，以及 `docker-toolchain-check`、`docker-build`、`docker-build-check`、`docker-shell`、`docker-ci`、`docker-release-check`、`docker-release-final-check`、`docker-goalcli`、`docker-goalcli-image`、`docker-goalcli-version`、`docker-runtime-check`、`docker-drift-check`、`docker-contract`。`docs/downstream-matrix.md` 的 `docker_contract_required` 与 `.agent/registries/downstream-adoption-status.yaml` 的 `docker_contract_status` 是采纳状态锚点。

## 常见失败与排查分类

- Docker CLI 缺失：安装 Docker 或在不具备 Docker 的静态检查环境中使用受控的 `XLIB_DOCKER_ALLOW_MISSING=1 make docker-toolchain-check`，不得把缺失环境记录为 release 通过证据。
- Docker daemon 不可用：启动 daemon 后重跑 `make docker-toolchain-check`。
- buildx/BuildKit 缺失：执行 `docker buildx inspect --bootstrap` 或更新 Docker。
- `GOWORK` 未关闭：重跑 `GOWORK=off make docker-ci` 或 `XLIB_CONTEXT=release_verify GOWORK=off make docker-release-check`。
- cache/volume 权限异常：删除 Docker volume 后重建，不能修改源码 gate 放宽规则。
- `rules-verify` 报 `ModuleNotFoundError: No module named 'yaml'`：确认 `Dockerfile` 安装 `python3-yaml`，且下游渲染产物继承同一 toolchain image；不得在 gate 运行中临时 `pip install`。
- 下游渲染漂移：运行 `scripts/check_rendered_template.sh`，确认 Docker 文件和 `make docker-*` targets 未丢失。
- `goalcli doctor` 或 `goalcli score --min 9.8` 失败：先修复 doctor details 和 score 维度；score 低于阈值表示发布质量证据不足，不得用 Docker gate 覆盖。

## 回滚与审计锚点

若 Docker contract 导致 CI 或下游失败，必须成组回滚或修复以下锚点，不能只删除 workflow：

- `Dockerfile`、`docker-compose.yml`、`.dockerignore`、`.devcontainer/devcontainer.json`。
- `scripts/docker/check_toolchain.sh`、`scripts/docker/docker_gate.sh`、Makefile 的 `docker-*` targets。
- `.github/workflows/docker-contract.yml` 与上传的 `release/docker/*.md` artifact。
- `docs/downstream-matrix.md`、`.agent/registries/downstream-adoption-status.yaml`、`.agent/registries/downstream-registry.yaml`。
- Release Evidence：`release/manifest/latest.json`、`release/standard-impact/latest.md`、score 和 workflow 字段。

回滚声明必须说明 Docker Toolchain Runtime 是工具链运行时，不是第二套 gate，并保留 parent plan #62 与相关 Issue 的审计链路。
