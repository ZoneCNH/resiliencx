package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestStandardImpactRequiresDownstreamSyncForHarnessGeneratorEvidence(t *testing.T) {
	report := runStandardImpact(t, []string{
		"scripts/check_docs.sh",
		"scripts/render_template.sh",
		"internal/tools/releasemanifest/main.go",
	})

	assertReportContains(t, report,
		"- downstream_sync_required: `true`",
		"- downstream_release_decision: `required`",
		"- changed_file_count: `3`",
		"## harness",
		"- `scripts/check_docs.sh`",
		"## generator",
		"- `scripts/render_template.sh`",
		"## evidence",
		"- `internal/tools/releasemanifest/main.go`",
		"- `required`",
	)
}

func TestStandardImpactRequiresDownstreamSyncForContextRuntimeV4Categories(t *testing.T) {
	report := runStandardImpact(t, []string{
		"cmd/goalcli/main.go",
		".agent/context/runtime.md",
		".agent/registries/command-registry.yaml",
		".agent/registries/issue-registry.yaml",
		".agent/registries/makefile-baseline.yaml",
		".agent/registries/makefile-target-registry.yaml",
		".github/CODEOWNERS",
		".github/dependabot.yml",
		".github/rulesets/default.yml",
		"infra/github-rules/default.yml",
		"templates/context-consumer/README.md",
	})

	assertReportContains(t, report,
		"- downstream_sync_required: `true`",
		"- context_runtime_change: `true`",
		"- governance_registry_change: `true`",
		"- downstream_release_decision: `required`",
		"- repository_rules_release_decision: `audit_required`",
		"- changed_file_count: `11`",
		"## context_runtime",
		"- `.agent/context/runtime.md`",
		"- `cmd/goalcli/main.go`",
		"## governance_registry",
		"- `.agent/registries/command-registry.yaml`",
		"- `.agent/registries/issue-registry.yaml`",
		"- `.agent/registries/makefile-baseline.yaml`",
		"- `.agent/registries/makefile-target-registry.yaml`",
		"## repository_rules",
		"- `.github/CODEOWNERS`",
		"- `.github/dependabot.yml`",
		"- `.github/rulesets/default.yml`",
		"- `infra/github-rules/default.yml`",
		"## downstream_context",
		"- `templates/context-consumer/README.md`",
		"- `required`",
		"- repository_rules: `audit_required`",
		"context_runtime、governance_registry",
		"downstream_context",
	)
}

func TestStandardImpactRequiresDownstreamSyncForDeletedImpactFiles(t *testing.T) {
	scriptsDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "check_standard_impact.sh")

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	if err := os.Mkdir(repoDir, 0o755); err != nil {
		t.Fatalf("create temp repo: %v", err)
	}

	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Standard Impact Test")
	runGit(t, repoDir, "config", "user.email", "standard-impact@example.com")
	writeFixtureFile(t, repoDir, "contracts/deleted.schema.json")
	writeFixtureFile(t, repoDir, ".agent/context/runtime.md")
	writeFixtureFile(t, repoDir, "internal/tools/releasemanifest/deleted.go")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "base")
	runGit(t, repoDir, "rm", "contracts/deleted.schema.json", ".agent/context/runtime.md", "internal/tools/releasemanifest/deleted.go")

	reportPath := filepath.Join(tempDir, "standard-impact.md")
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		"STANDARD_IMPACT_REPORT="+reportPath,
		"STANDARD_IMPACT_BASE=HEAD",
		"STANDARD_IMPACT_GENERATED_AT=2026-06-02T00:00:00Z",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standard impact check failed: %v\n%s", err, output)
	}

	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	report := string(reportBytes)

	assertReportContains(t, report,
		"- downstream_sync_required: `true`",
		"- context_runtime_change: `true`",
		"- downstream_release_decision: `required`",
		"- changed_file_count: `3`",
		"## contracts",
		"- `contracts/deleted.schema.json`",
		"## context_runtime",
		"- `.agent/context/runtime.md`",
		"## evidence",
		"- `internal/tools/releasemanifest/deleted.go`",
		"- `required`",
	)
}

