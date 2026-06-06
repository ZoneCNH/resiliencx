package validation

import (
	"fmt"
	"reflect"
	"strings"
)

func RequireNonEmpty(field string, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

type runtimeFileOwner struct {
	path           string
	owner          string
	reviewRequired string
	reviewRule     string
	rationale      string
}

type executionContext struct {
	name   string
	fields map[string]string
}

var allowedRuntimeOwners = map[string]bool{
	"ci":           true,
	"gate-runtime": true,
	"governance":   true,
	"leader":       true,
	"release":      true,
	"security":     true,
	"standard":     true,
	"testing":      true,
}

// ValidateRuntimeFileOwnership checks the local .agent file-ownership index for
// the control-plane classifications goalcli depends on before allowing the
// runtime-file-ownership gate to pass.
func ValidateRuntimeFileOwnership(path string, content string) []string {
	var gaps []string
	if strings.TrimSpace(content) == "" {
		return []string{path + " must not be empty"}
	}
	if !containsYAMLKey(content, "schema_version") {
		gaps = append(gaps, path+" missing schema_version")
	}
	if !containsYAMLKey(content, "owners") {
		gaps = append(gaps, path+" missing owners")
	}

	owners := parseRuntimeFileOwners(content)
	if len(owners) == 0 {
		gaps = append(gaps, path+" owners must include at least one path classification")
	}

	seen := map[string]bool{}
	byPath := map[string]runtimeFileOwner{}
	for _, owner := range owners {
		if seen[owner.path] {
			gaps = append(gaps, path+" duplicate owner entry "+owner.path)
			continue
		}
		seen[owner.path] = true
		byPath[owner.path] = owner
		if owner.owner == "" {
			gaps = append(gaps, path+" "+owner.path+" missing owner")
		} else if !allowedRuntimeOwners[owner.owner] {
			gaps = append(gaps, path+" "+owner.path+" unknown owner "+owner.owner)
		}
		if strings.HasPrefix(owner.path, "/") {
			gaps = append(gaps, path+" "+owner.path+" must be repository-relative")
		}
		if owner.reviewRequired == "" {
			gaps = append(gaps, path+" "+owner.path+" missing review_required")
		} else if owner.reviewRequired != "true" && owner.reviewRequired != "false" {
			gaps = append(gaps, path+" "+owner.path+" review_required must be true or false")
		} else if owner.reviewRequired == "true" && owner.reviewRule == "" {
			gaps = append(gaps, path+" "+owner.path+" missing review_rule")
		}
		if owner.rationale == "" {
			gaps = append(gaps, path+" "+owner.path+" missing rationale")
		}
	}

	requireRuntimeOwner(path, byPath, ".agent/", "governance", true, &gaps)
	requireRuntimeOwner(path, byPath, "cmd/goalcli/", "gate-runtime", true, &gaps)
	requireRuntimeOwner(path, byPath, "contracts/", "standard", true, &gaps)
	return gaps
}

// ValidateExecutionContext checks the repository context manifest for the
// semantic context names and safety attributes goalcli accepts at runtime.
func ValidateExecutionContext(path string, content string, expected []string) []string {
	var gaps []string
	if strings.TrimSpace(content) == "" {
		return []string{path + " must not be empty"}
	}
	if !containsYAMLKey(content, "schema_version") {
		gaps = append(gaps, path+" missing schema_version")
	}
	if !containsYAMLKey(content, "contexts") {
		gaps = append(gaps, path+" missing contexts")
	}

	expectedSet := map[string]bool{}
	for _, context := range expected {
		expectedSet[context] = true
	}
	contexts := parseExecutionContexts(content)
	if len(contexts) == 0 {
		gaps = append(gaps, path+" contexts must include at least one execution context")
	}

	seen := map[string]bool{}
	byName := map[string]executionContext{}
	for _, context := range contexts {
		if context.name == "" {
			continue
		}
		if seen[context.name] {
			gaps = append(gaps, path+" duplicate context "+context.name)
			continue
		}
		seen[context.name] = true
		byName[context.name] = context
		if !expectedSet[context.name] {
			gaps = append(gaps, path+" unknown context "+context.name)
		}
		requireContextField(path, context, "write_scope", &gaps)
		requireBoolContextField(path, context, "mutates_files", &gaps)
		requireBoolContextField(path, context, "release_evidence", &gaps)
		requireContextField(path, context, "requires_gowork", &gaps)
		for field, value := range context.fields {
			if contextFieldMustBeRelative(field) && strings.HasPrefix(value, "/") {
				gaps = append(gaps, path+" "+context.name+" "+field+" must be repository-relative")
			}
		}
	}
	for _, context := range expected {
		if !seen[context] {
			gaps = append(gaps, path+" missing context "+context)
		}
	}

	localWrite, hasLocalWrite := byName["local_write"]
	releaseVerify, hasReleaseVerify := byName["release_verify"]
	if hasLocalWrite {
		requireContextValue(path, localWrite, "mutates_files", "true", &gaps)
		requireContextValue(path, localWrite, "release_evidence", "false", &gaps)
	}
	if hasReleaseVerify {
		requireContextValue(path, releaseVerify, "mutates_files", "false", &gaps)
		requireContextValue(path, releaseVerify, "release_evidence", "true", &gaps)
		requireContextValue(path, releaseVerify, "requires_gowork", "off", &gaps)
	}
	if hasLocalWrite && hasReleaseVerify && reflect.DeepEqual(localWrite.fields, releaseVerify.fields) {
		gaps = append(gaps, path+" local_write and release_verify must have distinct semantics")
	}
	return gaps
}

func requireRuntimeOwner(path string, owners map[string]runtimeFileOwner, ownerPath string, ownerName string, reviewRequired bool, gaps *[]string) {
	owner, ok := owners[ownerPath]
	if !ok {
		*gaps = append(*gaps, path+" owners must include "+ownerPath)
		return
	}
	if owner.owner != ownerName {
		*gaps = append(*gaps, path+" "+ownerPath+" owner must be "+ownerName)
	}
	wantReviewRequired := fmt.Sprintf("%t", reviewRequired)
	if owner.reviewRequired != wantReviewRequired {
		*gaps = append(*gaps, path+" "+ownerPath+" review_required must be "+wantReviewRequired)
	}
}

func containsYAMLKey(content string, key string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(stripInlineYAMLComment(line))
		if trimmed == key+":" || strings.HasPrefix(trimmed, key+": ") {
			return true
		}
	}
	return false
}

