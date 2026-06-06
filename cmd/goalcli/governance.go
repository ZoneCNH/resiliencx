package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ZoneCNH/resiliencx/internal/validation"
)

const (
	projectReleaseVersion    = "v0.4.14"
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
	return emitReport(stdout, "version", "passed", []string{"resiliencx release " + projectReleaseVersion, "goalcli governance runtime " + governanceRuntimeVersion, "goalcli governance CLI available"}, nil)
}

func runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("doctor", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("doctor", err, stderr)
	}
	required := []string{
		".agent/harness/harness.yaml",
		".agent/index.yaml",
		".agent/registries/issue-registry.yaml",
		".agent/registries/command-registry.yaml",
		".agent/registries/makefile-target-registry.yaml",
		".agent/registries/makefile-baseline.yaml",
		".github/workflows/adoption-check.yml",
		"mk/governance.mk",
		"docs/standard/goalcli-cli-contract.md",
		"contracts/goalcli-report.schema.json",
		"Makefile",
	}
	if isXlibStandardSourceModule() {
		required = append([]string{"docs/goal/goal.md"}, required...)
		required = append(required,
			"docs/standard/docker-toolchain-standard.md",
			"contracts/docker-toolchain.schema.json",
			"Dockerfile",
			".dockerignore",
			"docker-compose.yml",
			".devcontainer/devcontainer.json",
			"scripts/docker/check_toolchain.sh",
			"scripts/docker/docker_gate.sh",
		)
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
// 提示运行 make install-hooks。对应 .agent/runtime/standard/goal-runtime-canonical.md
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

func runMainGuard(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli main-guard", flag.ContinueOnError)
	flags.SetOutput(stderr)
	context := flags.String("context", envDefault("XLIB_CONTEXT", "local_write"), "execution context")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if !validContext(*context) {
		write(stderr, "ERROR: invalid context %q\n", *context)
		return 2
	}
	branch := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if *context == "local_write" && (branch == "main" || branch == "master") {
		return emitReport(stdout, "main-guard", "failed", nil, []string{"local_write is forbidden on " + branch})
	}
	return emitReport(stdout, "main-guard", "passed", []string{"context=" + *context, "branch=" + fallback(branch, "unknown")}, nil)
}

func runWorktreeGuard(args []string, stdout io.Writer, stderr io.Writer) int {
	return runWorktreeGate("worktree-guard", args, stdout, stderr)
}

func runWorktreeCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	return runWorktreeGate("worktree-check", args, stdout, stderr)
}

func runWorktreeGate(command string, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	context := flags.String("context", envDefault("XLIB_CONTEXT", "local_write"), "execution context")
	flags.Bool("json", false, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		write(stderr, "ERROR: %s invalid arguments: unexpected positional argument %q\n", command, flags.Arg(0))
		return 2
	}
	if !validContext(*context) {
		write(stderr, "ERROR: invalid context %q\n", *context)
		return 2
	}
	details, gaps := evaluateWorktreeGate(*context)
	if len(gaps) > 0 {
		return emitReport(stdout, command, "failed", details, gaps)
	}
	return emitReport(stdout, command, "passed", details, nil)
}

func evaluateWorktreeGate(context string) ([]string, []string) {
	top := gitOutput("rev-parse", "--show-toplevel")
	common := gitOutput("rev-parse", "--path-format=absolute", "--git-common-dir")
	branch := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	isWorkerTree := strings.Contains(top, string(filepath.Separator)+".worktree"+string(filepath.Separator)) || strings.Contains(top, string(filepath.Separator)+".worktrees"+string(filepath.Separator)) || strings.Contains(common, string(filepath.Separator)+"worktrees"+string(filepath.Separator))
	details := []string{"context=" + context, "top=" + fallback(top, "unknown"), "branch=" + fallback(branch, "unknown")}
	if context == "local_write" {
		if branch == "main" || branch == "master" {
			return details, []string{"local_write is forbidden on " + branch}
		}
		if !isWorkerTree {
			return details, []string{"local_write requires a worker worktree"}
		}
	}
	return details, nil
}

func runContextCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("context-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("context-check", err, stderr)
	}
	required := []string{"docs/goal", "docs/goal/goal.md"}
	var gaps []string
	for _, path := range required {
		if !fileExists(path) {
			gaps = append(gaps, "missing "+path)
		}
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: context-check found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "context-check", "failed", nil, gaps)
	}
	return emitReport(stdout, "context-check", "passed", []string{"docs/goal context is present"}, nil)
}

func runSpecCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("spec-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("spec-check", err, stderr)
	}
	if !fileExists("docs") {
		write(stderr, "ERROR: spec-check found 1 gap(s)\n")
		return emitReport(stdout, "spec-check", "failed", nil, []string{"missing docs"})
	}
	found := false
	var gaps []string
	paths, err := trackedDocsMarkdownFiles()
	if err != nil {
		gaps = append(gaps, "scan docs: "+err.Error())
	}
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			gaps = append(gaps, "read "+path+": "+readErr.Error())
			continue
		}
		if strings.Contains(string(data), "REQ-") {
			found = true
		}
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: spec-check found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "spec-check", "failed", nil, gaps)
	}
	details := []string{fmt.Sprintf("scanned_markdown=%d", len(paths))}
	if !found {
		details = append(details, "warning: no docs markdown file contains REQ-")
	}
	return emitReport(stdout, "spec-check", "passed", details, nil)
}

func trackedDocsMarkdownFiles() ([]string, error) {
	out, err := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "docs").Output()
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, path := range strings.Split(string(out), "\x00") {
		if path == "" || filepath.Ext(path) != ".md" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func runDesignCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("design-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("design-check", err, stderr)
	}
	if !fileExists("docs/adr") {
		return emitReport(stdout, "design-check", "passed", []string{"warning: optional docs/adr not present"}, nil)
	}
	return emitReport(stdout, "design-check", "passed", []string{"docs/adr is present"}, nil)
}

func runTaskCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("task-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("task-check", err, stderr)
	}
	if fileExists(".agent/registries/command-registry.yaml") {
		return emitReport(stdout, "task-check", "passed", []string{".agent/registries/command-registry.yaml is present"}, nil)
	}
	if fileExists(".agent/registries/commands.yaml") {
		return emitReport(stdout, "task-check", "failed", nil, []string{"canonical .agent/registries/command-registry.yaml missing; .agent/registries/commands.yaml is compatibility coverage only"})
	}
	return emitReport(stdout, "task-check", "failed", nil, []string{"missing .agent/registries/command-registry.yaml"})
}

func runPRCheck(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli pr-check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	context := flags.String("context", envDefault("XLIB_CONTEXT", "local_write"), "execution context")
	dryRun := flags.Bool("dry-run", false, "")
	flags.Bool("json", false, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		write(stderr, "ERROR: pr-check invalid arguments: unexpected positional argument %q\n", flags.Arg(0))
		return 2
	}
	if !validContext(*context) {
		write(stderr, "ERROR: invalid context %q\n", *context)
		return 2
	}
	if *dryRun {
		return emitReport(stdout, "pr-check", "passed", []string{"mode=dry-run", "context=" + *context, "delegates=worktree-check,lint,test"}, nil)
	}
	if details, gaps := evaluateWorktreeGate(*context); len(gaps) > 0 {
		return emitReport(stdout, "pr-check", "failed", details, gaps)
	}
	if code := runExternal(stdin, stderr, stderr, "make", "lint"); code != 0 {
		return emitReport(stdout, "pr-check", "failed", nil, []string{fmt.Sprintf("make lint exited %d", code)})
	}
	if code := runExternal(stdin, stderr, stderr, "make", "test"); code != 0 {
		return emitReport(stdout, "pr-check", "failed", nil, []string{fmt.Sprintf("make test exited %d", code)})
	}
	return emitReport(stdout, "pr-check", "passed", []string{"context=" + *context, "make lint passed", "make test passed"}, nil)
}

func runRegistryCheck(command string, required map[string][]string, stdout io.Writer, stderr io.Writer) int {
	return emitRegistryCheckResult(command, registryContractGaps(required), stdout, stderr)
}

func registryContractGaps(required map[string][]string) []string {
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
	return gaps
}

func emitRegistryCheckResult(command string, gaps []string, stdout io.Writer, stderr io.Writer) int {
	if len(gaps) > 0 {
		write(stderr, "ERROR: %s found %d gap(s)\n", command, len(gaps))
		return emitReport(stdout, command, "failed", nil, gaps)
	}
	return emitReport(stdout, command, "passed", []string{"registry contract satisfied"}, nil)
}

func runEvidenceCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("evidence-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("evidence-check", err, stderr)
	}
	return runRegistryCheck("evidence-check", map[string][]string{
		".agent/evidence/done-assertion.yaml":           {"DONE with evidence", "commit", "gates"},
		".agent/evidence/evidence-artifact-policy.yaml": {"redaction", "sha256", "release/manifest/latest.json"},
		".agent/harness/harness.yaml":                   {"manifest", "checksum", "required_fields"},
		".agent/evidence/evidence-artifacts.yaml":       {"release_evidence", "execution_evidence", "schema:", "contracts/execution-evidence.schema.json"},
		"contracts/execution-evidence.schema.json":      {"evidence_id", "stdout_sha256", "commit", "exit_code", "artifact_path"},
	}, stdout, stderr)
}

func runCLIContract(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("cli-contract", args, internalCommandFlagSpec{boolFlags: []string{"json", "explain"}, stringFlags: []string{"output"}}); err != nil {
		return invalidInternalArgsExit("cli-contract", err, stderr)
	}
	return runRegistryCheck("cli-contract", map[string][]string{
		"docs/standard/goalcli-cli-contract.md":   goalcliCLIContractNeedles(),
		"contracts/goalcli-report.schema.json":    {"command", "status", "details", "gaps"},
		".agent/registries/command-registry.yaml": requiredCommandRegistryNeedles(),
	}, stdout, stderr)
}