func TestStandardImpactUsesUpstreamMergeBaseForCleanBranches(t *testing.T) {
	scriptsDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "check_standard_impact.sh")

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	if err := os.Mkdir(repoDir, 0o755); err != nil {
		t.Fatalf("create temp repo: %v", err)
	}

	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Standard Impact Test")
	runGit(t, repoDir, "config", "user.email", "standard-impact@example.com")
	writeFixtureFile(t, repoDir, "README.md")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "base")

	runGit(t, repoDir, "switch", "-c", "feature")
	runGit(t, repoDir, "branch", "--set-upstream-to=main")
	writeFixtureFile(t, repoDir, "scripts/check_docs.sh")
	writeFixtureFile(t, repoDir, "docs/standard/README.md")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "feature")

	reportPath := filepath.Join(tempDir, "standard-impact.md")
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		"STANDARD_IMPACT_REPORT="+reportPath,
		"STANDARD_IMPACT_BASE=",
		"STANDARD_IMPACT_GENERATED_AT=2026-06-02T00:00:00Z",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standard impact check failed: %v\n%s", err, output)
	}

	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}

	assertReportContains(t, string(report),
		"- generated_at: `2026-06-02T00:00:00Z`",
		"- downstream_sync_required: `true`",
		"- changed_file_count: `2`",
		"## docs",
		"- `docs/standard/README.md`",
		"## harness",
		"- `scripts/check_docs.sh`",
		"- `required`",
	)
}

func TestStandardImpactIgnoresLocalAgentRuntimeState(t *testing.T) {
	report := runStandardImpact(t, []string{
		".omc/state/mission-state.json",
		".omx/state/ralph-progress.json",
		".worktree/scratch/README.md",
		"docs/standard/README.md",
	})

	assertReportContains(t, report,
		"- changed_file_count: `1`",
		"## docs",
		"- `docs/standard/README.md`",
		"- `not_required`",
	)

	for _, localStatePath := range []string{".omc/", ".omx/", ".worktree/"} {
		if strings.Contains(report, localStatePath) {
			t.Fatalf("report included local runtime state %q:\n%s", localStatePath, report)
		}
	}
}

