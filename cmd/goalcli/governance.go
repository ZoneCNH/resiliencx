package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ZoneCNH/xlib-standard/pkg/templatex"
)

const (
	// projectReleaseVersion 引用 templatex.Version 作为唯一版本号来源。
	projectReleaseVersion    = templatex.Version
	governanceRuntimeVersion = "v2.9.3"
)

type gateReport struct {
	Command string   `json:"command"`
	Status  string   `json:"status"`
	Details []string `json:"details,omitempty"`
	Gaps    []string `json:"gaps,omitempty"`
}

func emitReport(stdout io.Writer, command, status string, details []string, gaps []string) int {
	report := gateReport{Command: command, Status: status, Details: details, Gaps: gaps}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		write(stdout, "{\"command\":%q,\"status\":%q}\n", command, status)
	} else {
		write(stdout, "%s\n", data)
	}
	if status == "passed" {
		return 0
	}
	return 1
}

func runVersion(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("version", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("version", err, stderr)
	}
	return emitReport(stdout, "version", "passed", []string{"xlib-standard release " + projectReleaseVersion, "goalcli governance runtime " + governanceRuntimeVersion, "goalcli governance CLI available"}, nil)
}

func runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("doctor", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("doctor", err, stderr)
	}
	required := []string{
		".agent/harness.yaml",
		".agent/issue-registry.yaml",
		".agent/command-registry.yaml",
		".agent/makefile-target-registry.yaml",
		".agent/makefile-baseline.yaml",
		"docs/standard/goalcli-cli-contract.md",
		"contracts/goalcli-report.schema.json",
		"Makefile",
	}
	if isXlibStandardSourceModule() {
		required = append([]string{"docs/goal/goal.md"}, required...)
	}
	var gaps []string
	for _, path := range required {
		if !fileExists(path) {
			gaps = append(gaps, "missing "+path)
		}
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: doctor found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "doctor", "failed", nil, gaps)
	}
	details := []string{"required governance files are present"}
	details = append(details, hooksStatusDetail())
	return emitReport(stdout, "doctor", "passed", details, nil)
}

// hooksStatusDetail 返回 git hooks 启用状态作为 informational details。
// 不影响 doctor 的 pass/fail：CI 环境无须本地 hooks；本地环境若未启用，
// 提示运行 make install-hooks。对应 .agent/standard/goal-runtime-canonical.md
// 中的 RULE-WORKTREE-001 / RULE-SECRET-001 本地防线。
func hooksStatusDetail() string {
	if !fileExists(".githooks/pre-commit") {
		return "hooks: .githooks/pre-commit 不存在（仓库可能未初始化 hooks 目录）"
	}
	current := strings.TrimSpace(gitOutput("config", "--get", "core.hooksPath"))
	if current == ".githooks" {
		return "hooks: ✅ core.hooksPath=.githooks 已启用"
	}
	if current == "" {
		return "hooks: ⚠️  core.hooksPath 未设置，运行 make install-hooks 启用本地 P0 防线"
	}
	return "hooks: ⚠️  core.hooksPath=" + current + "（非 .githooks），本地 P0 防线未启用"
}

func runRegistryCheck(command string, required map[string][]string, stdout io.Writer, stderr io.Writer) int {
	var gaps []string
	for path, needles := range required {
		content, err := os.ReadFile(path)
		if err != nil {
			gaps = append(gaps, "missing "+path)
			continue
		}
		text := string(content)
		for _, needle := range needles {
			if !strings.Contains(text, needle) {
				gaps = append(gaps, path+" missing "+needle)
			}
		}
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: %s found %d gap(s)\n", command, len(gaps))
		return emitReport(stdout, command, "failed", nil, gaps)
	}
	return emitReport(stdout, command, "passed", []string{"registry contract satisfied"}, nil)
}

