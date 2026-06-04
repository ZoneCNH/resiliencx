package main

import "io"

// Command 封装一个 CLI 命令的路由信息。
// Run 的签名：Run(name, args, stdin, stdout, stderr)。
// name 是命令名（args[0]），args 是剩余参数（原 args[1:]）。
// 需要命令名的处理器（如别名命令）可从 name 获取。
type Command struct {
	Run func(name string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int
}

// commandRegistry 是所有命令名到 Command 的映射表，替代原 switch-case 路由。
var commandRegistry = map[string]Command{
	// ── 版本与诊断 ──
	"version": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runVersion(args, stdout, stderr)
	}},
	"doctor": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runDoctor(args, stdout, stderr)
	}},

	// ── 守卫与检查 ──
	"main-guard": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runMainGuard(args, stdout, stderr)
	}},
	"worktree-guard": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runWorktreeGuard(args, stdout, stderr)
	}},
	"worktree-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runWorktreeCheck(args, stdout, stderr)
	}},
	"context-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runContextCheck(args, stdout, stderr)
	}},
	"spec-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runSpecCheck(args, stdout, stderr)
	}},
	"design-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runDesignCheck(args, stdout, stderr)
	}},
	"task-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runTaskCheck(args, stdout, stderr)
	}},
	"pr-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runPRCheck(args, stdin, stdout, stderr)
	}},
	"evidence-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runEvidenceCheck(args, stdout, stderr)
	}},

	// ── 治理与注册 ──
	"cli-contract": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runCLIContract(args, stdout, stderr)
	}},
	"issue-registry": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runIssueRegistry(args, stdout, stderr)
	}},
	"command-registry": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runCommandRegistry(args, stdout, stderr)
	}},
	"makefile-baseline": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runMakefileBaseline(args, stdout, stderr)
	}},

	// ── 上下文配置 ──
	"context-profile": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runContextProfile(args, stdout, stderr)
	}},
	"context-profile-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runContextProfileCheck("context-profile-check", args, stdout, stderr)
	}},
	"context-schema-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runContextProfileCheck("context-schema-check", args, stdout, stderr)
	}},
	// 上下文配置别名（共享同一个处理器）
	"context-lite":           contextProfileAliasCommand,
	"context-standard":       contextProfileAliasCommand,
	"context-full":           contextProfileAliasCommand,
	"context-release":        contextProfileAliasCommand,
	"context-fast-check":     contextProfileAliasCommand,
	"context-standard-check": contextProfileAliasCommand,
	"context-full-check":     contextProfileAliasCommand,

	// ── 债务分析 ──
	"debt": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runDebt(args, stdout, stderr)
	}},
	"architecture":        debtAliasCommand("architecture", "enforce"),
	"domain":              debtAliasCommand("domain", "enforce"),
	"docs-drift":          debtAliasCommand("docs", "warn"),
	"dependency-debt":     debtAliasCommand("dependency", "warn"),
	"testing-debt":        debtAliasCommand("testing", "warn"),
	"implementation-debt": debtAliasCommand("implementation", "observe"),
	"security-debt":       debtAliasCommand("security", "warn"),
	"downstream-debt":     debtAliasCommand("downstream", "warn"),
	"debt-evidence": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runDebtEvidence(args, stdout, stderr)
	}},

	// ── 目标运行时命令 ──
	"goal-acceptance":          goalRuntimeCommand,
	"goal-delivery":            goalRuntimeCommand,
	"goal-handover":            goalRuntimeCommand,
	"goal-downstream-adoption": goalRuntimeCommand,
	"goal-certify":             goalRuntimeCommand,
	"goal-runtime-final":       goalRuntimeCommand,

	// ── 分数与规则 ──
	"score": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runScore(args, stdout, stderr)
	}},
	"rules-consistency-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runRulesConsistencyCheck(args, stdout, stderr)
	}},
	"traceability-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runTraceabilityCheck(args, stdout, stderr)
	}},
	"self-improving-check": selfImprovingCheckCommand,
	"retro-check":          selfImprovingCheckCommand,

	// ── 计划命令（尚未实现，输出 planned 状态） ──
	"minimal-kernel":          plannedCommand,
	"done-assertion":          plannedCommand,
	"agent-team-contract":     plannedCommand,
	"scope-lock":              plannedCommand,
	"pr-template":             plannedCommand,
	"acceptance-matrix":       plannedCommand,
	"runtime-health":          plannedCommand,
	"goal-runtime":            plannedCommand,
	"naming":                  plannedCommand,
	"upgrade-standard":        plannedCommand,
	"conformance-profile":     plannedCommand,
	"downstream-registry":     plannedCommand,
	"self-healing-skeleton":   plannedCommand,
	"policy-schema":           plannedCommand,
	"github-settings":         plannedCommand,
	"toolchain":               plannedCommand,
	"evidence-artifacts":      plannedCommand,
	"install-runtime":         plannedCommand,
	"upgrade-runtime":         plannedCommand,
	"release-ready":           plannedCommand,
	"evidence-replay":         plannedCommand,
	"attest-conformance":      plannedCommand,
	"pack-standard":           plannedCommand,
	"pack-gate":               plannedCommand,
	"pack-evidence":           plannedCommand,
	"runtime-file-ownership":  plannedCommand,
	"downstream-baseline":     plannedCommand,
	"downstream-adoption":     plannedCommand,
	"autoresearch":            plannedCommand,
	"changelog":               plannedCommand,
	"github-governance":       plannedCommand,
	"governance-fixture-test": plannedCommand,
	"supply-chain":            plannedCommand,
	"execution-context":       plannedCommand,

	// ── 外部脚本命令 ──
	"boundary":                     {Run: externalCmd("./scripts/check_boundary.sh")},
	"contracts":                    {Run: externalCmd("./scripts/check_contracts.sh")},
	"dependency-check":             {Run: externalCmd("./scripts/check_dependency_diff.sh")},
	"docs-check":                   {Run: externalCmd("./scripts/check_docs.sh")},
	"evidence":                     {Run: externalCmd("go", "run", "./internal/tools/releasemanifest", "--out", "release/manifest/latest.json")},
	"manifest":                     {Run: externalCmd("go", "run", "./internal/tools/releasemanifest", "--out", "release/manifest/latest.json")},
	"integration":                  {Run: externalCmd("./scripts/run_integration.sh")},
	"debt-evidence-checksum-check": {Run: externalCmd("./scripts/hash_release_evidence.sh", "--check", "release/debt/latest.json", "release/debt/latest.json.sha256")},
	"debt-evidence-hash":           {Run: externalCmd("./scripts/hash_release_evidence.sh", "release/debt/latest.json", "release/debt/latest.json.sha256")},
	"release-evidence-check":       {Run: externalCmd("./scripts/check_release_evidence.sh")},
	"release-evidence-checksum-check": {Run: func(_ string, _ []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runExternal(stdin, stdout, stderr, "./scripts/hash_release_evidence.sh", "--check")
	}},
	"release-evidence-hash": {Run: func(_ string, _ []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runExternal(stdin, stdout, stderr, "./scripts/hash_release_evidence.sh")
	}},
	"release-final-check": {Run: externalCmd("make", "release-final-check")},
	"render-check": {Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runExternal(stdin, stdout, stderr, "./scripts/check_rendered_template.sh", args...)
	}},
	"rules-verify": {Run: externalCmd("python3", "scripts/verify_rules.py")},
	"secrets":      {Run: externalCmd("./scripts/check_secrets.sh")},
	"security": {Run: func(_ string, _ []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runExternalSequence(stdin, stdout, stderr,
			externalCommand{name: "govulncheck", args: []string{"./..."}},
			externalCommand{name: "./scripts/check_secrets.sh"},
		)
	}},
	"standard-impact-check": {Run: externalCmd("./scripts/check_standard_impact.sh")},
}