func TestStandardImpactCanonicalizesRetiredRuntimeRenamePairs(t *testing.T) {
	scriptsDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "check_standard_impact.sh")

	retiredGate := "xlib" + "gate"
	retiredKit := "goal" + "kit"
	retiredKitUpper := "GOAL" + "KIT"
	oldFiles := []string{
		"cmd/" + retiredGate + "/main.go",
		"internal/" + retiredGate + "/README.md",
		"contracts/" + retiredGate + "-report.schema.json",
		".agent/standard/" + retiredKit + "-" + retiredGate + "-mapping.md",
		"docs/adr/ADR-20260603-001-" + retiredKit + "-" + retiredGate + "-runtime.md",
		"docs/plans/" + retiredKit + "-v0.1.0-migration-index.md",
		"docs/plans/" + retiredKit + "-v0.1.0-roadmap.md",
		"docs/standard/" + retiredKit + "-runtime.md",
		"docs/standard/" + retiredGate + "-cli-contract.md",
		"release/evidence/" + retiredKit + "/GOAL-20260603-XLIB-" + retiredKitUpper + "-001.json",
	}
	newFiles := []string{
		"cmd/goalcli/main.go",
		"internal/goalcli/README.md",
		"contracts/goalcli-report.schema.json",
		".agent/docs/standard/goalcli-mapping.md",
		"docs/adr/ADR-20260603-001-goalcli-runtime.md",
		"docs/plans/goalcli-v0.1.0-migration-index.md",
		"docs/plans/goalcli-v0.1.0-roadmap.md",
		"docs/standard/goalcli-runtime.md",
		"docs/standard/goalcli-cli-contract.md",
		"release/evidence/goalcli/GOAL-20260603-XLIB-GOALCLI-001.json",
	}

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	if err := os.Mkdir(repoDir, 0o755); err != nil {
		t.Fatalf("create temp repo: %v", err)
	}

	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Standard Impact Test")
	runGit(t, repoDir, "config", "user.email", "standard-impact@example.com")
	for _, file := range oldFiles {
		writeFixtureFile(t, repoDir, file)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "base")
	rmArgs := append([]string{"rm", "--"}, oldFiles...)
	runGit(t, repoDir, rmArgs...)
	for _, file := range newFiles {
		writeFixtureFile(t, repoDir, file)
	}

	reportPath := filepath.Join(tempDir, "standard-impact.md")
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		"STANDARD_IMPACT_REPORT="+reportPath,
		"STANDARD_IMPACT_BASE=",
		"STANDARD_IMPACT_GENERATED_AT=2026-06-02T00:00:00Z",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standard impact check failed: %v\n%s", err, output)
	}

	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	report := string(reportBytes)
	lowerReport := strings.ToLower(report)
	if strings.Contains(lowerReport, retiredGate) || strings.Contains(lowerReport, retiredKit) || strings.Contains(report, retiredKitUpper) {
		t.Fatalf("report included retired authority name:\n%s", report)
	}

	assertReportContains(t, report,
		"- downstream_sync_required: `true`",
		"- context_runtime_change: `true`",
		"- changed_file_count: `10`",
		"## docs",
		"- `.agent/docs/standard/goalcli-mapping.md`",
		"- `docs/adr/ADR-20260603-001-goalcli-runtime.md`",
		"- `docs/plans/goalcli-v0.1.0-migration-index.md`",
		"- `docs/plans/goalcli-v0.1.0-roadmap.md`",
		"- `docs/standard/goalcli-runtime.md`",
		"## contracts",
		"- `contracts/goalcli-report.schema.json`",
		"## context_runtime",
		"- `cmd/goalcli/main.go`",
		"- `docs/standard/goalcli-cli-contract.md`",
		"## evidence",
		"- `release/evidence/goalcli/GOAL-20260603-XLIB-GOALCLI-001.json`",
		"## other",
		"- `internal/goalcli/README.md`",
	)
}

func TestStandardImpactSortsCommittedAndWorktreeChanges(t *testing.T) {
	scriptsDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "check_standard_impact.sh")

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	if err := os.Mkdir(repoDir, 0o755); err != nil {
		t.Fatalf("create temp repo: %v", err)
	}

	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Standard Impact Test")
	runGit(t, repoDir, "config", "user.email", "standard-impact@example.com")
	writeFixtureFile(t, repoDir, "README.md")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "base")

	runGit(t, repoDir, "switch", "-c", "feature")
	runGit(t, repoDir, "branch", "--set-upstream-to=main")
	writeFixtureFile(t, repoDir, "renovate.json")
	writeFixtureFile(t, repoDir, "scripts/check_dependency_diff_test.go")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "feature")
	writeFixtureFile(t, repoDir, ".github/dependabot.yml")
	writeFixtureFile(t, repoDir, ".gitignore")

	reportPath := filepath.Join(tempDir, "standard-impact.md")
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		"STANDARD_IMPACT_REPORT="+reportPath,
		"STANDARD_IMPACT_BASE=",
		"STANDARD_IMPACT_GENERATED_AT=2026-06-02T00:00:00Z",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standard impact check failed: %v\n%s", err, output)
	}

	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	report := string(reportBytes)

	assertReportContains(t, report,
		"- downstream_sync_required: `true`",
		"- downstream_release_decision: `required`",
		"- repository_rules_release_decision: `audit_required`",
		"- changed_file_count: `4`",
		"## repository_rules",
		"- `.github/dependabot.yml`",
		"## other",
	)
	assertReportOrder(t, report,
		"- `.github/dependabot.yml`",
		"- `.gitignore`",
		"- `renovate.json`",
		"- `scripts/check_dependency_diff_test.go`",
	)
}

