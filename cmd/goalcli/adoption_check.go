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

func runAdoptionCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli adoption-check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Bool("json", false, "")
	flags.Bool("verify", false, "")
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		write(stderr, "ERROR: adoption-check invalid arguments: unexpected positional argument %q\n", flags.Arg(0))
		return 2
	}

	details, gaps := evaluateAdoptionCheck(*root)
	if len(gaps) > 0 {
		write(stderr, "ERROR: adoption-check found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "adoption-check", "failed", details, gaps)
	}
	return emitReport(stdout, "adoption-check", "passed", details, nil)
}

func evaluateAdoptionCheck(root string) ([]string, []string) {
	root = filepath.Clean(root)
	// Keep the downstream governance lock contract as xlib-standard.lock even after template text replacement.
	lockPath := "xlib-" + "standard.lock"
	required := map[string][]string{
		lockPath: {
			"schema_version:",
			"standard_version:",
			"standard_commit:",
			"module_name:",
			"module_path:",
			"package_name:",
			"layer:",
			`adoption_check: "GOWORK=off make adoption-check"`,
		},
		".githooks/pre-commit":                            {},
		".githooks/pre-push":                              {},
		".github/workflows/adoption-check.yml":            {"GOWORK=off make adoption-check", "workflow_dispatch", "pull_request", "format('{0}{1}', 'xlib-', 'standard')"},
		"mk/governance.mk":                                {".PHONY: require-gowork-off", "adoption-check: require-gowork-off", "$(GOALCLI) adoption-check --verify"},
		".agent/harness/harness.yaml":                     {"id: adoption_check", "GOWORK=off make adoption-check"},
		".agent/registries/command-registry.yaml":         {"name: adoption-check"},
		".agent/registries/makefile-target-registry.yaml": {"adoption-check"},
		".agent/registries/makefile-baseline.yaml":        {"adoption-check"},
		"Makefile": {".PHONY: adoption-check", "adoption-check: require-gowork-off", "$(GOALCLI) adoption-check --verify"},
	}
	sourceRepository := isAdoptionSourceRepository(root)
	if sourceRepository {
		delete(required, lockPath)
	}

	var gaps []string
	for path, needles := range required {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			gaps = append(gaps, "missing "+path)
			continue
		}
		text := string(content)
		for _, needle := range needles {
			if !strings.Contains(text, needle) {
				gaps = append(gaps, fmt.Sprintf("%s missing %s", path, needle))
			}
		}
	}
	appendProtectMainRulesetGaps(root, &gaps)

	details := []string{
		"governance lock present",
		"git hooks present",
		"adoption workflow present",
		"governance Makefile target present",
		"harness adoption gate present",
		"main ruleset blocks direct push",
		"main ruleset requires adoption-check/governance-check/release-check",
	}
	if sourceRepository {
		details[0] = "source repository governance pack present"
	}
	return details, gaps
}

func isAdoptionSourceRepository(root string) bool {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return false
	}
	return strings.Contains(string(content), "module github.com/ZoneCNH/resiliencx")
}

type protectMainRuleset struct {
	Name         string            `json:"name"`
	Target       string            `json:"target"`
	Enforcement  string            `json:"enforcement"`
	BypassActors []json.RawMessage `json:"bypass_actors"`
	Conditions   struct {
		RefName struct {
			Include []string `json:"include"`
		} `json:"ref_name"`
	} `json:"conditions"`
	Rules []protectMainRule `json:"rules"`
}

type protectMainRule struct {
	Type       string          `json:"type"`
	Parameters json.RawMessage `json:"parameters"`
}

type requiredStatusChecksParameters struct {
	RequiredStatusChecks []struct {
		Context string `json:"context"`
	} `json:"required_status_checks"`
}

func appendProtectMainRulesetGaps(root string, gaps *[]string) {
	const path = ".github/rulesets/protect-main.json"
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		*gaps = append(*gaps, "missing "+path)
		return
	}

	var ruleset protectMainRuleset
	if err := json.Unmarshal(content, &ruleset); err != nil {
		*gaps = append(*gaps, path+" is not valid JSON")
		return
	}
	if ruleset.Name != "protect-main" {
		*gaps = append(*gaps, path+" missing protect-main name")
	}
	if ruleset.Target != "branch" {
		*gaps = append(*gaps, path+" must target branch")
	}
	if ruleset.Enforcement != "active" {
		*gaps = append(*gaps, path+" enforcement must be active")
	}
	if len(ruleset.BypassActors) > 0 {
		*gaps = append(*gaps, path+" must not allow bypass actors")
	}
	if !containsString(ruleset.Conditions.RefName.Include, "~DEFAULT_BRANCH") {
		*gaps = append(*gaps, path+" must protect default branch")
	}

	ruleTypes := make(map[string]bool)
	statusChecks := make(map[string]bool)
	for _, rule := range ruleset.Rules {
		ruleTypes[rule.Type] = true
		if rule.Type != "required_status_checks" {
			continue
		}
		var params requiredStatusChecksParameters
		if err := json.Unmarshal(rule.Parameters, &params); err != nil {
			*gaps = append(*gaps, path+" required_status_checks parameters are invalid")
			continue
		}
		for _, check := range params.RequiredStatusChecks {
			statusChecks[check.Context] = true
		}
	}
	for _, ruleType := range []string{"pull_request", "required_status_checks", "non_fast_forward", "deletion"} {
		if !ruleTypes[ruleType] {
			*gaps = append(*gaps, path+" missing "+ruleType+" rule")
		}
	}
	for _, check := range []string{"adoption-check", "governance-check", "release-check"} {
		if !statusChecks[check] {
			*gaps = append(*gaps, path+" required checks missing "+check)
		}
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
