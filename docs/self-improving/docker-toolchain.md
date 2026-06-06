# Docker Toolchain Runtime 自改进记录

Docker Toolchain Runtime 只把既有 Makefile gate 放入统一容器边界执行，不新增第二套质量标准。`scripts/docker/check_toolchain.sh` 是静态契约 oracle，负责检查 Docker 文件、workflow、依赖更新配置、manifest schema、下游模板继承和 `release/docker/toolchain-check.md`。

## 证据流

- `make docker-toolchain-check` 写入宿主工具链报告。
- `make docker-drift-check` 以 `--drift` 模式重放静态契约，防止 Docker 文件、CI、docs、registry 和 manifest 脱节。
- `make docker-runtime-check` 验证 `goalcli-runtime` 容器入口仍能执行既有 goalcli gate。
- `make docker-contract` 汇总 Docker Toolchain Runtime contract gate。
- CI 上传 `release/evidence/docker-toolchain-summary.json` 与 `release/docker/*.md`，作为 Docker contract artifact。

## 下游继承规则

`scripts/check_rendered_template.sh` 必须确认渲染后的下游库继承 Docker contract 文件和全部 Docker Makefile 目标：`docker-toolchain-check`、`docker-build`、`docker-build-check`、`docker-shell`、`docker-ci`、`docker-release-check`、`docker-release-final-check`、`docker-goalcli`、`docker-goalcli-image`、`docker-goalcli-version`、`docker-runtime-check`、`docker-drift-check`、`docker-contract`。

## 维护约束

Docker 依赖升级必须经过 `renovate.json` 和 `.github/dependabot.yml` 的 Docker manager，且保持 `contracts/docker-toolchain.schema.json`、`release/manifest/template.json`、`.agent/registries/*` 和 `.agent/harness/harness.yaml` 同步。任何放宽 gate 的变更都必须解释为什么既有 `make ci`、`make release-check` 与 Harness 语义没有被替代。
