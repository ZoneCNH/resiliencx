package validation

import (
	"strings"
	"testing"
)

func TestRequireNonEmptyRejectsEmptyValue(t *testing.T) {
	if err := RequireNonEmpty("name", ""); err == nil {
		t.Fatal("expected empty value to fail")
	}
}

func TestRequireNonEmptyAcceptsValue(t *testing.T) {
	if err := RequireNonEmpty("name", "resiliencx"); err != nil {
		t.Fatalf("expected value to pass: %v", err)
	}
}

func TestValidateRuntimeFileOwnershipAcceptsControlPlaneIndex(t *testing.T) {
	gaps := ValidateRuntimeFileOwnership(".agent/policies/runtime-file-ownership.yaml", runtimeFileOwnershipFixture())
	if len(gaps) != 0 {
		t.Fatalf("gaps = %#v; want none", gaps)
	}
}

func TestValidateRuntimeFileOwnershipRejectsMissingControlPlaneClassification(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  "cmd/goalcli/":
    owner: gate-runtime
    review_required: true
    rationale: CLI validators.
  "contracts/":
    owner: standard
    review_required: true
    rationale: Schema contracts.
`
	gaps := ValidateRuntimeFileOwnership(".agent/policies/runtime-file-ownership.yaml", fixture)
	if !validationGapsContain(gaps, ".agent/policies/runtime-file-ownership.yaml owners must include .agent/") {
		t.Fatalf("gaps = %#v; want .agent/ classification gap", gaps)
	}
}

func TestValidateRuntimeFileOwnershipRejectsInvalidReviewRequired(t *testing.T) {
	fixture := runtimeFileOwnershipFixture()
	fixture = strings.Replace(fixture, "review_required: true", "review_required: maybe", 1)

	gaps := ValidateRuntimeFileOwnership(".agent/policies/runtime-file-ownership.yaml", fixture)
	if !validationGapsContain(gaps, ".agent/policies/runtime-file-ownership.yaml .agent/ review_required must be true or false") {
		t.Fatalf("gaps = %#v; want boolean review_required gap", gaps)
	}
}

func TestValidateRuntimeFileOwnershipRejectsDuplicateEntries(t *testing.T) {
	fixture := runtimeFileOwnershipFixture() + `  ".agent/":
    owner: governance
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Duplicate control plane.
`
	gaps := ValidateRuntimeFileOwnership(".agent/policies/runtime-file-ownership.yaml", fixture)
	if !validationGapsContain(gaps, ".agent/policies/runtime-file-ownership.yaml duplicate owner entry .agent/") {
		t.Fatalf("gaps = %#v; want duplicate owner gap", gaps)
	}
}

func TestValidateRuntimeFileOwnershipRejectsUnknownOwner(t *testing.T) {
	fixture := strings.Replace(runtimeFileOwnershipFixture(), "owner: governance", "owner: mystery-owner", 1)

	gaps := ValidateRuntimeFileOwnership(".agent/policies/runtime-file-ownership.yaml", fixture)
	if !validationGapsContain(gaps, ".agent/policies/runtime-file-ownership.yaml .agent/ unknown owner mystery-owner") {
		t.Fatalf("gaps = %#v; want unknown owner gap", gaps)
	}
}

func TestValidateRuntimeFileOwnershipRequiresReviewRule(t *testing.T) {
	fixture := strings.Replace(runtimeFileOwnershipFixture(), "    review_rule: RULE-CHANGE\n", "", 1)

	gaps := ValidateRuntimeFileOwnership(".agent/policies/runtime-file-ownership.yaml", fixture)
	if !validationGapsContain(gaps, ".agent/policies/runtime-file-ownership.yaml .agent/ missing review_rule") {
		t.Fatalf("gaps = %#v; want missing review_rule gap", gaps)
	}
}

func TestValidateRuntimeFileOwnershipRejectsAbsoluteOwnerPath(t *testing.T) {
	fixture := strings.Replace(runtimeFileOwnershipFixture(), "\".agent/\":", "\"/tmp/.agent/\":", 1)

	gaps := ValidateRuntimeFileOwnership(".agent/policies/runtime-file-ownership.yaml", fixture)
	if !validationGapsContain(gaps, ".agent/policies/runtime-file-ownership.yaml /tmp/.agent/ must be repository-relative") {
		t.Fatalf("gaps = %#v; want repository-relative path gap", gaps)
	}
}

func TestValidateExecutionContextAcceptsSemanticManifest(t *testing.T) {
	gaps := ValidateExecutionContext(".agent/policies/execution-context.yaml", executionContextFixture(), executionContextsFixture())
	if len(gaps) != 0 {
		t.Fatalf("gaps = %#v; want none", gaps)
	}
}

func TestValidateExecutionContextRejectsUnknownContext(t *testing.T) {
	fixture := strings.Replace(executionContextFixture(), "release_verify:", "release_magic:", 1)

	gaps := ValidateExecutionContext(".agent/policies/execution-context.yaml", fixture, executionContextsFixture())
	if !validationGapsContain(gaps, ".agent/policies/execution-context.yaml unknown context release_magic") {
		t.Fatalf("gaps = %#v; want unknown context gap", gaps)
	}
}

func TestValidateExecutionContextRequiresDistinctLocalWriteAndReleaseVerify(t *testing.T) {
	fixture := strings.Replace(executionContextFixture(), "mutates_files: false\n    release_evidence: true", "mutates_files: true\n    release_evidence: false", 1)

	gaps := ValidateExecutionContext(".agent/policies/execution-context.yaml", fixture, executionContextsFixture())
	if !validationGapsContain(gaps, ".agent/policies/execution-context.yaml release_verify mutates_files must be false") ||
		!validationGapsContain(gaps, ".agent/policies/execution-context.yaml release_verify release_evidence must be true") {
		t.Fatalf("gaps = %#v; want release_verify semantic gaps", gaps)
	}
}

func TestValidateRuntimeFileOwnership_EmptyContent(t *testing.T) {
	gaps := ValidateRuntimeFileOwnership("test.yaml", "")
	if !validationGapsContain(gaps, "test.yaml must not be empty") {
		t.Fatalf("gaps = %#v; want empty content gap", gaps)
	}
}

func TestValidateRuntimeFileOwnership_MissingSchemaVersion(t *testing.T) {
	fixture := `owners:
  ".agent/":
    owner: governance
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Control plane.
`
	gaps := ValidateRuntimeFileOwnership("test.yaml", fixture)
	if !validationGapsContain(gaps, "test.yaml missing schema_version") {
		t.Fatalf("gaps = %#v; want missing schema_version", gaps)
	}
}

func TestValidateRuntimeFileOwnership_MissingOwners(t *testing.T) {
	fixture := `schema_version: "2.9.3"
`
	gaps := ValidateRuntimeFileOwnership("test.yaml", fixture)
	if !validationGapsContain(gaps, "test.yaml missing owners") {
		t.Fatalf("gaps = %#v; want missing owners", gaps)
	}
}

func TestValidateRuntimeFileOwnership_MissingOwner(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/":
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Control plane.
`
	gaps := ValidateRuntimeFileOwnership("test.yaml", fixture)
	if !validationGapsContain(gaps, "test.yaml .agent/ missing owner") {
		t.Fatalf("gaps = %#v; want missing owner", gaps)
	}
}

