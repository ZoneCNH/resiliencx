package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func runEvidenceCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("evidence-check", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("evidence-check", err, stderr)
	}
	return runRegistryCheck("evidence-check", map[string][]string{
		".agent/done-assertion.yaml":               {"DONE with evidence", "commit", "gates"},
		".agent/evidence-artifact-policy.yaml":     {"redaction", "sha256", "release/manifest/latest.json"},
		".agent/harness.yaml":                      {"manifest", "checksum", "required_fields"},
		".agent/evidence-artifacts.yaml":           {"release_evidence", "execution_evidence", "schema:", "contracts/execution-evidence.schema.json"},
		"contracts/execution-evidence.schema.json": {"evidence_id", "stdout_sha256", "commit", "exit_code", "artifact_path"},
	}, stdout, stderr)
}

func runCLIContract(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("cli-contract", args, internalCommandFlagSpec{boolFlags: []string{"json", "explain"}, stringFlags: []string{"output"}}); err != nil {
		return invalidInternalArgsExit("cli-contract", err, stderr)
	}
	return runRegistryCheck("cli-contract", map[string][]string{
		"docs/standard/goalcli-cli-contract.md": commandRegistryRequiredCommands(),
		"contracts/goalcli-report.schema.json":  {"command", "status", "details", "gaps"},
		".agent/command-registry.yaml":          requiredCommandRegistryNeedles(),
	}, stdout, stderr)
}

func runIssueRegistry(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := validateInternalCommandArgs("issue-registry", args, internalCommandFlagSpec{boolFlags: []string{"json"}}); err != nil {
		return invalidInternalArgsExit("issue-registry", err, stderr)
	}
	var gaps []string
	appendIssueRegistryGaps(".agent/issue-registry.yaml", &gaps)
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
	return runRegistryCheck("command-registry", map[string][]string{
		".agent/command-registry.yaml": requiredCommandRegistryNeedles(),
	}, stdout, stderr)
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