// runRulesConsistencyCheck 校验 .agent/standard/goal-runtime-canonical.md（叙事层）
// 与 .agent/rules/iron-rules.md（机器层）引用的 RULE-* 编号集合一致，
// 并要求两侧引用的所有 RULE-* 都在 .agent/rules/registry.yaml 中登记。
//
// 用途：PR #36 引入双 SSOT 后，防止两份文档的铁律编号映射悄然漂移。
func runRulesConsistencyCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("rules-consistency-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("rules-consistency-check", err, stderr)
	}

	canonicalPath := ".agent/standard/goal-runtime-canonical.md"
	ironPath := ".agent/rules/iron-rules.md"
	registryPath := ".agent/rules/registry.yaml"

	canonical, err := os.ReadFile(canonicalPath)
	if err != nil {
		write(stderr, "ERROR: read %s: %v\n", canonicalPath, err)
		return 1
	}
	iron, err := os.ReadFile(ironPath)
	if err != nil {
		write(stderr, "ERROR: read %s: %v\n", ironPath, err)
		return 1
	}
	registry, err := os.ReadFile(registryPath)
	if err != nil {
		write(stderr, "ERROR: read %s: %v\n", registryPath, err)
		return 1
	}

	canonRules := extractCanonicalIronRuleIDs(string(canonical))
	ironRules := extractIronRulesIDs(string(iron))
	registryRules := extractRegistryRuleIDs(string(registry))

	var gaps []string

	if len(canonRules) == 0 {
		gaps = append(gaps, fmt.Sprintf("%s: 未发现八条铁律段的 RULE-* 引用", canonicalPath))
	}
	if len(ironRules) == 0 {
		gaps = append(gaps, fmt.Sprintf("%s: 未发现七律段的 RULE-* 引用", ironPath))
	}
	if len(registryRules) == 0 {
		gaps = append(gaps, fmt.Sprintf("%s: 未发现 RULE-* 登记", registryPath))
	}

	for id := range canonRules {
		if !registryRules[id] {
			gaps = append(gaps, fmt.Sprintf("%s 引用 %s 未在 %s 登记", canonicalPath, id, registryPath))
		}
	}
	for id := range ironRules {
		if !registryRules[id] {
			gaps = append(gaps, fmt.Sprintf("%s 引用 %s 未在 %s 登记", ironPath, id, registryPath))
		}
	}

	for id := range canonRules {
		if !ironRules[id] {
			gaps = append(gaps, fmt.Sprintf("漂移：%s 引用 %s 但 %s 未引用", canonicalPath, id, ironPath))
		}
	}

	if len(gaps) > 0 {
		write(stderr, "ERROR: rules-consistency-check found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "rules-consistency-check", "failed", nil, gaps)
	}
	details := []string{
		fmt.Sprintf("canonical=%d iron=%d registry=%d 引用集合一致", len(canonRules), len(ironRules), len(registryRules)),
	}
	return emitReport(stdout, "rules-consistency-check", "passed", details, nil)
}

// extractCanonicalIronRuleIDs 抓取 canonical 的"八条铁律"段表格中
// 形如 `| RULE-XXX-NNN |` 的 ID。仅在该段内（首个 `## 1.` 之后到下一个 `##` 之前）。
func extractCanonicalIronRuleIDs(text string) map[string]bool {
	out := map[string]bool{}
	startIdx := strings.Index(text, "## 1.")
	if startIdx < 0 {
		return out
	}
	section := text[startIdx:]
	if nextIdx := strings.Index(section[5:], "\n## "); nextIdx >= 0 {
		section = section[:nextIdx+5]
	}
	re := regexp.MustCompile(`\|\s*(RULE-[A-Z]+(?:-[A-Z]+)*-\d+)\s*\|`)
	for _, m := range re.FindAllStringSubmatch(section, -1) {
		out[m[1]] = true
	}
	return out
}

// extractIronRulesIDs 抓取 iron-rules 的"七律"段中括号内引用的 RULE-* ID。
// 仅在 `## 七律` 段内。
func extractIronRulesIDs(text string) map[string]bool {
	out := map[string]bool{}
	startIdx := strings.Index(text, "## 七律")
	if startIdx < 0 {
		return out
	}
	section := text[startIdx:]
	if nextIdx := strings.Index(section[6:], "\n## "); nextIdx >= 0 {
		section = section[:nextIdx+6]
	}
	re := regexp.MustCompile(`RULE-[A-Z]+(?:-[A-Z]+)*-\d+`)
	for _, m := range re.FindAllString(section, -1) {
		out[m] = true
	}
	return out
}

// extractRegistryRuleIDs 抓取 registry.yaml 中所有 `- id: RULE-XXX-NNN` 行的 ID。
func extractRegistryRuleIDs(text string) map[string]bool {
	out := map[string]bool{}
	re := regexp.MustCompile(`(?m)^\s*-\s*id:\s*(RULE-[A-Z]+(?:-[A-Z]+)*-\d+)`)
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		out[m[1]] = true
	}
	return out
}

