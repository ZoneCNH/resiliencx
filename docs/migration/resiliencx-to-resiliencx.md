# resiliencx 到 resiliencx 迁移指南

## 目标

保留 `.agent/traceability/traceability-matrix.md` 中的历史迁移路径，并把当前事实收敛到 `docs/migration/baselib-template-to-xlib-standard.md` 与 `docs/standard/xlib-standard.md`。本文件是兼容性入口：旧身份只允许出现在 ADR、迁移指南、历史变更记录和兼容性说明中。

## 名称规则

| 旧名 | 新名 | 允许保留位置 |
| --- | --- | --- |
| `baselib-template` | `resiliencx` | ADR、迁移指南、CHANGELOG、兼容性说明 |
| `foundationx` | `kernel` | ADR、迁移指南、历史 compatibility note |

## 迁移要求

- README 主标题和主叙事必须使用 `resiliencx`。
- 生成示例默认使用 `kernel` / `github.com/ZoneCNH/kernel`。
- 标准文档必须声明 `resiliencx` 同时承担 Standard Source、Go Reference Template、Generator、Harness 和 Evidence Runtime。
- Go module path、包名、render script 和 CI 迁移由实现 gate 与下游矩阵证明；本文档只定义语义约束。

## 当前标准入口

- [`docs/standard/xlib-standard.md`](../standard/xlib-standard.md)
- [`docs/migration/baselib-template-to-xlib-standard.md`](baselib-template-to-xlib-standard.md)
- [`docs/downstream-matrix.md`](../downstream-matrix.md)

## 验证

- `GOWORK=off go run ./cmd/goalcli traceability-check`
- `GOWORK=off make docs-check`
- `GOWORK=off make adoption-check`