func TestValidateRuntimeFileOwnership_MissingReviewRequired(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/":
    owner: governance
    review_rule: RULE-CHANGE
    rationale: Control plane.
`
	gaps := ValidateRuntimeFileOwnership("test.yaml", fixture)
	if !validationGapsContain(gaps, "test.yaml .agent/ missing review_required") {
		t.Fatalf("gaps = %#v; want missing review_required", gaps)
	}
}

func TestValidateRuntimeFileOwnership_FalseReviewRequired(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/":
    owner: governance
    review_required: false
    rationale: Control plane.
`
	gaps := ValidateRuntimeFileOwnership("test.yaml", fixture)
	// false review_required means no review_rule check
	for _, g := range gaps {
		if strings.Contains(g, "missing review_rule") {
			t.Fatalf("should not require review_rule when review_required is false: %v", gaps)
		}
	}
}

func TestValidateRuntimeFileOwnership_MissingRationale(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/":
    owner: governance
    review_required: true
    review_rule: RULE-CHANGE
`
	gaps := ValidateRuntimeFileOwnership("test.yaml", fixture)
	if !validationGapsContain(gaps, "test.yaml .agent/ missing rationale") {
		t.Fatalf("gaps = %#v; want missing rationale", gaps)
	}
}

func TestValidateExecutionContext_EmptyContent(t *testing.T) {
	gaps := ValidateExecutionContext("test.yaml", "", []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml must not be empty") {
		t.Fatalf("gaps = %#v; want empty content gap", gaps)
	}
}

func TestValidateExecutionContext_MissingSchemaVersion(t *testing.T) {
	fixture := `contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml missing schema_version") {
		t.Fatalf("gaps = %#v; want missing schema_version", gaps)
	}
}