// --- 共享辅助函数 ---

type internalCommandFlagSpec struct {
	boolFlags   []string
	stringFlags []string
}

func validateInternalCommandArgs(command string, args []string, spec internalCommandFlagSpec) error {
	flags := flag.NewFlagSet("goalcli "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	for _, name := range spec.boolFlags {
		flags.Bool(name, false, "")
	}
	for _, name := range spec.stringFlags {
		flags.String(name, "", "")
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected positional argument %q", flags.Arg(0))
	}
	return nil
}

func invalidInternalArgsExit(command string, err error, stderr io.Writer) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	write(stderr, "ERROR: %s invalid arguments: %v\n", command, err)
	return 2
}

func isXlibStandardSourceModule() bool {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return true
	}
	sourceModule := strings.Join([]string{"github.com", "ZoneCNH", "xlib" + "-standard"}, "/")
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1] == sourceModule
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func envDefault(name, fallbackValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallbackValue
}

func fallback(value, fallbackValue string) string {
	if value == "" {
		return fallbackValue
	}
	return value
}

func validContext(value string) bool {
	for _, context := range validExecutionContexts {
		if value == context {
			return true
		}
	}
	return false
}

var validExecutionContexts = []string{"local_write", "local_readonly", "ci_pull_request", "ci_main_verify", "release_verify"}

var commandRegistryCommands = []string{
	"version",
	"doctor",
	"minimal-kernel",
	"main-guard",
	"worktree-guard",
	"worktree-check",
	"context-check",
	"spec-check",
	"design-check",
	"task-check",
	"pr-check",
	"evidence-check",
	"done-assertion",
	"cli-contract",
	"issue-registry",
	"command-registry",
	"makefile-baseline",
	"context-profile",
	"context-profile-check",
	"context-schema-check",
	"context-lite",
	"context-standard",
	"context-full",
	"context-release",
	"context-fast-check",
	"context-standard-check",
	"context-full-check",
	"agent-team-contract",
	"scope-lock",
	"pr-template",
	"acceptance-matrix",
	"runtime-health",
	"goal-runtime",
	"goal-acceptance",
	"goal-delivery",
	"goal-handover",
	"goal-downstream-adoption",
	"goal-certify",
	"goal-runtime-final",
	"naming",
	"upgrade-standard",
	"conformance-profile",
	"downstream-registry",
	"self-healing-skeleton",
	"policy-schema",
	"github-settings",
	"github-governance",
	"governance-fixture-test",
	"toolchain",
	"evidence-artifacts",
	"install-runtime",
	"upgrade-runtime",
	"release-ready",
	"evidence-replay",
	"attest-conformance",
	"pack-standard",
	"pack-gate",
	"pack-evidence",
	"runtime-file-ownership",
	"downstream-baseline",
	"downstream-adoption",
	"autoresearch",
	"changelog",
	"supply-chain",
	"execution-context",
	"boundary",
	"contracts",
	"dependency-check",
	"docs-check",
	"evidence",
	"manifest",
	"integration",
	"release-evidence-check",
	"release-evidence-checksum-check",
	"release-evidence-hash",
	"release-final-check",
	"render-check",
	"rules-verify",
	"score",
	"secrets",
	"security",
	"standard-impact-check",
}

func commandRegistryRequiredCommands() []string {
	return append([]string(nil), commandRegistryCommands...)
}

func requiredCommandRegistryNeedles() []string {
	needles := make([]string, 0, len(commandRegistryCommands))
	for _, command := range commandRegistryCommands {
		needles = append(needles, "name: "+command)
	}
	return needles
}

func gitOutput(args ...string) string {
	cmd := exec.Command("git", args...)
	data, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func flagValue(args []string, name string, fallbackValue string) string {
	for i, arg := range args {
		if arg == "--"+name && i+1 < len(args) {
			return args[i+1]
		}
		prefix := "--" + name + "="
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return fallbackValue
}

func flagProvided(args []string, name string) bool {
	for _, arg := range args {
		if arg == "--"+name || strings.HasPrefix(arg, "--"+name+"=") {
			return true
		}
	}
	return false
}
