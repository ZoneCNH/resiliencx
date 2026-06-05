# 规则补丁

规则补丁用于记录 standards、boundary 或 Evidence 的策略更新。每条记录必须关联 REQ/ADR，并说明是否影响 docs-check、boundary、contracts、score 或 release gates。

## 2026-06-04：L0/L1/L2/L3 分层治理边界

- ADR：`docs/adr/ADR-20260604-001-layer-governance.md`
- REQ：`REQ-001`、`REQ-009`、`REQ-010`、`REQ-011`
- 规则：L3 私有；公开基础库不得包含业务模型、业务消息 schema、生产密钥或 `/home/k8s/secrets/env/*` 读取逻辑；新增 L2 适配库必须进入下游矩阵、采纳状态和 gate 锚点。
- Gate 影响：docs-check 增加分层治理文档、ADR、`natsx`、`GOPRIVATE` 和例外字段锚点；boundary、contracts、score 和 release gates 无命令变更；release Evidence 继续记录 downstream sync 结论。
