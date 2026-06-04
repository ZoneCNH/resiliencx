package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

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
		".agent/command-registry.yaml":          requiredCommandRegistryNeedles(),
		".agent/makefile-target-registry.yaml":  contextTargets,
		".agent/makefile-baseline.yaml":         contextTargets,
		"docs/standard/goalcli-cli-contract.md": commandRegistryRequiredCommands(),
		"Makefile":                              {"release-final-check:", "$(MAKE) context-release"},
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
	appendIssueRegistryGaps(".agent/issue-registry.yaml", &gaps)
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

func validContextProfileName(profile string) bool {
	switch profile {
	case "lite", "standard", "full", "release":
		return true
	default:
		return false
	}
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
