# 故障排查

## `GOWORK=off is required`

治理和发布目标要求隔离 Go workspace。重新执行命令时在前面加上：

```sh
GOWORK=off make <target>
```

## Makefile 出现 duplicate target warning

应只有一个同名目标定义。若看到 `overriding recipe` 或 `ignoring old recipe`，先检查是否新增了重复 `.PHONY` 或重复 target，再运行：

```sh
GOWORK=off make --warn-undefined-variables governance-check
```

## 缺少 `golangci-lint` 或启用漏洞扫描时缺少 `govulncheck`

本地 `make lint` 依赖 `golangci-lint`。`make security` 默认只运行 secret scan；只有设置 `XLIB_ENABLE_VULNCHECK=1` 且一周窗口到期、状态文件缺失，或设置 `XLIB_FORCE_VULNCHECK=1` 时才依赖 `govulncheck`。CI 默认不安装或访问漏洞库；Security workflow 每周定时强制执行漏洞扫描。启用漏洞扫描时本地可按 workflow 中的固定版本安装 `govulncheck`，或记录为未运行的本地工具缺口。

## release manifest 缺失

`release/manifest/latest.json` 和 `latest.json.sha256` 是生成产物，不应提交。运行 release/evidence 相关命令重新生成，并通过 artifact 或校验和复核。

## 下游 kernel/configx 未通过

不要用本仓库的占位文件替代下游证据。需要在真实下游仓库运行采纳/兼容命令，并把输出记录到正式证据链。

## Docker Toolchain Runtime

Docker 相关失败按以下分类排查：

- CLI 缺失：安装 Docker；仅静态检查环境可用 `XLIB_DOCKER_ALLOW_MISSING=1 GOWORK=off make docker-toolchain-check` 记录环境不可用，不得当作 release 通过证据。
- daemon 不可用：启动 Docker daemon 后重跑 `GOWORK=off make docker-toolchain-check`。
- buildx/BuildKit 缺失：运行 `docker buildx inspect --bootstrap` 或升级 Docker。
- `GOWORK` 未关闭：使用 `GOWORK=off make docker-ci` 或 `XLIB_CONTEXT=release_verify GOWORK=off make docker-release-check`。
- Git 报 `dubious ownership`：确认 Docker image 包含 `safe.directory /workspace`，且 `scripts/docker/docker_gate.sh` 与 `docker-shell` 注入 `GIT_CONFIG_KEY_0=safe.directory` / `GIT_CONFIG_VALUE_0=/workspace`；不得把 trust 放宽到宿主根目录。
- `rules-verify` 报 `ModuleNotFoundError: No module named 'yaml'`：确认 Docker image 安装 `python3-yaml`；该依赖用于 `verify_rules.py` 读取 YAML registry，不应在 gate 运行时临时安装。
- 下游模板漂移：确认渲染产物包含 `Dockerfile`、`docker-compose.yml`、`.dockerignore`、`.devcontainer/devcontainer.json`、`scripts/docker/docker_gate.sh`，以及 `docker-toolchain-check`、`docker-build`、`docker-build-check`、`docker-shell`、`docker-ci`、`docker-release-check`、`docker-release-final-check`、`docker-goalcli`、`docker-goalcli-image`、`docker-goalcli-version`、`docker-runtime-check`、`docker-drift-check`、`docker-contract`。
