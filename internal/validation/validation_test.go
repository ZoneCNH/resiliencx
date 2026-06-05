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
