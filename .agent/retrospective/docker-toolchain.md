# Docker Toolchain Runtime 回顾

## 决策

- Docker Toolchain Runtime 定位为工具链运行时，不作为第二套 gate。
- `goalcli-runtime` 是容器内执行 goalcli gate 的唯一入口。
- `docker-drift-check` 保留为静态契约回放，避免 CI、manifest、docs、registry 和下游模板漂移。

## 风险

- 本地缺少 Docker daemon 时只能运行静态 contract，不能证明镜像 build 成功。
- Docker base image digest 变更会影响 release manifest 和 CI artifact，需要通过依赖更新 PR 单独审查。
- downstream 渲染若丢失 Docker targets，会使 `scripts/check_rendered_template.sh` 失败。

## 验证

- `GOWORK=off make docker-toolchain-check`
- `GOWORK=off make docker-drift-check`
- `GOWORK=off make docker-runtime-check`
- `GOWORK=off make docker-contract`

## 后续

保持 `release/evidence/docker-toolchain-summary.json` 与 `release/docker/toolchain-check.md` 在 CI artifact 中可追溯。若 Docker contract 失败，优先修复契约锚点，不得绕过 `make ci`、`make release-check` 或 Harness gate。