func TestValidateExecutionContext_MissingContexts(t *testing.T) {
	fixture := `schema_version: "2.9.3"
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml missing contexts") {
		t.Fatalf("gaps = %#v; want missing contexts", gaps)
	}
}

func TestValidateExecutionContext_MissingExpectedContext(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write", "release_verify"})
	if !validationGapsContain(gaps, "test.yaml missing context release_verify") {
		t.Fatalf("gaps = %#v; want missing context", gaps)
	}
}

func TestValidateExecutionContext_MissingBoolField(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    release_evidence: false
    requires_gowork: off
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml local_write missing mutates_files") {
		t.Fatalf("gaps = %#v; want missing mutates_files", gaps)
	}
}

func TestValidateExecutionContext_InvalidBoolField(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: maybe
    release_evidence: false
    requires_gowork: off
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml local_write mutates_files must be true or false") {
		t.Fatalf("gaps = %#v; want invalid bool field", gaps)
	}
}

func TestValidateExecutionContext_MissingWriteScope(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml local_write missing write_scope") {
		t.Fatalf("gaps = %#v; want missing write_scope", gaps)
	}
}

func TestValidateExecutionContext_MissingRequiresGowork(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml local_write missing requires_gowork") {
		t.Fatalf("gaps = %#v; want missing requires_gowork", gaps)
	}
}

func TestValidateExecutionContext_DuplicateContext(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml duplicate context local_write") {
		t.Fatalf("gaps = %#v; want duplicate context", gaps)
	}
}

func TestValidateExecutionContext_ContextFieldMustBeRelative(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
    output_path: /tmp/output
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml local_write output_path must be repository-relative") {
		t.Fatalf("gaps = %#v; want repository-relative path gap", gaps)
	}
}

func TestValidateExecutionContext_LocalWriteSemanticCheck(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: false
    release_evidence: true
    requires_gowork: off
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write"})
	if !validationGapsContain(gaps, "test.yaml local_write mutates_files must be true") {
		t.Fatalf("gaps = %#v; want mutates_files must be true", gaps)
	}
	if !validationGapsContain(gaps, "test.yaml local_write release_evidence must be false") {
		t.Fatalf("gaps = %#v; want release_evidence must be false", gaps)
	}
}

func TestValidateExecutionContext_ReleaseVerifyGoworkCheck(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  release_verify:
    write_scope: release_read_only
    mutates_files: false
    release_evidence: true
    requires_gowork: on
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"release_verify"})
	if !validationGapsContain(gaps, "test.yaml release_verify requires_gowork must be off") {
		t.Fatalf("gaps = %#v; want requires_gowork must be off", gaps)
	}
}

func TestContainsYAMLKey_NoMatch(t *testing.T) {
	if containsYAMLKey("foo: bar\nbaz: qux", "missing") {
		t.Fatal("expected false for missing key")
	}
}

func TestStripInlineYAMLComment_NoHash(t *testing.T) {
	got := stripInlineYAMLComment("foo: bar")
	if got != "foo: bar" {
		t.Fatalf("expected 'foo: bar', got %q", got)
	}
}

func TestParseRuntimeFileOwners_EmptyOwners(t *testing.T) {
	owners := parseRuntimeFileOwners(`schema_version: "2.9.3"
`)
	if len(owners) != 0 {
		t.Fatalf("expected 0 owners, got %d", len(owners))
	}
}

func TestParseExecutionContexts_EmptyContexts(t *testing.T) {
	contexts := parseExecutionContexts(`schema_version: "2.9.3"
`)
	if len(contexts) != 0 {
		t.Fatalf("expected 0 contexts, got %d", len(contexts))
	}
}

func TestValidateRuntimeFileOwnership_WrongRequiredOwner(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/":
    owner: testing
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Control plane.
  "cmd/goalcli/":
    owner: gate-runtime
    review_required: true
    review_rule: GATE-RUNTIME-CHANGE
    rationale: Goalcli.
  "contracts/":
    owner: standard
    review_required: true
    review_rule: CONTRACT-CHANGE
    rationale: Public.
`
	gaps := ValidateRuntimeFileOwnership("test.yaml", fixture)
	if !validationGapsContain(gaps, "test.yaml .agent/ owner must be governance") {
		t.Fatalf("gaps = %#v; want wrong owner for .agent/", gaps)
	}
}

