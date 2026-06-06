package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ZoneCNH/resiliencx/internal/releasequality"
)

func main() {
	exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

var exit = os.Exit

const (
	enableVulncheckEnv        = "XLIB_ENABLE_VULNCHECK"
	forceVulncheckEnv         = "XLIB_FORCE_VULNCHECK"
	vulncheckIntervalHoursEnv = "XLIB_VULNCHECK_INTERVAL_HOURS"
	vulncheckStateEnv         = "XLIB_VULNCHECK_STATE"
	defaultVulncheckInterval  = 7 * 24 * time.Hour
	defaultVulncheckStatePath = ".cache/security/govulncheck-last-run"
)

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		write(stderr, usage)
		return 2
	}
	switch args[0] {
	case "version":
		return runVersion(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "main-guard":
		return runMainGuard(args[1:], stdout, stderr)
	case "worktree-guard":
		return runWorktreeGuard(args[1:], stdout, stderr)
	case "worktree-check":
		return runWorktreeCheck(args[1:], stdout, stderr)
	case "context-check":
		return runContextCheck(args[1:], stdout, stderr)
	case "spec-check":
		return runSpecCheck(args[1:], stdout, stderr)
	case "design-check":
		return runDesignCheck(args[1:], stdout, stderr)
	case "task-check":
		return runTaskCheck(args[1:], stdout, stderr)
	case "pr-check":
		return runPRCheck(args[1:], stdin, stdout, stderr)
	case "evidence-check":
		return runEvidenceCheck(args[1:], stdout, stderr)
	case "cli-contract":
		return runCLIContract(args[1:], stdout, stderr)
	case "issue-registry":
		return runIssueRegistry(args[1:], stdout, stderr)
	case "command-registry":
		return runCommandRegistry(args[1:], stdout, stderr)
	case "makefile-baseline":
		return runMakefileBaseline(args[1:], stdout, stderr)
	case "adoption-check":
		return runAdoptionCheck(args[1:], stdout, stderr)
	case "audit-goal":
		return runAuditGoal(args[1:], stdout, stderr)
	case "dashboard-generate":
		return runDashboardGenerate(args[1:], stdout, stderr)
	case "context-profile":
		return runContextProfile(args[1:], stdout, stderr)
	case "context-profile-check":
		return runContextProfileCheck("context-profile-check", args[1:], stdout, stderr)
	case "context-schema-check":
		return runContextProfileCheck("context-schema-check", args[1:], stdout, stderr)
	case "schema":
		return runSchemaCommand(args[1:], stdout, stderr)
	case "schema-check":
		return runSchemaCheck(args[1:], stdout, stderr)
	case "context-lite", "context-standard", "context-full", "context-release", "context-fast-check", "context-standard-check", "context-full-check":
		return runContextProfileAlias(args[0], args[1:], stdout, stderr)
	case "debt":
		return runDebt(args[1:], stdout, stderr)
	case "architecture":
		return runDebtAlias("architecture", "enforce", args[1:], stdout, stderr)
	case "domain":
		return runDebtAlias("domain", "enforce", args[1:], stdout, stderr)
	case "docs-drift":
		return runDebtAlias("docs", "warn", args[1:], stdout, stderr)
	case "dependency-debt":
		return runDebtAlias("dependency", "warn", args[1:], stdout, stderr)
	case "testing-debt":
		return runDebtAlias("testing", "warn", args[1:], stdout, stderr)
	case "implementation-debt":
		return runDebtAlias("implementation", "observe", args[1:], stdout, stderr)
	case "security-debt":
		return runDebtAlias("security", "warn", args[1:], stdout, stderr)
	case "downstream-debt":
		return runDebtAlias("downstream", "warn", args[1:], stdout, stderr)
	case "downstream-sync-plan":
		return runDownstreamSyncPlan(args[1:], stdout, stderr)
	case "goal-acceptance", "goal-delivery", "goal-handover", "goal-downstream-adoption", "goal-certify", "goal-runtime-final":
		return runGoalRuntimeCommand(args[0], args[1:], stdout, stderr)
	case "minimal-kernel", "done-assertion", "agent-team-contract", "scope-lock", "pr-template", "acceptance-matrix", "runtime-health", "goal-runtime", "naming", "upgrade-standard", "conformance-profile", "downstream-registry", "self-healing-skeleton", "policy-schema", "github-settings", "toolchain", "evidence-artifacts", "install-runtime", "upgrade-runtime", "release-ready", "evidence-replay", "attest-conformance", "pack-standard", "pack-gate", "pack-evidence", "runtime-file-ownership", "downstream-baseline", "downstream-adoption", "autoresearch", "changelog", "github-governance", "governance-fixture-test", "supply-chain", "execution-context":
		return runPlannedCommand(args[0], args[1:], stdout, stderr)
	case "boundary":
		return runExternal(stdin, stdout, stderr, "./scripts/check_boundary.sh")
	case "contracts":
		return runExternal(stdin, stdout, stderr, "./scripts/check_contracts.sh")
	case "debt-evidence":
		return runDebtEvidence(args[1:], stdout, stderr)
	case "debt-evidence-checksum-check":
		return runExternal(stdin, stdout, stderr, "./scripts/hash_release_evidence.sh", "--check", "release/debt/latest.json", "release/debt/latest.json.sha256")
	case "debt-evidence-hash":
		return runExternal(stdin, stdout, stderr, "./scripts/hash_release_evidence.sh", "release/debt/latest.json", "release/debt/latest.json.sha256")
	case "dependency-check":
		return runExternal(stdin, stdout, stderr, "./scripts/check_dependency_diff.sh")
	case "docs-check":
		return runExternal(stdin, stdout, stderr, "./scripts/check_docs.sh")
	case "docker-toolchain-check":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/check_toolchain.sh", args[1:]...)
	case "docker-drift-check":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/check_toolchain.sh", append([]string{"--drift"}, args[1:]...)...)
	case "docker-build":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"build"}, args[1:]...)...)
	case "docker-build-check":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"build-check"}, args[1:]...)...)
	case "docker-shell":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"shell"}, args[1:]...)...)
	case "docker-ci":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"ci"}, args[1:]...)...)
	case "docker-release-check":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"release-check"}, args[1:]...)...)
	case "docker-release-final-check":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"release-final-check"}, args[1:]...)...)
	case "docker-goalcli":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"goalcli"}, args[1:]...)...)
	case "docker-goalcli-image":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"goalcli-image"}, args[1:]...)...)
	case "docker-goalcli-version":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"goalcli-version"}, args[1:]...)...)
	case "docker-runtime-check":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"runtime-check"}, args[1:]...)...)
	case "docker-contract":
		return runExternal(stdin, stdout, stderr, "./scripts/docker/docker_gate.sh", append([]string{"contract"}, args[1:]...)...)
	case "evidence", "manifest":
		return runExternal(stdin, stdout, stderr, "go", "run", "./internal/tools/releasemanifest", "--out", "release/manifest/latest.json")
	case "integration":
		return runExternal(stdin, stdout, stderr, "./scripts/run_integration.sh")
	case "release-evidence-check":
		return runExternal(stdin, stdout, stderr, "./scripts/check_release_evidence.sh")
	case "release-evidence-checksum-check":
		return runExternal(stdin, stdout, stderr, "./scripts/hash_release_evidence.sh", "--check")
	case "release-evidence-hash":
		return runExternal(stdin, stdout, stderr, "./scripts/hash_release_evidence.sh")
	case "release-final-check":
		return runExternal(stdin, stdout, stderr, "make", "release-final-check")
	case "render-check":
		return runExternal(stdin, stdout, stderr, "./scripts/check_rendered_template.sh", args[1:]...)
	case "rules-consistency-check":
		return runRulesConsistencyCheck(args[1:], stdout, stderr)
	case "rules-verify":
		return runExternal(stdin, stdout, stderr, "python3", "scripts/verify_rules.py")
	case "score":
		return runScore(args[1:], stdout, stderr)
	case "secrets":
		return runExternal(stdin, stdout, stderr, "./scripts/check_secrets.sh", args[1:]...)
	case "secret-check":
		return runExternal(stdin, stdout, stderr, "./scripts/check_secrets.sh", args[1:]...)
	case "secret":
		return runSecretCommand(args[1:], stdin, stdout, stderr)
	case "security":
		return runSecurity(stdin, stdout, stderr)
	case "standard-impact-check":
		return runExternal(stdin, stdout, stderr, "./scripts/check_standard_impact.sh")
	case "self-improving-check", "retro-check":
		return runSelfImprovingCheck(args[0], args[1:], stdout, stderr)
	case "traceability-check":
		return runTraceabilityCheck(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		write(stdout, usage)
		return 0
	default:
		write(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func runSecurity(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if os.Getenv(enableVulncheckEnv) != "1" {
		write(stderr, "security: govulncheck suspended; set %s=1 to run vulnerability scan\n", enableVulncheckEnv)
		return runExternal(stdin, stdout, stderr, "./scripts/check_secrets.sh")
	}

	now := time.Now().UTC()
	due, statePath, interval, err := vulncheckDue(now)
	if err != nil {
		write(stderr, "ERROR: %v\n", err)
		return 2
	}
	if due {
		if code := runExternal(stdin, stdout, stderr, "govulncheck", "./..."); code != 0 {
			return code
		}
		if err := recordVulncheckRun(statePath, now); err != nil {
			write(stderr, "ERROR: %v\n", err)
			return 1
		}
	} else {
		write(stderr, "security: govulncheck skipped; last successful scan is within %s; set %s=1 to force\n", formatDuration(interval), forceVulncheckEnv)
	}

	return runExternal(stdin, stdout, stderr, "./scripts/check_secrets.sh")
}

func vulncheckDue(now time.Time) (bool, string, time.Duration, error) {
	interval, err := vulncheckInterval()
	if err != nil {
		return false, "", 0, err
	}
	statePath := os.Getenv(vulncheckStateEnv)
	if statePath == "" {
		statePath = defaultVulncheckStatePath
	}
	if os.Getenv(forceVulncheckEnv) == "1" {
		return true, statePath, interval, nil
	}

	data, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return true, statePath, interval, nil
	}
	if err != nil {
		return false, statePath, interval, fmt.Errorf("read govulncheck state %s: %w", statePath, err)
	}

	lastRun, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(data)))
	if parseErr == nil {
		return !now.Before(lastRun.UTC().Add(interval)), statePath, interval, nil
	}
	return true, statePath, interval, nil
}

