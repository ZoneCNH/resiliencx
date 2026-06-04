package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/ZoneCNH/xlib-standard/internal/releasequality"
)

func main() {
	exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

var exit = os.Exit

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		write(stderr, usage)
		return 2
	}
	name := args[0]
	switch name {
	case "help", "-h", "--help":
		write(stdout, usage)
		return 0
	default:
		cmd, ok := lookupCommand(name)
		if !ok {
			write(stderr, "unknown command %q\n", name)
			return 2
		}
		return cmd.Run(args[0], args[1:], stdin, stdout, stderr)
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

type externalCommand struct {
	name string
	args []string
}

func runExternalSequence(stdin io.Reader, stdout io.Writer, stderr io.Writer, commands ...externalCommand) int {
	for _, command := range commands {
		if code := runExternal(stdin, stdout, stderr, command.name, command.args...); code != 0 {
			return code
		}
	}
	return 0
}

func write(writer io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(writer, format, args...)
}

const usage = `usage: goalcli <command> [args]

commands:
  agent-team-contract [--dry-run]
  acceptance-matrix
  architecture [debt args]
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
  design-check [--json]
  downstream-adoption
  downstream-baseline
  downstream-registry
  downstream-debt [debt args]
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
  secrets
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
  traceability-check [--matrix .agent/traceability-matrix.md] [--json]
  upgrade-runtime [--dry-run]
  upgrade-standard [--dry-run]
  version [--json]
  worktree-check [--context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify]
  worktree-guard [--context local_write|local_readonly|ci_pull_request|ci_main_verify|release_verify]
`
