# 发布

## 版本

Full Goal Runtime v3.1 release candidate。

## 必需 Evidence

- `GOWORK=off go test ./...`
- `GOWORK=off make boundary`
- `GOWORK=off make contracts`
- `GOWORK=off make docs-check`
- `GOWORK=off make security`
- `GOWORK=off make release-check`
- `GOWORK=off make release-final-check`
- `GOWORK=off make release-preflight VERSION=<version>`
- `goalcli score --min 9.8`
- `release/manifest/latest.json`
- `release/manifest/latest.json.sha256`

## 必需工具

- `golangci-lint`
- `goalcli`
- `govulncheck`（仅当 `XLIB_ENABLE_VULNCHECK=1` 且一周窗口到期、状态文件缺失，或 `XLIB_FORCE_VULNCHECK=1` 强制漏洞扫描时必需）

缺少默认必需工具时，相关 gate 必须失败并记录为 blocker；漏洞扫描到期/强制执行时缺少 `govulncheck` 同样必须失败，不能降级为通过。

## 发布规则

没有 Evidence 不得发布。`release/manifest/latest.json` 和 `.sha256` 是生成产物与 CI artifact，不提交到源码历史。不得在源码、README、测试日志、release manifest、PR 描述或 Evidence 中包含 `/home/k8s/secrets/env/*` 的真实内容。