func parseRuntimeFileOwners(content string) []runtimeFileOwner {
	var owners []runtimeFileOwner
	var current *runtimeFileOwner
	inOwners := false

	flush := func() {
		if current != nil {
			owners = append(owners, *current)
			current = nil
		}
	}

	for _, rawLine := range strings.Split(content, "\n") {
		line := stripInlineYAMLComment(rawLine)
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if !inOwners {
			if trimmed == "owners:" {
				inOwners = true
			}
			continue
		}
		if indent == 0 {
			flush()
			break
		}
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			flush()
			current = &runtimeFileOwner{path: unquoteYAMLScalar(strings.TrimSuffix(trimmed, ":"))}
			continue
		}
		if current == nil || indent < 4 {
			continue
		}
		field, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		value = unquoteYAMLScalar(strings.TrimSpace(value))
		switch strings.TrimSpace(field) {
		case "owner":
			current.owner = value
		case "review_required":
			current.reviewRequired = value
		case "review_rule":
			current.reviewRule = value
		case "rationale":
			current.rationale = value
		}
	}
	flush()
	return owners
}

func parseExecutionContexts(content string) []executionContext {
	var contexts []executionContext
	var current *executionContext
	inContexts := false

	flush := func() {
		if current != nil {
			contexts = append(contexts, *current)
			current = nil
		}
	}

	for _, rawLine := range strings.Split(content, "\n") {
		line := stripInlineYAMLComment(rawLine)
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if !inContexts {
			if trimmed == "contexts:" {
				inContexts = true
			}
			continue
		}
		if indent == 0 {
			flush()
			break
		}
		if indent == 2 && strings.HasPrefix(trimmed, "- ") {
			flush()
			current = &executionContext{name: unquoteYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))), fields: map[string]string{}}
			continue
		}
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			flush()
			current = &executionContext{name: unquoteYAMLScalar(strings.TrimSuffix(trimmed, ":")), fields: map[string]string{}}
			continue
		}
		if current == nil || indent < 4 {
			continue
		}
		field, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		current.fields[strings.TrimSpace(field)] = unquoteYAMLScalar(strings.TrimSpace(value))
	}
	flush()
	return contexts
}

func requireContextField(path string, context executionContext, field string, gaps *[]string) {
	if context.fields[field] == "" {
		*gaps = append(*gaps, path+" "+context.name+" missing "+field)
	}
}

func requireBoolContextField(path string, context executionContext, field string, gaps *[]string) {
	value := context.fields[field]
	if value == "" {
		*gaps = append(*gaps, path+" "+context.name+" missing "+field)
		return
	}
	if value != "true" && value != "false" {
		*gaps = append(*gaps, path+" "+context.name+" "+field+" must be true or false")
	}
}

func requireContextValue(path string, context executionContext, field string, want string, gaps *[]string) {
	if got := context.fields[field]; got != "" && got != want {
		*gaps = append(*gaps, path+" "+context.name+" "+field+" must be "+want)
	}
}

func contextFieldMustBeRelative(field string) bool {
	return strings.Contains(field, "path") || strings.Contains(field, "root") || strings.Contains(field, "manifest")
}

func stripInlineYAMLComment(line string) string {
	if before, _, ok := strings.Cut(line, "#"); ok {
		return before
	}
	return line
}

func unquoteYAMLScalar(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
}