func runIssueRegistry(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("issue-registry", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("issue-registry", err, stderr)
	}
	var gaps []string
	appendIssueRegistryGaps(".agent/registries/issue-registry.yaml", &gaps)
	if len(gaps) > 0 {
		write(stderr, "ERROR: issue-registry found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "issue-registry", "failed", nil, gaps)
	}
	return emitReport(stdout, "issue-registry", "passed", []string{"issue registry entries are implemented, unique, and contiguous"}, nil)
}

func runCommandRegistry(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("command-registry", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("command-registry", err, stderr)
	}
	gaps := registryContractGaps(map[string][]string{
		".agent/registries/command-registry.yaml": requiredCommandRegistryNeedles(),
	})
	appendYAMLListDuplicateGaps(".agent/registries/command-registry.yaml", "name", "command", &gaps)
	appendAgentIndexGaps(".agent/index.yaml", &gaps)
	appendGeneratedArtifactClassificationGaps(".agent/index.yaml", ".agent/registries/generated-artifacts.yaml", &gaps)
	appendGeneratedArtifactsGaps(".agent/registries/generated-artifacts.yaml", ".agent/index.yaml", &gaps)
	appendHarnessAliasGaps(".agent/harness/harness.yaml", &gaps)
	appendHarnessGateLinkSemanticsGaps(".agent/harness/harness.yaml", &gaps)
	appendRulesEnforcedByGaps(".agent/rules/registry.yaml", &gaps)
	if len(gaps) > 0 {
		write(stderr, "ERROR: command-registry found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "command-registry", "failed", nil, gaps)
	}
	return emitReport(stdout, "command-registry", "passed", []string{"command registry entries are complete and unique", ".agent/index.yaml control-plane classification satisfied", ".agent generated-artifact and harness gate-link contracts satisfied"}, nil)
}

func runMakefileBaseline(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("makefile-baseline", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("makefile-baseline", err, stderr)
	}
	requiredTargets := requiredMakefileTargets()
	required := map[string][]string{"Makefile": {}, ".agent/registries/makefile-target-registry.yaml": requiredTargets, ".agent/registries/makefile-baseline.yaml": requiredTargets}
	for _, target := range requiredTargets {
		required["Makefile"] = append(required["Makefile"], ".PHONY: "+target, target+":")
	}
	gaps := registryContractGaps(required)
	appendYAMLSequenceDuplicateGaps(".agent/registries/makefile-target-registry.yaml", "targets", "target", &gaps)
	appendYAMLMapSectionDuplicateGaps(".agent/registries/makefile-baseline.yaml", "baseline_targets", "target", &gaps)
	return emitRegistryCheckResult("makefile-baseline", gaps, stdout, stderr)
}

func requiredMakefileTargets() []string {
	requiredTargets := append([]string{"fmt", "vet", "lint", "test", "race", "boundary", "security", "contracts", "schema-check", "docs-check", "rules-verify", "downstream-sync-plan", "adoption-check", "evidence", "score-check", "main-guard", "worktree-guard", "worktree-check", "context-check", "spec-check", "design-check", "task-check", "pr-check", "evidence-check", "cli-contract", "issue-registry", "command-registry", "makefile-baseline", "audit-goal", "dashboard-generate", "governance-check", "p1-governance-check", "execution-context", "p2-runtime-check", "release-check", "release-final-check"}, contextRuntimeTargets()...)
	requiredTargets = append(requiredTargets, dockerMakefileTargets()...)
	return append(requiredTargets, goalcliMakefileTargets()...)
}

func dockerMakefileTargets() []string {
	return []string{
		"docker-toolchain-check",
		"docker-build",
		"docker-build-check",
		"docker-shell",
		"docker-ci",
		"docker-release-check",
		"docker-release-final-check",
		"docker-goalcli",
		"docker-goalcli-image",
		"docker-goalcli-version",
		"docker-runtime-check",
		"docker-drift-check",
		"docker-contract",
	}
}

func goalcliMakefileTargets() []string {
	return []string{
		"goal-acceptance",
		"goal-delivery",
		"goal-handover",
		"goal-downstream-adoption",
		"goal-certify",
		"goal-runtime-final",
	}
}

var contextProfileGates = map[string][]string{
	"lite":     {"governance-check"},
	"standard": {"governance-check", "p1-governance-check", "docs-check"},
	"full":     {"governance-check", "p1-governance-check", "p2-runtime-check"},
	"release":  {"context-full", "integration", "dependency-check", "standard-impact-check", "score-check", "debt-evidence", "evidence", "release-evidence-hash", "release-evidence-check", "release-evidence-checksum-check"},
}

func runContextProfile(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli context-profile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	profile := flags.String("profile", "standard", "context runtime profile")
	flags.Bool("json", false, "")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		write(stderr, "ERROR: context-profile invalid arguments: unexpected positional argument %q\n", flags.Arg(0))
		return 2
	}
	normalized := normalizeContextProfile(*profile)
	gates, ok := contextProfileGates[normalized]
	if !ok {
		write(stderr, "ERROR: invalid context profile %q\n", *profile)
		return 2
	}
	return emitReport(stdout, "context-profile", "passed", contextProfileDetails(normalized, gates), nil)
}

func runContextProfileAlias(command string, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs(command, args, internalCommandFlagSpec{boolFlags: []string{"json", "strict"}}); err != nil {
		return invalidInternalArgsExit(command, err, stderr)
	}
	profile := mapContextAliasToProfile(command)
	gates := contextProfileGates[profile]
	return emitReport(stdout, command, "passed", contextProfileDetails(profile, gates), nil)
}

func runContextProfileCheck(command string, args []string, stdout io.Writer, stderr io.Writer) int {
	profile, err := parseContextProfileCheckProfile(command, args)
	if err != nil {
		return invalidInternalArgsExit(command, err, stderr)
	}
	if profile != "" {
		normalized := normalizeContextProfile(profile)
		if _, ok := contextProfileGates[normalized]; !ok {
			write(stderr, "ERROR: invalid context profile %q\n", profile)
			return 2
		}
	}
	contextTargets := contextRuntimeTargets()
	required := map[string][]string{
		".agent/registries/command-registry.yaml":         requiredCommandRegistryNeedles(),
		".agent/registries/makefile-target-registry.yaml": contextTargets,
		".agent/registries/makefile-baseline.yaml":        contextTargets,
		"docs/standard/goalcli-cli-contract.md":           goalcliCLIContractNeedles(),
		"Makefile":                                        {"release-final-check:", "$(MAKE) context-release"},
	}
	for _, target := range contextTargets {
		required["Makefile"] = append(required["Makefile"], ".PHONY: "+target, target+":")
	}
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
	appendIssueRegistryGaps(".agent/registries/issue-registry.yaml", &gaps)
	if makefile, err := os.ReadFile("Makefile"); err == nil {
		makefileText := string(makefile)
		appendMakefileDuplicateGaps(makefileText, contextTargets, &gaps)
		appendContextProfileContractGaps(makefileText, &gaps)
		appendMakefileTargetDependencyGaps(makefileText, "context-lite", []string{"require-gowork-off", "governance-check"}, []string{"context-profile-check", "main-guard", "worktree-guard", "release-check", "release-final-check"}, &gaps)
		appendMakefileTargetDependencyGaps(makefileText, "context-standard", []string{"require-gowork-off", "governance-check", "p1-governance-check", "docs-check"}, []string{"context-lite", "context-profile-check", "release-check", "release-final-check"}, &gaps)
		appendMakefileTargetDependencyGaps(makefileText, "context-full", []string{"require-gowork-off", "governance-check", "p1-governance-check", "p2-runtime-check"}, []string{"context-standard", "docs-check", "context-profile-check", "release-check", "release-final-check"}, &gaps)
		appendMakefileTargetDependencyGaps(makefileText, "context-release", []string{"require-gowork-off", "context-full", "integration", "dependency-check", "standard-impact-check", "score-check", "debt-evidence"}, []string{"context-standard", "release-check", "release-final-check"}, &gaps)
		appendMakefileTargetForbiddenReferenceGaps(makefileText, "context-release", []string{"release-check", "release-final-check"}, &gaps)
		appendContextProfileDAGGaps(makefileText, &gaps)
		appendReleaseFinalDelegationGaps(makefileText, &gaps)
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: %s found %d gap(s)\n", command, len(gaps))
		return emitReport(stdout, command, "failed", nil, gaps)
	}
	return emitReport(stdout, command, "passed", []string{"context runtime v4.0 profile DAG and registry contract satisfied", ".agent/context not required or claimed", "context profiles reject unknown gates", "context profile Makefile dependencies parse continuations", "context-release excludes release-check and release-final-check", "release-final-check delegates to context-release without self-recursion"}, nil)
}

func parseContextProfileCheckProfile(command string, args []string) (string, error) {
	flags := flag.NewFlagSet("goalcli "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Bool("json", false, "")
	flags.Bool("strict", false, "")
	profile := flags.String("profile", "", "")
	if err := flags.Parse(args); err != nil {
		return "", err
	}
	if flags.NArg() > 0 {
		return "", fmt.Errorf("unexpected positional argument %q", flags.Arg(0))
	}
	return *profile, nil
}

func contextRuntimeTargets() []string {
	return []string{
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
	}
}

func contextProfileDetails(profile string, gates []string) []string {
	return []string{
		"context_runtime=v4.0",
		"profile=" + profile,
		"gates=" + strings.Join(gates, ","),
		"legacy_aliases=context-fast-check,context-standard-check,context-full-check",
		"release_final_delegates=context-release",
	}
}

func normalizeContextProfile(profile string) string {
	switch profile {
	case "fast":
		return "lite"
	default:
		return profile
	}
}

func mapContextAliasToProfile(command string) string {
	switch command {
	case "context-lite", "context-fast-check":
		return "lite"
	case "context-full", "context-full-check":
		return "full"
	case "context-release":
		return "release"
	default:
		return "standard"
	}
}

func makefileTargetBlock(content, target string) string {
	lines := strings.Split(content, "\n")
	var block []string
	inBlock := false
	for _, line := range lines {
		if strings.HasPrefix(line, target+":") {
			inBlock = true
			block = append(block, line)
			continue
		}
		if inBlock {
			if line != "" && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") && strings.Contains(line, ":") {
				break
			}
			block = append(block, line)
		}
	}
	return strings.Join(block, "\n")
}

func appendMakefileDuplicateGaps(content string, targets []string, gaps *[]string) {
	for _, target := range targets {
		if count := makefileTargetDefinitionCount(content, target); count != 1 {
			*gaps = append(*gaps, fmt.Sprintf("Makefile target %s must be defined exactly once, found %d", target, count))
		}
	}
}

func makefileTargetDefinitionCount(content, target string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, target+":") {
			count++
		}
	}
	return count
}

