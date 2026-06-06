# 分层治理规则

本文定义 `resiliencx -> L0 -> L1 -> L2 -> L3` 的规则、管理边界、约束限制和后续更新迭代流程。若本文与更高权威文件冲突，以 `CONSTITUTION.md`、`docs/standard/module-boundary.md`、`.agent/harness/harness.yaml` 和 release Evidence gate 为准。决策背景见 [ADR-20260604-001](../adr/ADR-20260604-001-layer-governance.md)。

## 分层边界

| 层级 | 仓库/库 | 可见性 | 职责 |
| --- | --- | --- | --- |
| Standard/Runtime | `resiliencx` | 可公开 | 标准、模板、generator、Harness、Evidence 和治理规则 |
| L0 | `kernel` | 可公开 | 通用 runtime primitive，不包含 profile runtime 或业务模型 |
| L1 | `configx`、`observex`、`testkitx` | 可公开 | 配置、观测、测试等横切治理能力 |
| L2 | `redisx`、`kafkax`、`postgresx`、`taosx`、`ossx`、`clickhousex`、`natsx` | 可公开 | 具体基础设施适配，不包含业务 repository、业务 schema 或应用策略 |
| L3 | `x.go`、`market-data`、`market-engine`、`macro-data`、`macro-engine`、`regime-engine` | 私有 | 业务组合、业务模型、业务策略、生产配置注入和部署 wiring |

公开仓库只能承载可复用基础能力。凡是暴露业务语义、业务 topic/subject、业务指标、交易策略、生产凭据路径或客户数据语义的内容，必须停留在 L3 私有仓库。

## 依赖规则

- 允许方向只能是 L3 -> L2 -> L1 -> L0 -> Standard。
- `resiliencx` 不依赖 `kernel`、L1/L2 生成库或 `x.go`。
- L0 不依赖 L1/L2 或 `x.go`。
- L1/L2 可以依赖 `kernel` 和更低层公共 contracts，但不得依赖 L3。
- L3 可以组合所有基础库，但不得把业务实现反向复制回基础库。
- 基础库不得读取 `/home/k8s/secrets/env/*`；生产密钥读取和注入只属于 L3 组合层。

私有 Go module 应在开发机或 CI 环境配置 `GOPRIVATE`，例如：

```bash
export GOPRIVATE=github.com/ZoneCNH/x.go,github.com/ZoneCNH/market-*,github.com/ZoneCNH/macro-*,github.com/ZoneCNH/regime-engine
```

## 规则等级

| 等级 | 处理方式 | 规则 |
| --- | --- | --- |
| P0 | fail closed，阻断 release | 基础库不得导入 `x.go` 或 L3；公开仓库不得包含业务模型、业务消息 schema、业务策略、生产密钥或真实连接串；release 完成声明必须有 Evidence；不得绕过 required gate |
| P1 | 本仓库治理阻断 | public API、contracts、Harness、Evidence 或标准文本发生破坏性变化时，必须有版本决策、迁移说明和同步影响判断；新增 L2 适配库必须进入下游矩阵和采纳状态清单 |
| P2 | gap explicit | 下游未验证、未采纳、缺少 Evidence 或私有业务系统未覆盖时，只能记录 blocked、known gap 或待办 owner，不得写成 adopted/passed |

P0 没有临时例外。若业务确实需要改变 P0，需要先修改标准、gate 和 Evidence 协议，并通过 release 验证后才能生效。

## 管理入口

- 标准文本：`docs/standard/**`
- 分层和模块边界：`docs/standard/layering.md`、`docs/standard/module-boundary.md`
- 下游目标：`docs/downstream-matrix.md`
- 同步策略：`docs/downstream-sync-policy.md`
- 采纳状态：`.agent/registries/downstream-adoption-status.yaml`
- 机器 gate：`.agent/harness/harness.yaml`、`cmd/goalcli`
- 规则事实：`.agent/rules/README.md`、生成的 `.agent/rules/registry.yaml`

`.agent/rules/registry.yaml` 是生成结果，不手工编辑。新增或修改治理规则时，优先改标准文档、Harness、`goalcli` 和对应测试，再由规则生成流程同步。

## 更新迭代流程

1. 在 `resiliencx` 提出标准变更、ADR 或 Issue，说明影响层级和是否触发下游同步。
2. 更新 `docs/standard/**`、contracts、模板、Harness 或 `goalcli`，并保持文档默认中文叙述。
3. 更新 `docs/downstream-matrix.md`、`docs/downstream-sync-policy.md` 和 `.agent/registries/downstream-adoption-status.yaml` 中受影响库的状态。
4. 使用 `GOWORK=off` 运行对应 gate，至少覆盖 `docs-check`；触达发布、contracts、security 或 generator 时运行更高等级 gate。
5. 发布 `resiliencx` tag 后，按 L0 -> L1 -> L2 顺序同步基础库。
6. L3 私有业务系统只消费已发布基础库版本，在私有 CI 中验证，不把业务 Evidence 写入公开仓库。
7. release Evidence 必须记录 downstream sync 结论：已同步、无需同步并说明原因，或 blocked 并给出 owner。

## 例外与债务

任何非 P0 例外都必须记录：

- owner
- 过期时间
- 影响仓库和层级
- 例外原因
- 替代 gate 或人工验证证据
- 回滚方案

未记录以上字段的例外只能视为 gap，不能作为完成证据。
