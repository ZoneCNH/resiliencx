package main

import (
	"fmt"
	"io"
	"strings"
)

func runMakefileBaseline(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("makefile-baseline", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("makefile-baseline", err, stderr)
	}
	requiredTargets := append([]string{"fmt", "vet", "lint", "test", "race", "boundary", "security", "contracts", "docs-check", "rules-verify", "evidence", "score-check", "main-guard", "worktree-guard", "worktree-check", "context-check", "spec-check", "design-check", "task-check", "pr-check", "evidence-check", "cli-contract", "issue-registry", "command-registry", "makefile-baseline", "governance-check", "p1-governance-check", "execution-context", "p2-runtime-check", "release-check", "release-final-check"}, contextRuntimeTargets()...)
	requiredTargets = append(requiredTargets, goalcliMakefileTargets()...)
	required := map[string][]string{"Makefile": {}, ".agent/makefile-target-registry.yaml": requiredTargets, ".agent/makefile-baseline.yaml": requiredTargets}
	for _, target := range requiredTargets {
		required["Makefile"] = append(required["Makefile"], ".PHONY: "+target, target+":")
	}
	return runRegistryCheck("makefile-baseline", required, stdout, stderr)
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