func makefileTargetNames(content string) map[string]bool {
	targets := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		if line == "" || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "#") || strings.Contains(line, ":=") {
			continue
		}
		header := strings.SplitN(line, ":", 2)[0]
		for _, target := range strings.Fields(header) {
			if target != ".PHONY" {
				targets[target] = true
			}
		}
	}
	return targets
}

func appendContextProfileContractGaps(makefileText string, gaps *[]string) {
	makefileTargets := makefileTargetNames(makefileText)
	for profile, gates := range contextProfileGates {
		if profile == "" {
			*gaps = append(*gaps, "context profile name must not be empty")
		}
		if !validContextProfileName(profile) {
			*gaps = append(*gaps, "unknown context profile "+profile)
		}
		for _, gate := range gates {
			if gate == "release-check" || gate == "release-final-check" {
				*gaps = append(*gaps, "context profile "+profile+" must not include "+gate)
			}
			if !makefileTargets[gate] {
				*gaps = append(*gaps, "context profile "+profile+" references unknown Makefile gate "+gate)
			}
		}
	}
	appendContextProfileCycleGaps(gaps)
}

func validContextProfileName(profile string) bool {
	switch profile {
	case "lite", "standard", "full", "release":
		return true
	default:
		return false
	}
}

func appendContextProfileCycleGaps(gaps *[]string) {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(profile string, path []string)
	visit = func(profile string, path []string) {
		if visiting[profile] {
			*gaps = append(*gaps, "context profile DAG cycle: "+strings.Join(append(path, profile), " -> "))
			return
		}
		if visited[profile] {
			return
		}
		visiting[profile] = true
		for _, gate := range contextProfileGates[profile] {
			if next, ok := contextGateProfile(gate); ok {
				visit(next, append(path, profile))
			}
		}
		visiting[profile] = false
		visited[profile] = true
	}
	for profile := range contextProfileGates {
		visit(profile, nil)
	}
}

func appendContextProfileDAGGaps(content string, gaps *[]string) {
	profileTargets := []string{"context-lite", "context-standard", "context-full", "context-release"}
	profileTargetSet := map[string]bool{}
	for _, target := range profileTargets {
		profileTargetSet[target] = true
	}
	allowedLeaf := map[string]bool{
		"require-gowork-off":              true,
		"governance-check":                true,
		"p1-governance-check":             true,
		"docs-check":                      true,
		"p2-runtime-check":                true,
		"integration":                     true,
		"dependency-check":                true,
		"standard-impact-check":           true,
		"score-check":                     true,
		"debt-evidence":                   true,
		"evidence":                        true,
		"release-evidence-hash":           true,
		"release-evidence-check":          true,
		"release-evidence-checksum-check": true,
		"context-profile-check":           true,
		"context-schema-check":            true,
		"context-profile":                 true,
		"context-fast-check":              true,
		"context-standard-check":          true,
		"context-full-check":              true,
	}
	graph := map[string][]string{}
	for _, target := range profileTargets {
		for _, dep := range makefileTargetDependencies(content, target) {
			switch {
			case profileTargetSet[dep] || dep == "release-final-check":
				graph[target] = append(graph[target], dep)
			case allowedLeaf[dep]:
				continue
			default:
				*gaps = append(*gaps, "Makefile "+target+" references unknown context gate "+dep)
			}
		}
	}
	appendMakefileProfileCycleGaps(graph, profileTargets, gaps)
	if makefileGraphReaches(graph, "context-release", "release-final-check", map[string]bool{}) {
		*gaps = append(*gaps, "Makefile context-release must not reach release-final-check")
	}
}

func makefileTargetDependencies(content, target string) []string {
	block := makefileTargetBlock(content, target)
	if block == "" {
		return nil
	}
	lines := strings.Split(block, "\n")
	if len(lines) == 0 {
		return nil
	}
	var dependencyLines []string
	for i, line := range lines {
		if i == 0 {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return nil
			}
			line = parts[1]
		} else {
			if strings.HasPrefix(line, "\t") {
				break
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
		}
		trimmed := strings.TrimSpace(line)
		continued := strings.HasSuffix(trimmed, "\\")
		trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, "\\"))
		if trimmed != "" {
			dependencyLines = append(dependencyLines, trimmed)
		}
		if !continued {
			break
		}
	}
	return strings.Fields(strings.Join(dependencyLines, " "))
}

func appendMakefileProfileCycleGaps(graph map[string][]string, roots []string, gaps *[]string) {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	reported := map[string]bool{}
	var visit func(target string, path []string)
	visit = func(target string, path []string) {
		if visiting[target] {
			cycle := append(path, target)
			key := strings.Join(cycle, " -> ")
			if !reported[key] {
				*gaps = append(*gaps, "Makefile context profile DAG cycle: "+key)
				reported[key] = true
			}
			return
		}
		if visited[target] {
			return
		}
		visiting[target] = true
		for _, dep := range graph[target] {
			if strings.HasPrefix(dep, "context-") {
				visit(dep, append(path, target))
			}
		}
		visiting[target] = false
		visited[target] = true
	}
	for _, root := range roots {
		visit(root, nil)
	}
}

func makefileGraphReaches(graph map[string][]string, start, target string, seen map[string]bool) bool {
	if start == target {
		return true
	}
	if seen[start] {
		return false
	}
	seen[start] = true
	for _, dep := range graph[start] {
		if makefileGraphReaches(graph, dep, target, seen) {
			return true
		}
	}
	return false
}

func contextGateProfile(gate string) (string, bool) {
	switch gate {
	case "context-lite", "context-fast-check":
		return "lite", true
	case "context-standard", "context-standard-check":
		return "standard", true
	case "context-full", "context-full-check":
		return "full", true
	case "context-release":
		return "release", true
	default:
		return "", false
	}
}

func appendMakefileTargetDependencyGaps(content, target string, required []string, forbidden []string, gaps *[]string) {
	block := makefileTargetBlock(content, target)
	if block == "" {
		*gaps = append(*gaps, "Makefile missing target block "+target)
		return
	}
	dependencies := makefileTargetDependencies(content, target)
	for _, token := range required {
		if !makefileDependencyHasToken(dependencies, token) {
			*gaps = append(*gaps, "Makefile "+target+" missing dependency "+token)
		}
	}
	for _, token := range forbidden {
		if makefileDependencyHasToken(dependencies, token) {
			*gaps = append(*gaps, "Makefile "+target+" must not depend on "+token)
		}
	}
}

func appendMakefileTargetForbiddenReferenceGaps(content, target string, forbidden []string, gaps *[]string) {
	block := makefileTargetBlock(content, target)
	if block == "" {
		*gaps = append(*gaps, "Makefile missing target block "+target)
		return
	}
	for _, token := range forbidden {
		if strings.Contains(block, token) {
			*gaps = append(*gaps, "Makefile "+target+" must not reference "+token)
		}
	}
}

func appendReleaseFinalDelegationGaps(content string, gaps *[]string) {
	block := makefileTargetBlock(content, "release-final-check")
	if block == "" {
		*gaps = append(*gaps, "Makefile missing target block release-final-check")
		return
	}
	if makefileDependencyHasToken(makefileTargetDependencies(content, "release-final-check"), "release-final-check") || strings.Contains(block, "$(MAKE) release-final-check") || strings.Contains(block, "make release-final-check") || strings.Contains(block, "$(GOALCLI) release-final-check") {
		*gaps = append(*gaps, "release-final-check must not call itself")
	}
	if !strings.Contains(block, "$(MAKE) context-release") {
		*gaps = append(*gaps, "release-final-check must call context-release")
	}
}

func makefileDependencyHasToken(dependencies []string, token string) bool {
	for _, field := range dependencies {
		if field == token {
			return true
		}
	}
	return false
}

var plannedCommandFiles = map[string][]string{
	"minimal-kernel":           {".agent/runtime/minimal-kernel.yaml"},
	"done-assertion":           {".agent/evidence/done-assertion.yaml"},
	"agent-team-contract":      {".agent/contracts/team-contract.yaml"},
	"scope-lock":               {".agent/contracts/scope-locks.yaml"},
	"pr-template":              {".agent/contracts/pr-template-contract.yaml", ".github/pull_request_template.md"},
	"acceptance-matrix":        {".agent/contracts/acceptance-matrix.yaml"},
	"runtime-health":           {".agent/contracts/runtime-health.yaml"},
	"goal-runtime":             {".agent/runtime/goal-runtime.md", ".agent/harness/harness.yaml"},
	"goal-acceptance":          {".agent/harness/harness.yaml"},
	"goal-delivery":            {".agent/harness/harness.yaml"},
	"goal-handover":            {".agent/harness/harness.yaml"},
	"goal-downstream-adoption": {".agent/harness/harness.yaml"},
	"goal-certify":             {".agent/harness/harness.yaml"},
	"goal-runtime-final":       {".agent/harness/harness.yaml"},
	"naming":                   {"docs/standard/repository-roles.md", "docs/standard/module-boundary.md"},
	"upgrade-standard":         {".agent/registries/downstream-registry.yaml"},
	"conformance-profile":      {".agent/policies/conformance-profiles.yaml"},
	"downstream-registry":      {".agent/registries/downstream-registry.yaml"},
	"self-healing-skeleton":    {".agent/traceability/failure-taxonomy.yaml", ".agent/traceability/root-cause.yaml", ".agent/traceability/regression-memory.yaml"},
	"policy-schema":            {".agent/policies/policy-schema.yaml"},
	"github-settings":          {".agent/policies/github-settings.yaml"},
	"github-governance":        {".agent/policies/github-governance.yaml"},
	"governance-fixture-test":  {".agent/harness/governance-fixture-test.yaml"},
	"toolchain":                {".agent/policies/toolchain.yaml"},
	"evidence-artifacts":       {".agent/evidence/evidence-artifact-policy.yaml"},
	"install-runtime":          {".agent/contracts/runtime-install.yaml"},
	"upgrade-runtime":          {".agent/contracts/runtime-upgrade.yaml"},
	"release-ready":            {".agent/release/release-readiness-formula.yaml", ".agent/release/release-required-gates.yaml", ".agent/evidence/evidence-replay.yaml", ".agent/policies/execution-context.yaml"},
	"evidence-replay":          {".agent/evidence/evidence-replay.yaml"},
	"attest-conformance":       {".agent/policies/conformance-profiles.yaml"},
	"pack-standard":            {".agent/contracts/standard-pack.yaml"},
	"pack-gate":                {".agent/harness/gate-pack.yaml"},
	"pack-evidence":            {".agent/evidence/evidence-pack.yaml"},
	"runtime-file-ownership":   {".agent/policies/runtime-file-ownership.yaml"},
	"downstream-baseline":      {".agent/registries/downstream-baseline-scan.yaml", ".agent/registries/downstream-registry.yaml"},
	"downstream-adoption":      {".agent/registries/downstream-adoption-modes.yaml", ".agent/registries/downstream-registry.yaml", ".agent/registries/downstream-adoption-status.yaml", "contracts/downstream-adoption-proof.schema.json", "docs/standard/downstream-registry.md"},
	"autoresearch":             {".agent/policies/autoresearch.yaml"},
	"changelog":                {".agent/archive/changelog.yaml"},
	"supply-chain":             {"docs/supply-chain.md"},
	"execution-context":        {".agent/policies/execution-context.yaml", "contracts/execution-context.schema.json"},
}