func vulncheckInterval() (time.Duration, error) {
	raw := os.Getenv(vulncheckIntervalHoursEnv)
	if raw == "" {
		return defaultVulncheckInterval, nil
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || hours <= 0 {
		return 0, fmt.Errorf("%s must be positive hours", vulncheckIntervalHoursEnv)
	}
	return time.Duration(hours) * time.Hour, nil
}

func recordVulncheckRun(statePath string, now time.Time) error {
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return fmt.Errorf("create govulncheck state directory: %w", err)
	}
	if err := os.WriteFile(statePath, []byte(now.Format(time.RFC3339Nano)+"\n"), 0o644); err != nil {
		return fmt.Errorf("write govulncheck state %s: %w", statePath, err)
	}
	return nil
}

func formatDuration(duration time.Duration) string {
	if duration%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(duration/time.Hour))
	}
	return duration.String()
}

func runSecretCommand(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		write(stderr, "usage: goalcli secret check [root]\n")
		return 2
	}
	switch args[0] {
	case "check":
		return runExternal(stdin, stdout, stderr, "./scripts/check_secrets.sh", args[1:]...)
	case "help", "-h", "--help":
		write(stdout, "usage: goalcli secret check [root]\n")
		return 0
	default:
		write(stderr, "unknown secret command %q\n", args[0])
		return 2
	}
}