func TestStandardImpactDoesNotRequireDownstreamSyncForDocsOnly(t *testing.T) {
	report := runStandardImpact(t, []string{
		"README.md",
		"docs/standard/README.md",
	})

	assertReportContains(t, report,
		"- downstream_sync_required: `false`",
		"- downstream_release_decision: `not_required`",
		"- repository_rules_release_decision: `not_required`",
		"- changed_file_count: `2`",
		"## docs",
		"- `README.md`",
		"- `docs/standard/README.md`",
		"- `not_required`",
	)

	if strings.Contains(report, "- downstream_release_decision: `required`") {
		t.Fatalf("docs-only report unexpectedly required downstream sync:\n%s", report)
	}
}

func TestStandardImpactReportIncludesDecisionEvidence(t *testing.T) {
	report := runStandardImpact(t, []string{
		"contracts/template.md",
		"docs/supply-chain.md",
		"pkg/resiliencx/client.go",
	})

	assertReportContains(t, report,
		"- primary_downstream: `github.com/ZoneCNH/kernel`",
		"- context_runtime_change: `false`",
		"- governance_registry_change: `false`",
		"- downstream_release_decision: `required`",
		"- repository_rules_release_decision: `not_required`",
		"- changed_file_count: `3`",
		"## contracts",
		"- `contracts/template.md`",
		"## docs",
		"- `docs/supply-chain.md`",
		"## other",
		"- `pkg/resiliencx/client.go`",
		"## Sync Decision",
	)
}

func TestStandardImpactDownstreamsMatchRegisteredTargets(t *testing.T) {
	publicTargets := parseAdoptionStatusTargets(t)
	matrixTargets := parseDownstreamMatrixTargets(t)
	if diff := diffNameSets(publicTargets, matrixTargets); diff != "" {
		t.Fatalf("downstream matrix targets differ from adoption status: %s", diff)
	}

	expectedImpactTargets := append([]string{}, publicTargets...)
	expectedImpactTargets = append(expectedImpactTargets, "x.go")
	expectedImpactTargets = uniqueSortedNames(expectedImpactTargets)

	impactTargets := parseStandardImpactDownstreams(t)
	if diff := diffNameSets(expectedImpactTargets, impactTargets); diff != "" {
		t.Fatalf("standard impact downstream targets differ from registered targets plus x.go: %s", diff)
	}

	for _, privateBusinessTarget := range []string{
		"market-data",
		"market-engine",
		"macro-data",
		"macro-engine",
		"regime-engine",
	} {
		if containsName(impactTargets, privateBusinessTarget) {
			t.Fatalf("standard impact targets included private L3 business system %q", privateBusinessTarget)
		}
	}
}

func runStandardImpact(t *testing.T, changedFiles []string) string {
	t.Helper()

	scriptsDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	scriptPath := filepath.Join(scriptsDir, "check_standard_impact.sh")

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "repo")
	if err := os.Mkdir(repoDir, 0o755); err != nil {
		t.Fatalf("create temp repo: %v", err)
	}

	initCmd := exec.Command("git", "init")
	initCmd.Dir = repoDir
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	for _, file := range changedFiles {
		path := filepath.Join(repoDir, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create parent dir for %s: %v", file, err)
		}
		if err := os.WriteFile(path, []byte("test fixture\n"), 0o644); err != nil {
			t.Fatalf("write changed file %s: %v", file, err)
		}
	}

	reportPath := filepath.Join(tempDir, "standard-impact.md")
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		"STANDARD_IMPACT_REPORT="+reportPath,
		"STANDARD_IMPACT_BASE=",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("standard impact check failed: %v\n%s", err, output)
	}

	report, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	return string(report)
}