var plannedCommandSemanticMarkers = map[string]map[string][]string{
	"agent-team-contract": {
		".agent/contracts/team-contract.yaml": {"schema_version:", "roles:", "rule:"},
	},
	"acceptance-matrix": {
		".agent/contracts/acceptance-matrix.yaml": {"schema_version:", "acceptance:"},
	},
	"runtime-health": {
		".agent/contracts/runtime-health.yaml": {"schema_version:", "checks:", "toolchain"},
	},
	"goal-acceptance": {
		".agent/harness/harness.yaml": {"goalcli_mva_gates:", "G12_ACCEPTANCE", "goal-acceptance"},
	},
	"goal-delivery": {
		".agent/harness/harness.yaml": {"goalcli_mva_gates:", "G13_DELIVERY", "goal-delivery"},
	},
	"goal-handover": {
		".agent/harness/harness.yaml": {"goalcli_mva_gates:", "G14_HANDOVER", "goal-handover"},
	},
	"goal-downstream-adoption": {
		".agent/harness/harness.yaml": {"goalcli_mva_gates:", "G15_DOWNSTREAM_ADOPTION", "goal-downstream-adoption"},
	},
	"goal-certify": {
		".agent/harness/harness.yaml": {"goalcli_mva_gates:", "G16_CERTIFY", "goal-certify"},
	},
	"goal-runtime-final": {
		".agent/harness/harness.yaml": {"goalcli_mva_gates:", "G12_G16_FINAL", "goal-runtime-final"},
	},
	"execution-context": {
		".agent/policies/execution-context.yaml": {"schema_version:", "contexts:", "local_write:", "ci_pull_request:", "release_verify:", "mutates_files:", "release_evidence:"},
	},
	"evidence-replay": {
		".agent/evidence/evidence-replay.yaml": {"schema_version:", "fixtures:", "ledger:", "expected_status:", "hash_chain"},
	},
	"downstream-adoption": {
		".agent/registries/downstream-adoption-modes.yaml":  {"schema_version:", "modes:", "patch-only"},
		".agent/registries/downstream-adoption-status.yaml": {"proof_contract:", "source_repo", "gate_outputs", "rollback"},
		"contracts/downstream-adoption-proof.schema.json":   {"source_repo", "source_commit", "gate_outputs", "rollback"},
		"docs/standard/downstream-registry.md":              {"Proof contract", "source_repo", "gate_outputs", "rollback"},
	},
	"runtime-file-ownership": {
		".agent/policies/runtime-file-ownership.yaml": {"schema_version:", "owners:", "owner:", "review_required:", "review_rule:", "rationale:"},
	},
}

func runPlannedCommand(command string, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validatePlannedCommandArgs(command, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		write(stderr, "ERROR: %s invalid arguments: %v\n", command, err)
		return 2
	}

	var details []string
	var gaps []string
	fileContents := map[string]string{}
	files, ok := plannedCommandFiles[command]
	if !ok || len(files) == 0 {
		write(stderr, "ERROR: %s has no manifest coverage\n", command)
		return emitReport(stdout, command, "failed", []string{"args=" + strings.Join(args, " ")}, []string{"planned command has no manifest coverage: " + command})
	}
	for _, path := range files {
		content, gap, ok := readPlannedCommandFile(path)
		if !ok {
			gaps = append(gaps, gap)
			continue
		}
		fileContents[path] = string(content)
		details = append(details, "found "+path)
		gaps = append(gaps, validatePlannedCommandFile(command, path, content)...)
	}
	if command == "release-ready" {
		details = append(details, releaseReadyDecisionDetails(args, fileContents, &gaps)...)
	}
	if command == "downstream-baseline" || command == "downstream-adoption" || command == "upgrade-standard" {
		mode := fallback(flagValue(args, "mode", ""), "patch-only")
		if flagProvided(args, "repo") {
			repo := flagValue(args, "repo", "")
			details = append(details, "repo="+repo, "mode="+mode, "dry_run=true")
			if !fileExists(repo) {
				gaps = append(gaps, "downstream repo unavailable in worker workspace: "+repo)
				return emitPlannedReport(stdout, stderr, command, "gap", details, gaps, args)
			}
		} else {
			details = append(details, "repo=manifest-only", "mode="+mode, "dry_run=true")
		}
	}
	if command == "evidence-replay" {
		replayDetails, replayGaps, status := evaluateEvidenceReplay(flagValue(args, "input", "testkit/governance/fixtures/evidence-replay/passed"))
		details = append(details, replayDetails...)
		gaps = append(gaps, replayGaps...)
		if status != "" && status != "passed" {
			return emitPlannedReport(stdout, stderr, command, status, details, gaps, args)
		}
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: %s found %d gap(s)\n", command, len(gaps))
		return emitReport(stdout, command, "failed", details, gaps)
	}
	details = append(details, "args="+strings.Join(args, " "))
	if plannedCommandVerifyRequested(args) {
		details = append(details, "local dry-run verifier satisfied manifest coverage")
	}
	return emitReport(stdout, command, "passed", details, nil)
}