func TestParseRuntimeFileOwners_WithInlineComments(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/": # control plane
    owner: governance # who owns this
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Control plane manifests.
`
	owners := parseRuntimeFileOwners(fixture)
	if len(owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(owners))
	}
	if owners[0].owner != "governance" {
		t.Fatalf("expected governance, got %s", owners[0].owner)
	}
}

func TestParseRuntimeFileOwners_QuotedValues(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/":
    owner: "governance"
    review_required: "true"
    review_rule: "RULE-CHANGE"
    rationale: "Control plane."
`
	owners := parseRuntimeFileOwners(fixture)
	if len(owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(owners))
	}
	if owners[0].owner != "governance" {
		t.Fatalf("expected governance, got %s", owners[0].owner)
	}
}

func TestParseRuntimeFileOwners_TopLevelBreak(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/":
    owner: governance
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Control plane.
top_level_key: value
`
	owners := parseRuntimeFileOwners(fixture)
	if len(owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(owners))
	}
}

func TestParseExecutionContexts_QuotedName(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  "local_write":
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	contexts := parseExecutionContexts(fixture)
	if len(contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(contexts))
	}
	if contexts[0].name != "local_write" {
		t.Fatalf("expected local_write, got %s", contexts[0].name)
	}
}

func TestParseExecutionContexts_TopLevelBreak(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
top_level_key: value
`
	contexts := parseExecutionContexts(fixture)
	if len(contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(contexts))
	}
}

