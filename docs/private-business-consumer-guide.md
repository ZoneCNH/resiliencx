# L3 私有业务系统消费指南

本文面向 `x.go`、`market-data`、`market-engine`、`macro-data`、`macro-engine` 和 `regime-engine` 等 L3 私有业务系统，说明它们如何消费公开基础库、如何设置规则边界，以及如何在后续更新迭代中保留可验证 Evidence。

L3 私有业务系统可以依赖 `resiliencx` 产出的 L0/L1/L2 基础库，但业务模型、业务 topic/subject、业务 schema、业务策略、客户数据语义、真实凭据和生产部署 wiring 必须留在私有仓库。公开仓库只记录通用规则、抽象影响结论和 owner，不记录业务细节。

## 消费边界

| 项目 | L3 私有业务系统可以做 | 不得做 |
| --- | --- | --- |
| 依赖方向 | 依赖 `kernel`、`configx`、`observex`、`testkitx` 和 L2 适配库的已发布版本 | 要求 `resiliencx` 反向依赖 L3 仓库 |
| 配置注入 | 从授权的 `/home/k8s/secrets/env/*` 或私有配置系统读取生产配置，并转换为基础库显式 Config | 让 L0/L1/L2 基础库直接读取生产密钥路径 |
| 业务语义 | 在私有仓库维护业务 repository、业务 schema、业务指标和策略 | 把业务模型、真实 topic/subject、交易策略或客户数据语义提交到公开基础库 |
| Evidence | 在私有 CI 中保存完整验证证据，对公开仓库只给出脱敏摘要 | 在公开 PR、Issue、日志、release manifest 或文档中暴露真实凭据、连接串或业务敏感字段 |
| 临时适配 | 在私有分支验证 `replace`、RC tag 或兼容层 | 把本地 `replace`、临时 wiring 或私有 hack 合入主干 |

## 私有仓库初始化

私有仓库应在开发机和 CI 环境显式声明私有 module 范围，避免 Go 工具链把私有 module 发送到公共代理或 checksum 服务：

```bash
go env -w GOPRIVATE=github.com/ZoneCNH/x.go,github.com/ZoneCNH/market-*,github.com/ZoneCNH/macro-*,github.com/ZoneCNH/regime-engine
go env -w GONOSUMDB=github.com/ZoneCNH/x.go,github.com/ZoneCNH/market-*,github.com/ZoneCNH/macro-*,github.com/ZoneCNH/regime-engine
```

基础库依赖应优先使用发布 tag。只有定位问题或迁移验证时才允许本地 `replace`，且必须满足：

- `replace` 只存在于临时分支或本地验证上下文。
- 主干、release 分支和 CI 不得提交临时 `replace`。
- 验证结论必须记录目标 tag、基础库 module、owner 和回滚方案。

## 私有 CI 最小 gate

每个 L3 私有业务系统在升级 L0/L1/L2 基础库时，至少运行以下 gate：

```bash
GOWORK=off go test ./...
GOWORK=off go test -race ./...
GOWORK=off go list -m all
GOWORK=off go mod why -m github.com/ZoneCNH/kernel
```

若引入或升级 L1/L2 库，应对相关 module 追加 `go mod why -m`，例如：

```bash
GOWORK=off go mod why -m github.com/ZoneCNH/configx
GOWORK=off go mod why -m github.com/ZoneCNH/redisx
GOWORK=off go mod why -m github.com/ZoneCNH/natsx
```

业务系统还应按自身风险补充 integration、smoke、contract、回放或影子环境验证。公开仓库不得要求私有 CI 输出真实业务样本；公开摘要只记录“已通过/blocked/known gap”、版本、owner 和脱敏原因。

## 密钥与脱敏规则

L3 私有业务系统可以读取授权的 `/home/k8s/secrets/env/*`，但基础库只能接收已经解析后的显式配置对象。不得提交以下内容：

- `.env`、真实连接串、token、AK/SK、数据库账号或生产 endpoint。
- 从 `/home/k8s/secrets/env/*` 读取到的原始内容。
- 带真实业务 topic/subject、客户标识、交易策略或内部指标含义的日志。
- 未脱敏的 Evidence、截图、测试夹具、PR 描述、Issue 或 release note。

脱敏摘要应保留可审计字段：基础库名称、目标版本、验证命令、结论、owner、时间和 known gap。敏感值只允许写成类别，例如“生产连接串已由私有 CI 验证，公开证据已脱敏”。

## 更新迭代流程

1. `resiliencx` 发布标准、模板或 gate 变更后，先判断影响层级：L0、L1、L2 或仅文档治理。
2. 按 L0 -> L1 -> L2 顺序升级基础库版本，优先选择发布 tag，不直接消费未发布主干。
3. L3 私有业务系统创建集成分支，更新 `go.mod`，运行最小 gate 和业务自有 gate。
4. 若验证通过，私有仓库合入并在公开摘要中记录版本、owner 和脱敏 Evidence 结论。
5. 若验证失败，先判断是私有业务适配问题还是基础库标准缺口。业务问题留在私有仓库；标准缺口用抽象语言提交 `resiliencx` ADR、Issue 或标准变更。
6. 公开仓库只记录 downstream sync 结论：已同步、无需同步并说明原因，或 blocked 并给出 owner。

## 例外与债务

P0 规则没有临时例外：基础库不得导入 L3，公开仓库不得包含真实密钥或业务敏感语义，required gate 不得绕过。

非 P0 例外必须记录以下字段，否则只能视为 gap，不能作为完成证据：

- owner
- 过期时间
- 影响仓库和层级
- 例外原因
- 替代 gate 或人工验证证据
- 回滚方案

## 公开摘要模板

公开仓库或跨仓库 release note 只能使用脱敏摘要：

| 库 | 目标版本 | 私有验证 | 状态 | owner | 公开摘要口径 |
| --- | --- | --- | --- | --- | --- |
| `kernel` | `vX.Y.Z` | `GOWORK=off go test ./...` | passed / blocked / known gap | `<owner>` | 不含业务数据的通用结果 |
| `configx` | `vX.Y.Z` | `GOWORK=off go mod why -m github.com/ZoneCNH/configx` | passed / blocked / known gap | `<owner>` | 配置注入已在私有 CI 验证，Evidence 已脱敏 |
| `natsx` | `vX.Y.Z` | private integration smoke | passed / blocked / known gap | `<owner>` | subject 和 payload schema 不公开 |

该模板不得替代私有仓库的完整 CI Evidence；它只用于公开标准源追踪 L3 是否受到影响。