type evidenceReplayExpectation struct {
	SchemaVersion string            `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	MaxAgeHours   int               `json:"max_age_hours"`
	Commands      map[string]string `json:"commands"`
}

type evidenceReplayEntry struct {
	Seq          int    `json:"seq"`
	Command      string `json:"command"`
	Status       string `json:"status"`
	StdoutSHA256 string `json:"stdout_sha256"`
	ArtifactPath string `json:"artifact_path"`
	PreviousHash string `json:"previous_hash"`
	EntryHash    string `json:"entry_hash"`
}

func evaluateEvidenceReplay(inputDir string) ([]string, []string, string) {
	if strings.TrimSpace(inputDir) == "" {
		inputDir = "testkit/governance/fixtures/evidence-replay/passed"
	}
	ledgerPath := filepath.Join(inputDir, "ledger.jsonl")
	expectedPath := filepath.Join(inputDir, "expected-status.json")
	details := []string{"fixture=" + inputDir}
	var gaps []string

	expectedData, err := os.ReadFile(expectedPath)
	if err != nil {
		return details, []string{"missing evidence replay expected status: " + expectedPath}, "gap"
	}
	var expected evidenceReplayExpectation
	if err := json.Unmarshal(expectedData, &expected); err != nil {
		return details, []string{"invalid evidence replay expected status " + expectedPath + ": " + err.Error()}, "failed"
	}
	if staleGap := evidenceReplayStaleGap(expected, expectedPath); staleGap != "" {
		return details, []string{staleGap}, "gap"
	}

	ledgerData, err := os.ReadFile(ledgerPath)
	if err != nil {
		return details, []string{"missing evidence replay ledger: " + ledgerPath}, "gap"
	}
	entries, parseGaps := parseEvidenceReplayLedger(ledgerData, ledgerPath)
	gaps = append(gaps, parseGaps...)
	statusByCommand := map[string]string{}
	previousHash := strings.Repeat("0", 64)
	for _, entry := range entries {
		if entry.Command == "" {
			gaps = append(gaps, fmt.Sprintf("%s entry %d missing command", ledgerPath, entry.Seq))
		}
		if entry.Status == "" {
			gaps = append(gaps, fmt.Sprintf("%s entry %d missing status", ledgerPath, entry.Seq))
		}
		artifactPath := filepath.Join(inputDir, entry.ArtifactPath)
		artifactData, err := os.ReadFile(artifactPath)
		if err != nil {
			gaps = append(gaps, fmt.Sprintf("%s entry %d missing artifact %s", ledgerPath, entry.Seq, entry.ArtifactPath))
		} else if got := sha256Hex(artifactData); got != entry.StdoutSHA256 {
			gaps = append(gaps, fmt.Sprintf("%s entry %d checksum mismatch for %s: got %s want %s", ledgerPath, entry.Seq, entry.ArtifactPath, got, entry.StdoutSHA256))
		}
		if entry.PreviousHash != previousHash {
			gaps = append(gaps, fmt.Sprintf("%s entry %d hash chain mismatch: previous_hash got %s want %s", ledgerPath, entry.Seq, entry.PreviousHash, previousHash))
		}
		if got := evidenceReplayEntryHash(entry); got != entry.EntryHash {
			gaps = append(gaps, fmt.Sprintf("%s entry %d entry_hash mismatch: got %s want %s", ledgerPath, entry.Seq, got, entry.EntryHash))
		}
		statusByCommand[entry.Command] = entry.Status
		previousHash = entry.EntryHash
	}
	if len(entries) == 0 {
		gaps = append(gaps, "evidence replay ledger has no entries: "+ledgerPath)
	}
	for command, wantStatus := range expected.Commands {
		if gotStatus, ok := statusByCommand[command]; !ok {
			gaps = append(gaps, "evidence replay missing expected command status: "+command)
		} else if gotStatus != wantStatus {
			gaps = append(gaps, fmt.Sprintf("evidence replay status mismatch for %s: got %s want %s", command, gotStatus, wantStatus))
		}
	}
	if len(gaps) > 0 {
		return details, gaps, "failed"
	}
	details = append(details,
		"replayed ledger="+ledgerPath,
		fmt.Sprintf("replayed commands=%d", len(entries)),
		"checksum verified",
		"hash chain verified",
		"expected command status verified",
	)
	return details, nil, "passed"
}

func evidenceReplayStaleGap(expected evidenceReplayExpectation, expectedPath string) string {
	if expected.GeneratedAt == "" || expected.MaxAgeHours <= 0 {
		return ""
	}
	generatedAt, err := time.Parse(time.RFC3339, expected.GeneratedAt)
	if err != nil {
		return "invalid evidence replay generated_at " + expectedPath + ": " + err.Error()
	}
	if time.Now().UTC().After(generatedAt.Add(time.Duration(expected.MaxAgeHours) * time.Hour)) {
		return fmt.Sprintf("stale evidence replay fixture: %s older than %d hours", expected.GeneratedAt, expected.MaxAgeHours)
	}
	return ""
}

func parseEvidenceReplayLedger(data []byte, ledgerPath string) ([]evidenceReplayEntry, []string) {
	var entries []evidenceReplayEntry
	var gaps []string
	for lineNumber, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var entry evidenceReplayEntry
		if err := json.Unmarshal([]byte(trimmed), &entry); err != nil {
			gaps = append(gaps, fmt.Sprintf("%s line %d invalid JSON: %v", ledgerPath, lineNumber+1, err))
			continue
		}
		if entry.Seq == 0 {
			entry.Seq = len(entries) + 1
		}
		entries = append(entries, entry)
	}
	return entries, gaps
}

func evidenceReplayEntryHash(entry evidenceReplayEntry) string {
	payload := fmt.Sprintf("%d|%s|%s|%s|%s", entry.Seq, entry.Command, entry.Status, entry.StdoutSHA256, entry.PreviousHash)
	return sha256Hex([]byte(payload))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func readPlannedCommandFile(path string) ([]byte, string, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "missing " + path, false
	}
	if info.IsDir() {
		return nil, path + " must be a file", false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, path + " unreadable: " + err.Error(), false
	}
	return content, "", true
}

func validatePlannedCommandFile(command string, path string, content []byte) []string {
	var gaps []string
	text := string(content)
	if strings.TrimSpace(text) == "" {
		gaps = append(gaps, path+" must not be empty")
	}
	if filepath.Ext(path) == ".json" && !json.Valid(content) {
		gaps = append(gaps, path+" must be valid JSON")
	}
	for _, marker := range plannedCommandMarkers(command, path) {
		if !strings.Contains(text, marker) {
			gaps = append(gaps, path+" missing semantic marker "+marker)
		}
	}
	if command == "runtime-file-ownership" && path == ".agent/policies/runtime-file-ownership.yaml" {
		gaps = append(gaps, validation.ValidateRuntimeFileOwnership(path, text)...)
	}
	if (command == "execution-context" || command == "release-ready") && path == ".agent/policies/execution-context.yaml" {
		gaps = append(gaps, validation.ValidateExecutionContext(path, text, validExecutionContexts)...)
	}
	return gaps
}

func releaseReadyDecisionDetails(args []string, fileContents map[string]string, gaps *[]string) []string {
	context := fallback(flagValue(args, "context", ""), "release_verify")
	verifyRequested := plannedCommandVerifyRequested(args)
	dryRunRequested := flagProvided(args, "dry-run")
	requireContract := verifyRequested
	requireReadiness := verifyRequested && !dryRunRequested
	details := []string{"context=" + context}
	if requireContract && context != "release_verify" {
		*gaps = append(*gaps, "release-ready requires context release_verify; got "+context)
	}

	requiredGates := countYAMLLinesWithValue(fileContents[".agent/release/release-required-gates.yaml"], "required_for_release", "true")
	releaseUsableGates := countYAMLLinesWithValue(fileContents[".agent/release/release-required-gates.yaml"], "release_usable", "true")
	blockedGates := requiredGates - releaseUsableGates
	score := 0
	if requiredGates > 0 {
		score = releaseUsableGates * 100 / requiredGates
	}
	verdict := "ready"
	if requiredGates == 0 {
		verdict = "gap"
		if requireContract {
			*gaps = append(*gaps, ".agent/release/release-required-gates.yaml gates must include required release gates")
		}
	} else if blockedGates > 0 {
		verdict = "not_ready"
		if requireReadiness {
			*gaps = append(*gaps, fmt.Sprintf("release-ready verdict not_ready: %d/%d required gates are not release_usable", blockedGates, requiredGates))
		}
	}

	evidenceFields := countYAMLListItems(fileContents[".agent/release/release-required-gates.yaml"], "required_release_evidence")
	if requireContract && evidenceFields == 0 {
		*gaps = append(*gaps, ".agent/release/release-required-gates.yaml missing required_release_evidence items")
	}
	replayReady := strings.Contains(fileContents[".agent/evidence/evidence-replay.yaml"], "replay:") && strings.Contains(fileContents[".agent/evidence/evidence-replay.yaml"], "strict: true")
	if requireContract && !replayReady {
		*gaps = append(*gaps, ".agent/evidence/evidence-replay.yaml replay must be strict")
	}

	mode := "readiness_gate"
	if dryRunRequested {
		mode = "dry_run_contract"
	}
	details = append(details,
		"mode="+mode,
		"verdict="+verdict,
		fmt.Sprintf("score=%d/100", score),
		fmt.Sprintf("required_gates=%d", requiredGates),
		fmt.Sprintf("release_usable_gates=%d", releaseUsableGates),
		fmt.Sprintf("evidence_replay=strict:%t", replayReady),
		fmt.Sprintf("required_evidence_fields=%d", evidenceFields),
		"reasons=release-ready uses required gate release_usable state, strict evidence replay, and release_verify context",
	)
	return details
}

func countYAMLLinesWithValue(content string, key string, value string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		field, got, ok := strings.Cut(strings.TrimSpace(stripInlineComment(line)), ":")
		if ok && strings.TrimSpace(field) == key && strings.TrimSpace(got) == value {
			count++
		}
	}
	return count
}

func countYAMLListItems(content string, listName string) int {
	count := 0
	inList := false
	for _, line := range strings.Split(content, "\n") {
		clean := stripInlineComment(line)
		trimmed := strings.TrimSpace(clean)
		indent := len(clean) - len(strings.TrimLeft(clean, " "))
		if !inList {
			if trimmed == listName+":" {
				inList = true
			}
			continue
		}
		if indent == 0 && trimmed != "" {
			break
		}
		if strings.HasPrefix(trimmed, "- ") {
			count++
		}
	}
	return count
}

func stripInlineComment(line string) string {
	if before, _, ok := strings.Cut(line, "#"); ok {
		return before
	}
	return line
}

func plannedCommandMarkers(command string, path string) []string {
	files, ok := plannedCommandSemanticMarkers[command]
	if !ok {
		return nil
	}
	return files[path]
}

func flagProvided(args []string, name string) bool {
	for _, arg := range args {
		if arg == "--"+name || strings.HasPrefix(arg, "--"+name+"=") {
			return true
		}
	}
	return false
}

func validatePlannedCommandArgs(command string, args []string) error {
	flags := flag.NewFlagSet("goalcli "+command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Bool("dry-run", false, "")
	flags.Bool("verify", false, "")
	flags.Bool("strict", false, "")
	flags.Bool("json", false, "")
	flags.String("repo", "", "")
	flags.String("mode", "", "")
	flags.String("input", "", "")
	context := flags.String("context", "", "")
	flags.String("profile", "", "")
	flags.String("output", "", "")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("unexpected positional argument %q", flags.Arg(0))
	}
	if *context != "" && !validContext(*context) {
		return fmt.Errorf("invalid context %q", *context)
	}
	return nil
}

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
	"audit-goal",
	"dashboard-generate",
	"traceability-check",
	"boundary",
	"contracts",
	"schema-check",
	"dependency-check",
	"docs-check",
	"debt",
	"architecture",
	"domain",
	"docs-drift",
	"dependency-debt",
	"security-debt",
	"testing-debt",
	"implementation-debt",
	"downstream-debt",
	"downstream-sync-plan",
	"adoption-check",
	"docker-toolchain-check",
	"docker-build",
	"docker-build-check",
	"docker-shell",
	"docker-ci",
	"docker-release-check",
	"docker-release-final-check",
	"docker-goalcli",
	"docker-goalcli-image",
	"docker-goalcli-version",
	"docker-runtime-check",
	"docker-drift-check",
	"docker-contract",
	"debt-evidence",
	"debt-evidence-checksum-check",
	"debt-evidence-hash",
	"evidence",
	"manifest",
	"integration",
	"release-evidence-check",
	"release-evidence-checksum-check",
	"release-evidence-hash",
	"release-final-check",
	"render-check",
	"rules-consistency-check",
	"rules-verify",
	"score",
	"secrets",
	"security",
	"standard-impact-check",
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
}

func commandRegistryRequiredCommands() []string {
	return append([]string(nil), commandRegistryCommands...)
}

func goalcliCLIContractNeedles() []string {
	needles := commandRegistryRequiredCommands()
	return append(needles, "worktree-check --context", "pr-check --context")
}

func requiredCommandRegistryNeedles() []string {
	needles := make([]string, 0, len(commandRegistryCommands))
	for _, command := range commandRegistryCommands {
		needles = append(needles, "name: "+command)
	}
	return needles
}

type agentIndexEntry struct {
	path  string
	block string
}

func appendAgentIndexGaps(path string, gaps *[]string) {
	content, err := os.ReadFile(path)
	if err != nil {
		*gaps = append(*gaps, "missing "+path)
		return
	}
	text := string(content)
	for _, needle := range []string{"schema_version:", "module: resiliencx", "control_plane:", "files:"} {
		if !strings.Contains(text, needle) {
			*gaps = append(*gaps, path+" missing "+needle)
		}
	}
	entries := parseAgentIndexEntries(text)
	if len(entries) == 0 {
		*gaps = append(*gaps, path+" must define files entries")
		return
	}

	allowedLayers := scalarSet("runtime_contract", "machine_contract", "registry", "evidence", "traceability", "policy", "archive", "documentation")
	allowedAuthorities := scalarSet("source_of_truth", "validated_mirror", "historical_snapshot")
	allowedMutability := scalarSet("hand_written", "append_only", "generated")
	seen := map[string]bool{}
	present := map[string]bool{}
	for _, entry := range entries {
		if entry.path == "" {
			*gaps = append(*gaps, path+" contains file entry without path")
			continue
		}
		if seen[entry.path] {
			*gaps = append(*gaps, path+" duplicate file entry "+entry.path)
		}
		seen[entry.path] = true
		present[entry.path] = true
		if !strings.HasPrefix(entry.path, ".agent/") {
			*gaps = append(*gaps, path+" "+entry.path+" must stay under .agent/")
		}
		if info, statErr := os.Stat(entry.path); statErr != nil {
			*gaps = append(*gaps, path+" references missing "+entry.path)
		} else if info.IsDir() {
			*gaps = append(*gaps, path+" "+entry.path+" must be a file")
		}

		appendRequiredAgentIndexField(path, entry, "layer", gaps)
		appendRequiredAgentIndexField(path, entry, "authority", gaps)
		appendRequiredAgentIndexField(path, entry, "mutability", gaps)
		appendRequiredAgentIndexField(path, entry, "owner", gaps)
		appendRequiredAgentIndexField(path, entry, "validator", gaps)
		appendRequiredAgentIndexField(path, entry, "purpose", gaps)
		appendAgentIndexEnumGap(path, entry, "layer", allowedLayers, gaps)
		appendAgentIndexEnumGap(path, entry, "authority", allowedAuthorities, gaps)
		appendAgentIndexEnumGap(path, entry, "mutability", allowedMutability, gaps)
	}
	for _, required := range requiredAgentIndexPaths() {
		if !present[required] {
			*gaps = append(*gaps, path+" missing file entry "+required)
		}
	}
	appendAgentIndexClassificationGaps(path, entries, gaps)
	appendUnclassifiedAgentFileGaps(filepath.Dir(path), path, present, gaps)
}

func appendAgentIndexClassificationGaps(indexPath string, entries []agentIndexEntry, gaps *[]string) {
	required := map[string]map[string]string{
		".agent/registries/generated-artifacts.yaml": {
			"layer":      "registry",
			"authority":  "source_of_truth",
			"mutability": "hand_written",
		},
		".agent/rules/registry.yaml": {
			"authority":  "validated_mirror",
			"mutability": "generated",
		},
		".agent/rules/agent-runtime-rules.md": {
			"authority":  "validated_mirror",
			"mutability": "generated",
		},
		".agent/rules/core-rules.md": {
			"authority":  "validated_mirror",
			"mutability": "generated",
		},
		".agent/rules/schema-registry-rules.md": {
			"authority":  "validated_mirror",
			"mutability": "generated",
		},
	}
	for _, entry := range entries {
		fields, ok := required[entry.path]
		if !ok {
			continue
		}
		for field, want := range fields {
			got, ok := blockYAMLValue(entry.block, field)
			if !ok || got != want {
				*gaps = append(*gaps, fmt.Sprintf("%s %s must classify %s as %s", indexPath, entry.path, field, want))
			}
		}
	}
}

func appendGeneratedArtifactsGaps(path string, indexPath string, gaps *[]string) {
	content, err := os.ReadFile(path)
	if err != nil {
		*gaps = append(*gaps, "missing "+path)
		return
	}
	text := string(content)
	for _, needle := range []string{"schema_version:", "classification:", "artifacts:"} {
		if !strings.Contains(text, needle) {
			*gaps = append(*gaps, path+" missing "+needle)
		}
	}
	for _, field := range []string{"artifact_class", "authority", "validated_by"} {
		if !blockHasNonEmptyYAMLValue(text, field) {
			*gaps = append(*gaps, path+" classification missing "+field)
		}
	}

	artifacts := parseYAMLSequenceBlocks(text, "artifacts", "path")
	if len(artifacts) == 0 {
		*gaps = append(*gaps, path+" must define generated artifact entries")
		return
	}
	validators := knownValidationRefs()
	for _, artifact := range artifacts {
		if artifact.value == "" {
			*gaps = append(*gaps, path+" contains artifact without path")
		}
		requireYAMLBlockValue(path, artifact.value, artifact.block, "classification", expectedGeneratedArtifactClassification(artifact.value), gaps)
		requireYAMLBlockValue(path, artifact.value, artifact.block, "source_control", "generated-only", gaps)
		if !blockHasNonEmptyYAMLValue(artifact.block, "generated_by") {
			*gaps = append(*gaps, path+" "+artifact.value+" missing generated_by")
		}
		validator, ok := blockYAMLValue(artifact.block, "validated_by")
		if !ok || validator == "" {
			*gaps = append(*gaps, path+" "+artifact.value+" missing validated_by")
		} else if !validators[validator] {
			*gaps = append(*gaps, path+" "+artifact.value+" validated_by "+validator+" is not a known goalcli or Makefile gate")
		}
	}

	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		*gaps = append(*gaps, "missing "+indexPath)
		return
	}
	for _, entry := range parseAgentIndexEntries(string(indexContent)) {
		if entry.path == path {
			requireYAMLBlockValue(indexPath, path, entry.block, "layer", "registry", gaps)
			requireYAMLBlockValue(indexPath, path, entry.block, "authority", "source_of_truth", gaps)
			requireYAMLBlockValue(indexPath, path, entry.block, "mutability", "hand_written", gaps)
			return
		}
	}
	*gaps = append(*gaps, indexPath+" missing file entry "+path)
}

func expectedGeneratedArtifactClassification(path string) string {
	if strings.HasPrefix(path, ".agent/rules/") {
		return "validated_mirror"
	}
	return "generated_artifact"
}

func appendHarnessAliasGaps(path string, gaps *[]string) {
	content, err := os.ReadFile(path)
	if err != nil {
		*gaps = append(*gaps, "missing "+path)
		return
	}
	entries := parseYAMLSequenceBlocks(string(content), "required_gates", "id")
	if len(entries) == 0 {
		*gaps = append(*gaps, path+" must define required_gates")
		return
	}
	byID := map[string]yamlSequenceBlock{}
	for _, entry := range entries {
		byID[entry.value] = entry
	}
	requiredAliases := map[string]string{
		"governance_chain":            "governance_check",
		"governance_release_scope":    "governance_check",
		"p1_governance_chain":         "p1_governance_check",
		"p1_governance_release_scope": "p1_governance_check",
		"p2_runtime_chain":            "p2_runtime_check",
		"p2_runtime_release_scope":    "p2_runtime_check",
	}
	for alias, target := range requiredAliases {
		entry, ok := byID[alias]
		if !ok {
			*gaps = append(*gaps, path+" missing required gate alias "+alias)
			continue
		}
		if _, ok := byID[target]; !ok {
			*gaps = append(*gaps, path+" "+alias+" alias_of missing target "+target)
		}
		requireYAMLBlockValue(path, alias, entry.block, "alias_of", target, gaps)
		if !blockHasNonEmptyYAMLValue(entry.block, "semantic_role") {
			*gaps = append(*gaps, path+" "+alias+" missing semantic_role")
		}
	}
}

func appendRulesEnforcedByGaps(path string, gaps *[]string) {
	content, err := os.ReadFile(path)
	if err != nil {
		*gaps = append(*gaps, "missing "+path)
		return
	}
	text := string(content)
	for _, needle := range []string{"generated_from:", "rules:"} {
		if !strings.Contains(text, needle) {
			*gaps = append(*gaps, path+" missing "+needle)
		}
	}
	rules := parseYAMLSequenceBlocks(text, "rules", "id")
	if len(rules) == 0 {
		*gaps = append(*gaps, path+" must define rules")
		return
	}
	commandNames := knownGoalcliCommandRefs()
	makeTargets := knownMakeTargetRefs()
	for _, rule := range rules {
		status, _ := blockYAMLValue(rule.block, "status")
		enforcedBy, _ := blockYAMLValue(rule.block, "enforced_by")
		switch status {
		case "active":
			if enforcedBy == "" {
				*gaps = append(*gaps, path+" "+rule.value+" active rule missing enforced_by")
				continue
			}
			if !knownEnforcementRef(enforcedBy, commandNames, makeTargets) {
				*gaps = append(*gaps, path+" "+rule.value+" enforced_by "+enforcedBy+" is not tied to a known goalcli command, Makefile target, script, or hook")
			}
		case "indexed":
			if enforcedBy != "" {
				*gaps = append(*gaps, path+" "+rule.value+" indexed rule must not set enforced_by")
			}
		case "deprecated":
			continue
		default:
			*gaps = append(*gaps, path+" "+rule.value+" invalid status "+status)
		}
	}
}

func parseAgentIndexEntries(text string) []agentIndexEntry {
	var entries []agentIndexEntry
	lines := strings.Split(text, "\n")
	inFiles := false
	var current *agentIndexEntry
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inFiles {
			if trimmed == "files:" {
				inFiles = true
			}
			continue
		}
		if isTopLevelYAMLKey(line) && trimmed != "files:" {
			break
		}
		if strings.HasPrefix(trimmed, "- path:") {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &agentIndexEntry{path: trimYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- path:")))}
		}
		if current != nil {
			current.block += line + "\n"
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}
	return entries
}

func appendRequiredAgentIndexField(indexPath string, entry agentIndexEntry, field string, gaps *[]string) {
	if !blockHasNonEmptyYAMLValue(entry.block, field) {
		*gaps = append(*gaps, indexPath+" "+entry.path+" missing "+field+":")
	}
}

func appendAgentIndexEnumGap(indexPath string, entry agentIndexEntry, field string, allowed map[string]bool, gaps *[]string) {
	value, ok := blockYAMLValue(entry.block, field)
	if !ok || value == "" {
		return
	}
	if !allowed[value] {
		*gaps = append(*gaps, indexPath+" "+entry.path+" invalid "+field+" "+value)
	}
}

func requiredAgentIndexPaths() []string {
	return []string{
		".agent/index.yaml",
		".agent/runtime/goal-runtime.md",
		".agent/runtime/object-model.md",
		".agent/runtime/state-machine.md",
		".agent/harness/harness.yaml",
		".agent/registries/command-registry.yaml",
		".agent/registries/issue-registry.yaml",
		".agent/registries/generated-artifacts.yaml",
		".agent/registries/command-implementation-status.yaml",
		".agent/registries/makefile-target-registry.yaml",
		".agent/registries/makefile-baseline.yaml",
		".agent/release/release-required-gates.yaml",
		".agent/evidence/evidence-protocol.md",
		".agent/evidence/ledger.jsonl",
		".agent/policies/runtime-file-ownership.yaml",
		".agent/policies/execution-context.yaml",
		".agent/policies/policy-schema.yaml",
		".agent/traceability/traceability-matrix.md",
		".agent/traceability/risk-register.md",
		".agent/traceability/decision-log.md",
		".agent/runtime/rollback-protocol.md",
		".agent/release/release-template.md",
		".agent/docs/agent-teams.md",
		".agent/rules/registry.yaml",
		".agent/rules/agent-runtime-rules.md",
		".agent/rules/core-rules.md",
		".agent/rules/schema-registry-rules.md",
		".agent/archive/retrospective.md",
	}
}

func appendUnclassifiedAgentFileGaps(root string, indexPath string, present map[string]bool, gaps *[]string) {
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			*gaps = append(*gaps, "read "+filepath.ToSlash(path)+": "+err.Error())
			return err
		}
		if entry.IsDir() {
			return nil
		}
		normalized := filepath.ToSlash(path)
		if !present[normalized] {
			*gaps = append(*gaps, indexPath+" missing file entry "+normalized)
		}
		return nil
	})
	if err != nil {
		*gaps = append(*gaps, "read "+filepath.ToSlash(root)+": "+err.Error())
	}
}

func appendGeneratedArtifactClassificationGaps(indexPath string, artifactsPath string, gaps *[]string) {
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return
	}
	entries := parseAgentIndexEntries(string(content))
	byPath := make(map[string]agentIndexEntry, len(entries))
	var generatedAgentPaths []string
	for _, entry := range entries {
		byPath[entry.path] = entry
		if strings.HasPrefix(entry.path, ".agent/") {
			mutability, _ := blockYAMLValue(entry.block, "mutability")
			if mutability == "generated" {
				generatedAgentPaths = append(generatedAgentPaths, entry.path)
			}
		}
	}

	if registry, ok := byPath[artifactsPath]; ok {
		authority, _ := blockYAMLValue(registry.block, "authority")
		if authority != "source_of_truth" {
			*gaps = append(*gaps, artifactsPath+" must be indexed as source_of_truth")
		}
		mutability, _ := blockYAMLValue(registry.block, "mutability")
		if mutability != "hand_written" {
			*gaps = append(*gaps, artifactsPath+" must be indexed as hand_written")
		}
	}

	artifactPaths, err := yamlListScalarValues(artifactsPath, "path")
	if err != nil {
		if len(generatedAgentPaths) > 0 {
			*gaps = append(*gaps, "missing "+artifactsPath+" for generated .agent files")
		}
		return
	}
	registered := scalarSet(artifactPaths...)
	for _, path := range generatedAgentPaths {
		if path == artifactsPath {
			continue
		}
		if !registered[path] {
			*gaps = append(*gaps, indexPath+" "+path+" mutability generated requires "+artifactsPath+" entry")
		}
	}
	for _, path := range artifactPaths {
		if !strings.HasPrefix(path, ".agent/") {
			continue
		}
		entry, ok := byPath[path]
		if !ok {
			*gaps = append(*gaps, artifactsPath+" references unindexed agent artifact "+path)
			continue
		}
		mutability, _ := blockYAMLValue(entry.block, "mutability")
		if mutability != "generated" {
			*gaps = append(*gaps, artifactsPath+" "+path+" must be indexed with mutability generated")
		}
		authority, _ := blockYAMLValue(entry.block, "authority")
		if authority == "source_of_truth" {
			*gaps = append(*gaps, artifactsPath+" "+path+" generated artifact must not be source_of_truth")
		}
	}
}

func appendHarnessGateLinkSemanticsGaps(path string, gaps *[]string) {
	content, err := os.ReadFile(path)
	if err != nil {
		*gaps = append(*gaps, "missing "+path)
		return
	}
	text := string(content)
	required := []string{
		"gate_link_semantics:",
		"duplicate_command_links: aliases",
		"duplicate_entries_do_not_create_new_authorities: true",
		"authority_source: required_gates[].id",
	}
	for _, needle := range required {
		if !strings.Contains(text, needle) {
			*gaps = append(*gaps, path+" missing "+needle)
		}
	}
}

func scalarSet(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

type issueRegistryEntry struct {
	id    string
	block string
}

var issueRegistryIDPattern = regexp.MustCompile(`^(P0|P1|P2|CTX)-([0-9]{3})$`)

func appendIssueRegistryGaps(path string, gaps *[]string) {
	content, err := os.ReadFile(path)
	if err != nil {
		*gaps = append(*gaps, "missing "+path)
		return
	}
	*gaps = append(*gaps, validateIssueRegistryEntries(path, parseIssueRegistryEntries(string(content)))...)
}

func parseIssueRegistryEntries(text string) []issueRegistryEntry {
	var entries []issueRegistryEntry
	var currentID string
	var currentLines []string
	flush := func() {
		if currentID != "" {
			entries = append(entries, issueRegistryEntry{id: currentID, block: strings.Join(currentLines, "\n")})
		}
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- id:") {
			flush()
			currentID = trimYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- id:")))
			currentLines = []string{line}
			continue
		}
		if currentID != "" {
			currentLines = append(currentLines, line)
		}
	}
	flush()
	return entries
}

func validateIssueRegistryEntries(path string, entries []issueRegistryEntry) []string {
	if len(entries) == 0 {
		return []string{path + " must contain issue entries"}
	}
	var gaps []string
	seen := map[string]bool{}
	numsByPrefix := map[string][]int{
		"P0":  {},
		"P1":  {},
		"P2":  {},
		"CTX": {},
	}
	for _, entry := range entries {
		if seen[entry.id] {
			gaps = append(gaps, path+" duplicate issue id "+entry.id)
		}
		seen[entry.id] = true
		match := issueRegistryIDPattern.FindStringSubmatch(entry.id)
		if match == nil {
			gaps = append(gaps, path+" invalid issue id "+entry.id)
			continue
		}
		num, _ := strconv.Atoi(match[2])
		numsByPrefix[match[1]] = append(numsByPrefix[match[1]], num)
		if !blockHasNonEmptyYAMLValue(entry.block, "title") {
			gaps = append(gaps, path+" "+entry.id+" missing title")
		}
		status, ok := blockYAMLValue(entry.block, "status")
		if !ok || status != "implemented" {
			gaps = append(gaps, path+" "+entry.id+" status must be implemented")
		}
		if !blockHasNonEmptyYAMLValue(entry.block, "command") {
			gaps = append(gaps, path+" "+entry.id+" missing command")
		}
		if !blockHasEvidence(entry.block) {
			gaps = append(gaps, path+" "+entry.id+" missing evidence")
		}
	}
	for _, prefix := range []string{"P0", "P1", "P2", "CTX"} {
		nums := numsByPrefix[prefix]
		if len(nums) == 0 {
			gaps = append(gaps, path+" missing "+prefix+"-001")
			continue
		}
		sort.Ints(nums)
		last := 0
		for _, num := range nums {
			if num == last {
				continue
			}
			if num != last+1 {
				gaps = append(gaps, fmt.Sprintf("%s %s ids must be contiguous; missing %s-%03d", path, prefix, prefix, last+1))
				break
			}
			last = num
		}
	}
	return gaps
}

func blockHasEvidence(block string) bool {
	value, ok := blockYAMLValue(block, "evidence")
	if ok && value != "" && value != "[]" {
		return true
	}
	return blockHasYAMLListItem(block, "evidence")
}

type yamlSequenceBlock struct {
	value string
	block string
}

func parseYAMLSequenceBlocks(text string, section string, key string) []yamlSequenceBlock {
	var entries []yamlSequenceBlock
	lines := strings.Split(text, "\n")
	inSection := false
	prefix := "- " + key + ":"
	var current *yamlSequenceBlock
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inSection {
			if trimmed == section+":" && line == trimmed {
				inSection = true
			}
			continue
		}
		if isTopLevelYAMLKey(line) {
			break
		}
		if strings.HasPrefix(trimmed, prefix) {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &yamlSequenceBlock{value: trimYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)))}
		}
		if current != nil {
			current.block += line + "\n"
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}
	return entries
}

func requireYAMLBlockValue(path string, label string, block string, field string, want string, gaps *[]string) {
	got, ok := blockYAMLValue(block, field)
	if !ok || got != want {
		*gaps = append(*gaps, fmt.Sprintf("%s %s must set %s: %s", path, label, field, want))
	}
}

func knownValidationRefs() map[string]bool {
	refs := knownMakeTargetRefs()
	for command := range knownGoalcliCommandRefs() {
		refs[command] = true
	}
	return refs
}

func knownGoalcliCommandRefs() map[string]bool {
	refs := scalarSet(commandRegistryRequiredCommands()...)
	for _, entry := range parseYAMLSequenceFileBlocks(".agent/registries/command-registry.yaml", "commands", "name") {
		if entry.value != "" {
			refs[entry.value] = true
		}
	}
	return refs
}

func knownMakeTargetRefs() map[string]bool {
	refs := scalarSet(requiredMakefileTargets()...)
	for _, target := range makefileTargets("Makefile") {
		refs[target] = true
	}
	return refs
}

func parseYAMLSequenceFileBlocks(path string, section string, key string) []yamlSequenceBlock {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseYAMLSequenceBlocks(string(content), section, key)
}

func makefileTargets(path string) []string {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var targets []string
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ".") || strings.HasPrefix(line, "\t") {
			continue
		}
		if i := strings.Index(trimmed, ":"); i > 0 {
			name := strings.TrimSpace(trimmed[:i])
			if name != "" && !strings.ContainsAny(name, " \t$") {
				targets = append(targets, name)
			}
		}
	}
	return targets
}

func knownEnforcementRef(ref string, commandNames map[string]bool, makeTargets map[string]bool) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	if ref == "goalcli" {
		info, err := os.Stat("cmd/goalcli/main.go")
		return err == nil && !info.IsDir()
	}
	if strings.HasPrefix(ref, "goalcli ") {
		fields := strings.Fields(ref)
		return len(fields) >= 2 && (commandNames[fields[1]] || makeTargets[fields[1]])
	}
	if strings.HasPrefix(ref, "make ") {
		fields := strings.Fields(ref)
		if len(fields) < 2 {
			return false
		}
		return makeTargets[fields[len(fields)-1]]
	}
	if strings.HasPrefix(ref, "scripts/") || strings.HasPrefix(ref, ".githooks/") {
		info, err := os.Stat(ref)
		return err == nil && !info.IsDir()
	}
	return commandNames[ref] || makeTargets[ref]
}

func blockHasNonEmptyYAMLValue(block string, key string) bool {
	value, ok := blockYAMLValue(block, key)
	return ok && value != "" && value != "[]"
}

func blockYAMLValue(block string, key string) (string, bool) {
	prefix := key + ":"
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return trimYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))), true
		}
	}
	return "", false
}

func blockHasYAMLListItem(block string, key string) bool {
	lines := strings.Split(block, "\n")
	inList := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !inList {
			if strings.HasPrefix(trimmed, key+":") {
				inList = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			return true
		}
		if strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "- ") {
			return false
		}
	}
	return false
}

func trimYAMLScalar(value string) string {
	if i := strings.Index(value, "#"); i >= 0 {
		value = value[:i]
	}
	value = strings.TrimSpace(value)
	return strings.Trim(value, `"'`)
}

