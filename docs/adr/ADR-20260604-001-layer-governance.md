# ADR-20260604-001：分层治理边界

## 状态

Accepted

## 背景

`resiliencx` 同时承担 Standard Source、Go Reference Template、Generator、Harness 和 Evidence Runtime。下游会继续扩展到 L0、L1、L2 和 L3，如果没有明确边界，公开基础库可能误收业务语义、生产凭据路径或私有运行证据。

## 决策

采用 `docs/standard/layer-governance-rules.md` 作为分层治理规则入口，并把 `docs-check` 纳入漂移检查。公开仓库只承载可复用基础能力；L3 私有业务系统的业务模型、生产 topic、客户数据语义、策略逻辑和真实 secrets 必须保留在私有仓库。

## 约束

- Standard、L0、L1、L2 不得依赖 L3 私有实现。
- L3 私有仓库只消费已发布基础库版本，不把业务 Evidence 写回公开仓库。
- 涉及分层、下游注册、模板生成或发布 gate 的变更必须运行 `GOWORK=off make docs-check`，并按影响范围补充治理、contracts 或 release gate。

## 后果

该 ADR 是分层治理规则的决策记录。后续修改必须同步更新 `docs/standard/layer-governance-rules.md`、`.agent/policies/layer-governance.yaml`、`contracts/layer-governance.schema.json` 和相关 Harness gate，避免文档、schema 与执行检查分叉。
