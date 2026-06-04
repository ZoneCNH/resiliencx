package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var plannedCommandFiles = map[string][]string{
	"minimal-kernel":           {".agent/minimal-kernel.yaml"},
	"done-assertion":           {".agent/done-assertion.yaml"},
	"agent-team-contract":      {".agent/team-contract.yaml"},
	"scope-lock":               {".agent/scope-locks.yaml"},
	"pr-template":              {".agent/pr-template-contract.yaml", ".github/pull_request_template.md"},
	"acceptance-matrix":        {".agent/acceptance-matrix.yaml"},
	"runtime-health":           {".agent/runtime-health.yaml"},
	"goal-runtime":             {".agent/goal-runtime.md", ".agent/harness.yaml"},
	"goal-acceptance":          {".agent/harness.yaml"},
	"goal-delivery":            {".agent/harness.yaml"},
	"goal-handover":            {".agent/harness.yaml"},
	"goal-downstream-adoption": {".agent/harness.yaml"},
	"goal-certify":             {".agent/harness.yaml"},
	"goal-runtime-final":       {".agent/harness.yaml"},
	"naming":                   {"docs/standard/repository-roles.md", "docs/standard/module-boundary.md"},
	"upgrade-standard":         {".agent/downstream-registry.yaml"},
	"conformance-profile":      {".agent/conformance-profiles.yaml"},
	"downstream-registry":      {".agent/downstream-registry.yaml"},
	"self-healing-skeleton":    {".agent/failure-taxonomy.yaml", ".agent/root-cause.yaml", ".agent/regression-memory.yaml"},
	"policy-schema":            {".agent/policy-schema.yaml"},
	"github-settings":          {".agent/github-settings.yaml"},
	"github-governance":        {".agent/github-governance.yaml"},
	"governance-fixture-test":  {".agent/governance-fixture-test.yaml"},
	"toolchain":                {".agent/toolchain.yaml"},
	"evidence-artifacts":       {".agent/evidence-artifact-policy.yaml"},
	"install-runtime":          {".agent/runtime-install.yaml"},
	"upgrade-runtime":          {".agent/runtime-upgrade.yaml"},
	"release-ready":            {".agent/release-readiness-formula.yaml"},
	"evidence-replay":          {".agent/evidence-replay.yaml"},
	"attest-conformance":       {".agent/conformance-profiles.yaml"},
	"pack-standard":            {".agent/standard-pack.yaml"},
	"pack-gate":                {".agent/gate-pack.yaml"},
	"pack-evidence":            {".agent/evidence-pack.yaml"},
	"runtime-file-ownership":   {".agent/runtime-file-ownership.yaml"},
	"downstream-baseline":      {".agent/downstream-baseline-scan.yaml", ".agent/downstream-registry.yaml"},
	"downstream-adoption":      {".agent/downstream-adoption-modes.yaml", ".agent/downstream-registry.yaml"},
	"autoresearch":             {".agent/autoresearch.yaml"},
	"changelog":                {".agent/changelog.yaml"},
	"supply-chain":             {"docs/supply-chain.md"},
	"execution-context":        {".agent/execution-context.yaml", "contracts/execution-context.schema.json"},
}

var plannedCommandSemanticMarkers = map[string]map[string][]string{
	"agent-team-contract": {
		".agent/team-contract.yaml": {"schema_version:", "roles:", "rule:"},
	},
	"acceptance-matrix": {
		".agent/acceptance-matrix.yaml": {"schema_version:", "acceptance:"},
	},
	"runtime-health": {
		".agent/runtime-health.yaml": {"schema_version:", "checks:", "toolchain"},
	},
	"goal-acceptance": {
		".agent/harness.yaml": {"goalcli_mva_gates:", "G12_ACCEPTANCE", "goal-acceptance"},
	},
	"goal-delivery": {
		".agent/harness.yaml": {"goalcli_mva_gates:", "G13_DELIVERY", "goal-delivery"},
	},
	"goal-handover": {
		".agent/harness.yaml": {"goalcli_mva_gates:", "G14_HANDOVER", "goal-handover"},
	},
	"goal-downstream-adoption": {
		".agent/harness.yaml": {"goalcli_mva_gates:", "G15_DOWNSTREAM_ADOPTION", "goal-downstream-adoption"},
	},
	"goal-certify": {
		".agent/harness.yaml": {"goalcli_mva_gates:", "G16_CERTIFY", "goal-certify"},
	},
	"goal-runtime-final": {
		".agent/harness.yaml": {"goalcli_mva_gates:", "G12_G16_FINAL", "goal-runtime-final"},
	},
	"execution-context": {
		".agent/execution-context.yaml": {"schema_version:", "contexts:", "local_write", "ci_pull_request", "release_verify"},
	},
	"runtime-file-ownership": {
		".agent/runtime-file-ownership.yaml": {"schema_version:", "owners:", "owner:", "review_required:", "rationale:"},
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
		details = append(details, "found "+path)
		gaps = append(gaps, validatePlannedCommandFile(command, path, content)...)
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
	return gaps
}

func plannedCommandMarkers(command string, path string) []string {
	files, ok := plannedCommandSemanticMarkers[command]
	if !ok {
		return nil
	}
	return files[path]
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