func appendYAMLListDuplicateGaps(path string, field string, label string, gaps *[]string) {
	values, err := yamlListScalarValues(path, field)
	if err != nil {
		*gaps = append(*gaps, "read "+path+": "+err.Error())
		return
	}
	for _, duplicate := range duplicateValues(values) {
		*gaps = append(*gaps, fmt.Sprintf("%s duplicate %s %s", path, label, duplicate))
	}
}

func appendYAMLSequenceDuplicateGaps(path string, section string, label string, gaps *[]string) {
	values, err := yamlSequenceValuesInSection(path, section)
	if err != nil {
		*gaps = append(*gaps, "read "+path+": "+err.Error())
		return
	}
	for _, duplicate := range duplicateValues(values) {
		*gaps = append(*gaps, fmt.Sprintf("%s duplicate %s %s", path, label, duplicate))
	}
}

func appendYAMLMapSectionDuplicateGaps(path string, section string, label string, gaps *[]string) {
	values, err := yamlMapKeysInSection(path, section)
	if err != nil {
		*gaps = append(*gaps, "read "+path+": "+err.Error())
		return
	}
	for _, duplicate := range duplicateValues(values) {
		*gaps = append(*gaps, fmt.Sprintf("%s duplicate %s %s", path, label, duplicate))
	}
}