func TestParseExecutionContexts_DashStyle(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  - local_write
  - release_verify
`
	contexts := parseExecutionContexts(fixture)
	if len(contexts) != 2 {
		t.Fatalf("expected 2 contexts, got %d: %v", len(contexts), contexts)
	}
}

func TestStripInlineYAMLComment_WithHash(t *testing.T) {
	got := stripInlineYAMLComment("foo: bar # comment")
	if got != "foo: bar " {
		t.Fatalf("expected 'foo: bar ', got %q", got)
	}
}

func TestContextFieldMustBeRelative(t *testing.T) {
	cases := []struct {
		field string
		want  bool
	}{
		{"output_path", true},
		{"root_dir", true},
		{"manifest_file", true},
		{"write_scope", false},
		{"mutates_files", false},
	}
	for _, tc := range cases {
		if got := contextFieldMustBeRelative(tc.field); got != tc.want {
			t.Fatalf("contextFieldMustBeRelative(%q) = %v, want %v", tc.field, got, tc.want)
		}
	}
}

func TestParseRuntimeFileOwners_FieldWithoutColon(t *testing.T) {
	fixture := `schema_version: "2.9.3"
owners:
  ".agent/":
    owner: governance
    no_colon_line
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Control plane.
`
	owners := parseRuntimeFileOwners(fixture)
	if len(owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(owners))
	}
	if owners[0].owner != "governance" {
		t.Fatalf("expected governance, got %s", owners[0].owner)
	}
}

func TestParseRuntimeFileOwners_BeforeOwners(t *testing.T) {
	fixture := `schema_version: "2.9.3"
not_owners:
  foo: bar
owners:
  ".agent/":
    owner: governance
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Control plane.
`
	owners := parseRuntimeFileOwners(fixture)
	if len(owners) != 1 {
		t.Fatalf("expected 1 owner, got %d", len(owners))
	}
}

func TestParseExecutionContexts_EmptyNameContext(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  "":
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	contexts := parseExecutionContexts(fixture)
	// Empty name context should be skipped
	found := false
	for _, c := range contexts {
		if c.name == "local_write" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected local_write context, got %v", contexts)
	}
}

func TestParseExecutionContexts_FieldWithoutColon(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    no_colon_line
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	contexts := parseExecutionContexts(fixture)
	if len(contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(contexts))
	}
}

func TestParseExecutionContexts_BeforeContexts(t *testing.T) {
	fixture := `schema_version: "2.9.3"
not_contexts:
  foo: bar
contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	contexts := parseExecutionContexts(fixture)
	if len(contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(contexts))
	}
}

func TestValidateExecutionContext_IdenticalLocalWriteAndReleaseVerify(t *testing.T) {
	fixture := `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
  release_verify:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
`
	gaps := ValidateExecutionContext("test.yaml", fixture, []string{"local_write", "release_verify"})
	if !validationGapsContain(gaps, "test.yaml local_write and release_verify must have distinct semantics") {
		t.Fatalf("gaps = %#v; want distinct semantics gap", gaps)
	}
}

func runtimeFileOwnershipFixture() string {
	return `schema_version: "2.9.3"
owners:
  ".agent/":
    owner: governance
    review_required: true
    review_rule: RULE-CHANGE
    rationale: Control plane manifests.
  "cmd/goalcli/":
    owner: gate-runtime
    review_required: true
    review_rule: GATE-RUNTIME-CHANGE
    rationale: Goalcli validator surface.
  "contracts/":
    owner: standard
    review_required: true
    review_rule: CONTRACT-CHANGE
    rationale: Public contracts.
`
}

func executionContextFixture() string {
	return `schema_version: "2.9.3"
contexts:
  local_write:
    write_scope: worktree
    mutates_files: true
    release_evidence: false
    requires_gowork: off
  local_readonly:
    write_scope: read_only
    mutates_files: false
    release_evidence: false
    requires_gowork: off
  ci_pull_request:
    write_scope: read_only
    mutates_files: false
    release_evidence: false
    requires_gowork: off
  ci_main_verify:
    write_scope: read_only
    mutates_files: false
    release_evidence: false
    requires_gowork: off
  release_verify:
    write_scope: release_read_only
    mutates_files: false
    release_evidence: true
    requires_gowork: off
`
}

func executionContextsFixture() []string {
	return []string{"local_write", "local_readonly", "ci_pull_request", "ci_main_verify", "release_verify"}
}

func validationGapsContain(gaps []string, want string) bool {
	for _, gap := range gaps {
		if gap == want {
			return true
		}
	}
	return false
}