// ── 复用的 Command 实例 ──

// contextProfileAliasCommand 处理所有上下文配置别名。
var contextProfileAliasCommand = Command{Run: func(name string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return runContextProfileAlias(name, args, stdout, stderr)
}}

// goalRuntimeCommand 处理所有目标运行时命令。
var goalRuntimeCommand = Command{Run: func(name string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return runGoalRuntimeCommand(name, args, stdout, stderr)
}}

// plannedCommand 处理所有计划命令。
var plannedCommand = Command{Run: func(name string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return runPlannedCommand(name, args, stdout, stderr)
}}

// selfImprovingCheckCommand 处理 self-improving-check 和 retro-check。
var selfImprovingCheckCommand = Command{Run: func(name string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return runSelfImprovingCheck(name, args, stdout, stderr)
}}

// debtAliasCommand 生成债务别名命令的 Command 实例。
func debtAliasCommand(section, mode string) Command {
	return Command{Run: func(_ string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runDebtAlias(section, mode, args, stdout, stderr)
	}}
}

// externalCmd 生成外部命令的 Run 函数。
func externalCmd(name string, extraArgs ...string) func(string, []string, io.Reader, io.Writer, io.Writer) int {
	return func(_ string, _ []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		return runExternal(stdin, stdout, stderr, name, extraArgs...)
	}
}

// lookupCommand 从注册表中查找命令。第二个返回值表示是否找到。
func lookupCommand(name string) (Command, bool) {
	cmd, ok := commandRegistry[name]
	return cmd, ok
}