func yamlListScalarValues(path string, field string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	prefix := "- " + field + ":"
	var values []string
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			values = append(values, trimYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))))
		}
	}
	return values, nil
}

func yamlSequenceValuesInSection(path string, section string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var values []string
	inSection := false
	header := section + ":"
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !inSection {
			inSection = trimmed == header && line == trimmed
			continue
		}
		if isTopLevelYAMLKey(line) {
			break
		}
		if strings.HasPrefix(trimmed, "- ") {
			values = append(values, trimYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))))
		}
	}
	return values, nil
}

func yamlMapKeysInSection(path string, section string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var values []string
	inSection := false
	header := section + ":"
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !inSection {
			inSection = trimmed == header && line == trimmed
			continue
		}
		if isTopLevelYAMLKey(line) {
			break
		}
		if strings.HasPrefix(line, "  ") && strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "- ") {
			key := strings.TrimSpace(strings.SplitN(trimmed, ":", 2)[0])
			values = append(values, trimYAMLScalar(key))
		}
	}
	return values, nil
}

func isTopLevelYAMLKey(line string) bool {
	trimmed := strings.TrimSpace(line)
	return line == trimmed && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "- ")
}

func duplicateValues(values []string) []string {
	seen := map[string]bool{}
	reported := map[string]bool{}
	var duplicates []string
	for _, value := range values {
		if value == "" {
			continue
		}
		if seen[value] && !reported[value] {
			duplicates = append(duplicates, value)
			reported[value] = true
		}
		seen[value] = true
	}
	sort.Strings(duplicates)
	return duplicates
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

func plannedCommandVerifyRequested(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "--verify", "--strict", "--context=release_verify":
			return true
		}
		if arg == "--context" {
			continue
		}
	}
	for i, arg := range args {
		if arg == "--context" && i+1 < len(args) && args[i+1] == "release_verify" {
			return true
		}
	}
	return false
}