func runScore(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli score", flag.ContinueOnError)
	flags.SetOutput(stderr)
	minimum := flags.Float64("min", releasequality.DefaultMinimum, "minimum acceptable release score")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	report := computeReleaseQuality(*minimum)
	data, err := marshalReleaseQuality(report)
	if err != nil {
		write(stderr, "ERROR: %v\n", err)
		return 1
	}
	write(stdout, "%s\n", data)
	if err := verifyReleaseQuality(report, *minimum); err != nil {
		write(stderr, "ERROR: %v\n", err)
		return 1
	}
	return 0
}

var (
	computeReleaseQuality = releasequality.Compute
	marshalReleaseQuality = releasequality.Marshal
	verifyReleaseQuality  = releasequality.Verify
)

func runExternal(stdin io.Reader, stdout io.Writer, stderr io.Writer, name string, args ...string) int {
	cmd := exec.Command(name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return exitError.ExitCode()
		}
		write(stderr, "ERROR: %v\n", err)
		return 1
	}
	return 0
}

func write(writer io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(writer, format, args...)
}

const usage = `usage: goalcli <command> [args]

commands:
  agent-team-contract [--dry-run]
  adoption-check [--verify] [--json] [--root <path>]
  acceptance-matrix
  architecture [debt args]
  audit-goal [--goal-id <id>] [--matrix .agent/traceability/traceability-matrix.md] [--json]
  dashboard-generate [--goal-id <id>] [--matrix .agent/traceability/traceability-matrix.md] [--format json|markdown]
  attest-conformance [--profile <name>]
  autoresearch
  boundary
  changelog
  cli-contract [--json|--output <path>|--explain]
  command-registry
  conformance-profile [--profile <name>]
  context-check [--json]
  context-fast-check
  context-full
  context-full-check
  context-lite
  context-profile [--profile <name>] [--json]
  context-profile-check [--json]
  context-release
  context-schema-check [--json]
  schema validate --all|--fixture <dir> [--report <path>] [--json]
  schema-check [--all|--fixture <dir>] [--report <path>] [--json]
  context-standard
  context-standard-check
  contracts
  debt [--config <path>] [--section <name>] [--mode <enforce|warn|observe>] [--min-score <score>] [--output json|markdown]
  debt lifecycle-check [--output <path>]
  debt patch-suggest [--output <path>]
  debt register-update [--output <path>]
  debt trend [--output <path>]
  debt-evidence
  debt-evidence-checksum-check
  debt-evidence-hash
  dependency-debt [debt args]
  dependency-check
  docs-drift [debt args]
  domain [debt args]
  doctor [--json]
  docs-check
  docker-build
  docker-build-check
  docker-ci
  docker-contract
  docker-drift-check
  docker-goalcli
  docker-goalcli-image
  docker-goalcli-version
  docker-release-check
  docker-release-final-check
  docker-runtime-check
  docker-shell
  docker-toolchain-check
  design-check [--json]
  downstream-adoption
  downstream-baseline
  downstream-registry
  downstream-debt [debt args]
  downstream-sync-plan [--impact-report <path>] [--output <path>|-] [--workspace-root <path>] [--format markdown|json]
  evidence
  evidence-artifacts
  evidence-check
  evidence-replay
  execution-context
  github-governance
  github-settings [--verify]
  goal-acceptance [--goal-id <id>] [--json]
  goal-delivery [--goal-id <id>] [--json]
  goal-handover [--goal-id <id>] [--json]
  goal-downstream-adoption [--goal-id <id>] [--json]
  goal-certify [--goal-id <id>] [--json]
  goal-runtime
  goal-runtime-final [--goal-id <id>] [--json] [--write-evidence]
  governance-fixture-test
  install-runtime [--dry-run]
  integration
  implementation-debt [debt args]
  issue-registry
  main-guard [--context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify]
  makefile-baseline
  manifest
  minimal-kernel
  done-assertion
  naming
  pack-evidence
  pack-gate
  pack-standard
  policy-schema
  pr-check [--context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify] [--dry-run] [--json]
  pr-template
  release-evidence-check
  release-evidence-checksum-check
  release-evidence-hash
  release-final-check
  release-ready
  render-check <rendered-dir>
  retro-check [--root <path>] [--strict]
  rules-consistency-check
  rules-verify
  runtime-file-ownership
  runtime-health
  scope-lock
  score [--min <score>]
  secret check [root]
  secret-check [root]
  secrets [root]
  security
  security-debt [debt args]
  self-improving-check [--root <path>] [--strict]
  self-healing-skeleton
  spec-check [--json]
  standard-impact-check
  supply-chain
  task-check [--json]
  toolchain
  testing-debt [debt args]
  traceability-check [--matrix .agent/traceability/traceability-matrix.md] [--json]
  upgrade-runtime [--dry-run]
  upgrade-standard [--dry-run]
  version [--json]
  worktree-check [--context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify]
  worktree-guard [--context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify]
`
