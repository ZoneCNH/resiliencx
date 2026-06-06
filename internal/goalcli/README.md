# internal/goalcli

该目录预留给 `cmd/goalcli` 可复用的内部实现。当前命令入口仍在 `cmd/goalcli`，本占位文件只固定目标结构，不表示已有逻辑迁移完成。

## GoalCLI 同步契约

`cmd/goalcli` 仍是当前权威执行入口；未来如将可复用逻辑迁入 `internal/goalcli`，不得改变 `goalcli` 的单一执行面。任何命令新增、重命名、删除或语义变化，都必须同步 `cmd/goalcli/main.go`、`cmd/goalcli/main_test.go`、`Makefile`、`.agent/registries/*`、`.agent/harness/*`、`docs/standard/goalcli-cli-contract.md`、`.agent/docs/standard/goalcli-mapping.md` 和本文件。

`cmd/goalcli/main_test.go` 必须固定 registry、usage 和 implementation-status 的同步关系；迁移实现时不得让 `commandRegistryRequiredCommands()`、`.agent/registries/command-registry.yaml` 或 `.agent/registries/command-implementation-status.yaml` 任一 surface 单独漂移。

同步后至少运行 `GOWORK=off go test ./cmd/goalcli` 与 `GOWORK=off make docs-check`；涉及 registry 或 Makefile gate 的变更还必须运行 `GOWORK=off make command-registry`、`GOWORK=off make makefile-baseline` 和 `GOWORK=off make cli-contract`。