func emitPlannedReport(stdout io.Writer, stderr io.Writer, command, status string, details []string, gaps []string, args []string) int {
	exitCode := emitReport(stdout, command, status, details, gaps)
	if status == "planned" || status == "gap" {
		if plannedCommandVerifyRequested(args) {
			write(stderr, "ERROR: %s is %s under --verify/strict context\n", command, status)
		} else {
			write(stderr, "ERROR: %s is %s and cannot satisfy a release gate\n", command, status)
		}
	}
	return exitCode
}

// runRulesConsistencyCheck 校验 .agent/runtime/standard/goal-runtime-canonical.md（叙事层）
// 与 .agent/rules/iron-rules.md（机器层）引用的 RULE-* 编号集合一致，
// 并要求两侧引用的所有 RULE-* 都在 .agent/rules/registry.yaml 中登记。
//
// 用途：PR #36 引入双 SSOT 后，防止两份文档的铁律编号映射悄然漂移。
func runRulesConsistencyCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("rules-consistency-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("rules-consistency-check", err, stderr)
	}

	canonicalPath := ".agent/runtime/standard/goal-runtime-canonical.md"
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

	// 从 canonical 抽"八条铁律"段表格的 RULE-* ID
	canonRules := extractCanonicalIronRuleIDs(string(canonical))
	// 从 iron-rules 抽"七律"段括号内的 RULE-* ID
	ironRules := extractIronRulesIDs(string(iron))
	// 从 registry.yaml 抽所有 - id: RULE-* 行
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

	// 两侧引用集合（去重并集）必须各自完全包含在 registry 中
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

	// canonical 的核心铁律 RULE-* 必须全部出现在 iron-rules 中（canonical ⊆ iron），
	// 但反向不强求：iron-rules 每条会附带多个关联 RULE-*（主+关联），
	// 这些不必都出现在 canonical 表格里。
	for id := range canonRules {
		if !ironRules[id] {
			gaps = append(gaps, fmt.Sprintf("漂移：%s 引用 %s 但 %s 未引用", canonicalPath, id, ironPath))
		}
	}
	appendRulesRegistryEnforcedByGaps(registryPath, string(registry), &gaps)

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

func appendRulesRegistryEnforcedByGaps(registryPath string, registryText string, gaps *[]string) {
	for _, ref := range extractRegistryEnforcedByValues(registryText) {
		if ref == "goalcli" {
			continue
		}
		fields := strings.Fields(ref)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "goalcli" {
			if len(fields) < 2 {
				*gaps = append(*gaps, registryPath+" enforced_by "+ref+" is missing a goalcli command")
				continue
			}
			command := fields[1]
			if !rulesRegistryGoalCLICommands()[command] {
				*gaps = append(*gaps, registryPath+" enforced_by "+ref+" references unknown goalcli command "+command)
			}
			continue
		}
		if target, ok := parseMakeEnforcerTarget(fields); ok {
			if !rulesRegistryMakeTargets()[target] {
				*gaps = append(*gaps, registryPath+" enforced_by "+ref+" references unknown make target "+target)
			}
			continue
		}
		if strings.HasPrefix(ref, ".githooks/") {
			info, err := os.Stat(ref)
			if err != nil || info.IsDir() {
				*gaps = append(*gaps, registryPath+" enforced_by "+ref+" references missing hook")
			}
			continue
		}
		*gaps = append(*gaps, registryPath+" enforced_by "+ref+" is not a supported gate reference")
	}
}

func extractRegistryEnforcedByValues(text string) []string {
	re := regexp.MustCompile(`(?m)^\s*enforced_by:\s*(.+)$`)
	var values []string
	for _, match := range re.FindAllStringSubmatch(text, -1) {
		value := trimYAMLScalar(match[1])
		if value != "" && value != "[]" {
			values = append(values, value)
		}
	}
	return values
}

func rulesRegistryGoalCLICommands() map[string]bool {
	commands := scalarSet(commandRegistryRequiredCommands()...)
	if registered, err := yamlListScalarValues(".agent/registries/command-registry.yaml", "name"); err == nil {
		for _, command := range registered {
			commands[command] = true
		}
	}
	for _, command := range []string{
		"rules-consistency-check",
		"self-improving-check",
		"retro-check",
		"traceability-check",
		"debt-evidence",
		"debt-evidence-checksum-check",
		"debt-evidence-hash",
	} {
		commands[command] = true
	}
	return commands
}

func rulesRegistryMakeTargets() map[string]bool {
	targets := scalarSet(requiredMakefileTargets()...)
	if values, err := yamlSequenceValuesInSection(".agent/registries/makefile-target-registry.yaml", "targets"); err == nil {
		for _, target := range values {
			targets[target] = true
		}
	}
	return targets
}

func parseMakeEnforcerTarget(fields []string) (string, bool) {
	for i, field := range fields {
		if field != "make" {
			continue
		}
		for _, candidate := range fields[i+1:] {
			if candidate == "" || strings.HasPrefix(candidate, "-") || strings.Contains(candidate, "=") {
				continue
			}
			return candidate, true
		}
		return "", false
	}
	return "", false
}