func parseStandardImpactDownstreams(t *testing.T) []string {
	t.Helper()

	script, err := os.ReadFile("check_standard_impact.sh")
	if err != nil {
		t.Fatalf("read standard impact script: %v", err)
	}

	modulePattern := regexp.MustCompile(`"github\.com/ZoneCNH/([^"]+)"`)
	var targets []string
	inDownstreamArray := false
	for _, line := range strings.Split(string(script), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "downstreams=(" {
			inDownstreamArray = true
			continue
		}
		if inDownstreamArray && trimmed == ")" {
			break
		}
		if !inDownstreamArray {
			continue
		}
		match := modulePattern.FindStringSubmatch(trimmed)
		if match == nil {
			continue
		}
		targets = append(targets, match[1])
	}
	if len(targets) == 0 {
		t.Fatalf("standard impact script has no downstream targets")
	}
	return uniqueSortedNames(targets)
}

func parseAdoptionStatusTargets(t *testing.T) []string {
	t.Helper()

	status, err := os.ReadFile(filepath.Join("..", ".agent", "registries", "downstream-adoption-status.yaml"))
	if err != nil {
		t.Fatalf("read downstream adoption status: %v", err)
	}

	var targets []string
	inStandardTargets := false
	for _, line := range strings.Split(string(status), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "standard_target_libraries:" {
			inStandardTargets = true
			continue
		}
		if inStandardTargets && trimmed != "" && !strings.HasPrefix(line, " ") {
			break
		}
		if !inStandardTargets || !strings.HasPrefix(trimmed, "- name: ") {
			continue
		}
		targets = append(targets, strings.TrimSpace(strings.TrimPrefix(trimmed, "- name: ")))
	}
	if len(targets) == 0 {
		t.Fatalf("downstream adoption status has no standard target libraries")
	}
	return uniqueSortedNames(targets)
}

func parseDownstreamMatrixTargets(t *testing.T) []string {
	t.Helper()

	matrix, err := os.ReadFile(filepath.Join("..", "docs", "downstream-matrix.md"))
	if err != nil {
		t.Fatalf("read downstream matrix: %v", err)
	}

	rowPattern := regexp.MustCompile("^\\| `[^`]+` \\| `github\\.com/ZoneCNH/([^`]+)` \\|")
	var targets []string
	for _, line := range strings.Split(string(matrix), "\n") {
		match := rowPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		targets = append(targets, match[1])
	}
	if len(targets) == 0 {
		t.Fatalf("downstream matrix has no target rows")
	}
	return uniqueSortedNames(targets)
}

func diffNameSets(want, got []string) string {
	wantSet := nameSet(want)
	gotSet := nameSet(got)

	var missing []string
	for name := range wantSet {
		if !gotSet[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	var extra []string
	for name := range gotSet {
		if !wantSet[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)

	if len(missing) == 0 && len(extra) == 0 {
		return ""
	}
	parts := make([]string, 0, 2)
	if len(missing) > 0 {
		parts = append(parts, "missing "+strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		parts = append(parts, "extra "+strings.Join(extra, ", "))
	}
	return strings.Join(parts, "; ")
}

func uniqueSortedNames(names []string) []string {
	keys := nameSet(names)
	result := make([]string, 0, len(keys))
	for name := range keys {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func nameSet(names []string) map[string]bool {
	result := make(map[string]bool, len(names))
	for _, name := range names {
		result[name] = true
	}
	return result
}

func containsName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}

func writeFixtureFile(t *testing.T, repoDir, file string) {
	t.Helper()

	path := filepath.Join(repoDir, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent dir for %s: %v", file, err)
	}
	if err := os.WriteFile(path, []byte("test fixture\n"), 0o644); err != nil {
		t.Fatalf("write fixture file %s: %v", file, err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func assertReportContains(t *testing.T, report string, want ...string) {
	t.Helper()

	for _, text := range want {
		if !strings.Contains(report, text) {
			t.Fatalf("report missing %q:\n%s", text, report)
		}
	}
}

func assertReportOrder(t *testing.T, report string, want ...string) {
	t.Helper()

	previousIndex := -1
	for _, text := range want {
		index := strings.Index(report, text)
		if index < 0 {
			t.Fatalf("report missing %q:\n%s", text, report)
		}
		if index <= previousIndex {
			t.Fatalf("report order mismatch at %q:\n%s", text, report)
		}
		previousIndex = index
	}
}
