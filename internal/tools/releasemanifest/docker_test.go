// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDockerEvidenceReturnsDefaultsWhenEnvUnset(t *testing.T) {
	for _, key := range []string{
		"DOCKER_TOOLCHAIN_ENABLED",
		"DOCKER_CONTRACT_VERSION",
		"DOCKER_GO_VERSION",
		"DOCKER_GOLANGCI_LINT_VERSION",
		"DOCKER_GOVULNCHECK_VERSION",
		"DOCKER_BUILDKIT_REQUIRED",
		"DOCKER_CACHE_MOUNTS",
		"DOCKER_BASE_IMAGE",
		"DOCKER_BASE_IMAGE_DIGEST",
		"DOCKER_TOOLCHAIN_IMAGE",
		"DOCKER_TOOLCHAIN_IMAGE_DIGEST",
		"DOCKER_RUNTIME_IMAGE",
		"DOCKER_RUNTIME_IMAGE_DIGEST",
		"DOCKER_VALIDATED_BY",
		"DOCKER_ARTIFACT_NAME",
		"DOCKER_ARTIFACT_URL",
		"WORKFLOW_RUN_ID",
		"ARTIFACT_NAME",
		"ARTIFACT_URL",
	} {
		t.Setenv(key, "")
	}

	got := buildDockerEvidence()

	if !got.Enabled {
		t.Fatal("docker.enabled = false, want default true")
	}
	if got.ContractVersion != "docker-toolchain/v2" {
		t.Fatalf("docker.contract_version = %q, want docker-toolchain/v2", got.ContractVersion)
	}
	if got.GoVersion != "1.23" {
		t.Fatalf("docker.go_version = %q, want 1.23", got.GoVersion)
	}
	if got.GolangCILintVersion != "golangci-lint v2.1.6" {
		t.Fatalf("docker.golangci_lint_version = %q, want golangci-lint v2.1.6", got.GolangCILintVersion)
	}
	if got.GovulncheckVersion != "govulncheck v1.1.4" {
		t.Fatalf("docker.govulncheck_version = %q, want govulncheck v1.1.4", got.GovulncheckVersion)
	}
	if !got.BuildKitRequired {
		t.Fatal("docker.buildkit_required = false, want default true")
	}
	if got.BaseImage != "golang:1.23-bookworm" {
		t.Fatalf("docker.base_image = %q, want golang:1.23-bookworm", got.BaseImage)
	}
	if got.BaseImageDigest != PlaceholderImageDigest {
		t.Fatalf("docker.base_image_digest = %q, want placeholder", got.BaseImageDigest)
	}
	if got.ToolchainImage != "resiliencx-toolchain:local" {
		t.Fatalf("docker.toolchain_image = %q, want resiliencx-toolchain:local", got.ToolchainImage)
	}
	if got.ToolchainImageDigest != PlaceholderImageDigest {
		t.Fatalf("docker.toolchain_image_digest = %q, want placeholder", got.ToolchainImageDigest)
	}
	if got.RuntimeImage != "resiliencx-goalcli-runtime:local" {
		t.Fatalf("docker.runtime_image = %q, want resiliencx-goalcli-runtime:local", got.RuntimeImage)
	}
	if got.RuntimeImageDigest != PlaceholderImageDigest {
		t.Fatalf("docker.runtime_image_digest = %q, want placeholder", got.RuntimeImageDigest)
	}
	for _, mount := range []string{"go-build", "go-mod", "golangci-lint"} {
		if !contains(got.CacheMounts, mount) {
			t.Fatalf("docker.cache_mounts = %v, want %s", got.CacheMounts, mount)
		}
	}
	for _, validator := range dockerEvidenceValidators {
		if !contains(got.ValidatedBy, validator) {
			t.Fatalf("docker.validated_by = %v, want %s", got.ValidatedBy, validator)
		}
	}
	if got.WorkflowRunID == "" {
		t.Fatal("docker.workflow_run_id is empty")
	}
	if got.ArtifactName == "" {
		t.Fatal("docker.artifact_name is empty")
	}
}

func TestBuildDockerEvidenceUsesToolchainImageFallbackChain(t *testing.T) {
	t.Setenv("DOCKER_TOOLCHAIN_IMAGE", "")
	t.Setenv("DOCKER_IMAGE", "custom-image:from-docker-image")
	t.Setenv("DOCKER_TOOLCHAIN_ENABLED", "true")
	t.Setenv("DOCKER_CONTRACT_VERSION", "docker-toolchain/v2")
	t.Setenv("DOCKER_GO_VERSION", "1.23")
	t.Setenv("DOCKER_GOLANGCI_LINT_VERSION", "test-lint")
	t.Setenv("DOCKER_GOVULNCHECK_VERSION", "test-govulncheck")
	t.Setenv("DOCKER_BUILDKIT_REQUIRED", "true")
	t.Setenv("DOCKER_BASE_IMAGE", "golang:1.23")
	t.Setenv("DOCKER_BASE_IMAGE_DIGEST", "sha256:base")
	t.Setenv("DOCKER_TOOLCHAIN_IMAGE_DIGEST", "sha256:toolchain")
	t.Setenv("DOCKER_RUNTIME_IMAGE", "runtime:latest")
	t.Setenv("DOCKER_RUNTIME_IMAGE_DIGEST", "sha256:runtime")

	got := buildDockerEvidence()

	if got.ToolchainImage != "custom-image:from-docker-image" {
		t.Fatalf("docker.toolchain_image = %q, want fallback to DOCKER_IMAGE value", got.ToolchainImage)
	}
}

func TestBuildDockerEvidenceDisabled(t *testing.T) {
	t.Setenv("DOCKER_TOOLCHAIN_ENABLED", "false")

	got := buildDockerEvidence()

	if got.Enabled {
		t.Fatal("docker.enabled = true, want false when DOCKER_TOOLCHAIN_ENABLED=false")
	}
}

func TestBuildDockerEvidenceWithCustomDigests(t *testing.T) {
	t.Setenv("DOCKER_BASE_IMAGE_DIGEST", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("DOCKER_TOOLCHAIN_IMAGE_DIGEST", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	t.Setenv("DOCKER_RUNTIME_IMAGE_DIGEST", "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")

	got := buildDockerEvidence()

	if !strings.HasPrefix(got.BaseImageDigest, "sha256:") {
		t.Fatalf("base_image_digest = %q, want sha256 prefix", got.BaseImageDigest)
	}
	if !strings.HasPrefix(got.ToolchainImageDigest, "sha256:") {
		t.Fatalf("toolchain_image_digest = %q, want sha256 prefix", got.ToolchainImageDigest)
	}
	if !strings.HasPrefix(got.RuntimeImageDigest, "sha256:") {
		t.Fatalf("runtime_image_digest = %q, want sha256 prefix", got.RuntimeImageDigest)
	}
	if got.BaseImageDigest == got.ToolchainImageDigest {
		t.Fatal("base and toolchain digests should differ")
	}
	if got.ToolchainImageDigest == got.RuntimeImageDigest {
		t.Fatal("toolchain and runtime digests should differ")
	}
}

func TestBuildGovernanceRuntimeRecordsExpectedStructure(t *testing.T) {
	got := buildGovernanceRuntime()

	if got.Runtime != "context-runtime-v4.0" {
		t.Fatalf("governance_runtime.runtime = %q, want context-runtime-v4.0", got.Runtime)
	}
	if got.Status != "present" {
		t.Fatalf("governance_runtime.status = %q, want present", got.Status)
	}
	if got.SchemaVersion != governanceRuntimeVersion {
		t.Fatalf("governance_runtime.schema_version = %q, want %q", got.SchemaVersion, governanceRuntimeVersion)
	}
	if got.RuntimeVersion != governanceRuntimeVersion {
		t.Fatalf("governance_runtime.runtime_version = %q, want %q", got.RuntimeVersion, governanceRuntimeVersion)
	}
	if got.ProfileCheck != "context-profile-check" {
		t.Fatalf("governance_runtime.profile_check = %q, want context-profile-check", got.ProfileCheck)
	}
	if got.ReleaseTarget != "context-release" {
		t.Fatalf("governance_runtime.release_target = %q, want context-release", got.ReleaseTarget)
	}

	wantProfiles := []string{"context-lite", "context-standard", "context-full", "context-release"}
	if len(got.Profiles) != len(wantProfiles) {
		t.Fatalf("governance_runtime.profiles = %v, want %v", got.Profiles, wantProfiles)
	}
	for _, p := range wantProfiles {
		if !contains(got.Profiles, p) {
			t.Fatalf("governance_runtime.profiles = %v, want %s", got.Profiles, p)
		}
	}

	wantAliases := []string{"context-fast-check", "context-standard-check", "context-full-check"}
	if len(got.LegacyAliases) != len(wantAliases) {
		t.Fatalf("governance_runtime.legacy_aliases = %v, want %v", got.LegacyAliases, wantAliases)
	}
	for _, alias := range wantAliases {
		if !contains(got.LegacyAliases, alias) {
			t.Fatalf("governance_runtime.legacy_aliases = %v, want %s", got.LegacyAliases, alias)
		}
	}

	assertGovernanceRuntimeEvidence(t, got)
}

func TestBuildGovernanceRuntimeReturnsCopiedStatusMaps(t *testing.T) {
	got1 := buildGovernanceRuntime()
	got2 := buildGovernanceRuntime()

	got1.GateStatuses["test"] = "mutated"
	if _, exists := got2.GateStatuses["test"]; exists {
		t.Fatal("governance runtime gate_statuses are shared between calls, want deep copy")
	}
}

func TestBuildStandardImpactEvidenceReturnsDefaultWhenReportMissing(t *testing.T) {
	chdir(t, t.TempDir())

	got, err := buildStandardImpactEvidence()
	if err != nil {
		t.Fatal(err)
	}

	if got.ReportPath != standardImpactReportPath {
		t.Fatalf("report_path = %q, want %q", got.ReportPath, standardImpactReportPath)
	}
	if got.Status != "missing" {
		t.Fatalf("status = %q, want missing", got.Status)
	}
	if got.DownstreamReleaseDecision != "not_required" {
		t.Fatalf("downstream_release_decision = %q, want not_required", got.DownstreamReleaseDecision)
	}
	if got.RepositoryRulesReleaseDecision != "not_required" {
		t.Fatalf("repository_rules_release_decision = %q, want not_required", got.RepositoryRulesReleaseDecision)
	}
	if got.ReportSHA256 != "" {
		t.Fatalf("report_sha256 = %q, want empty", got.ReportSHA256)
	}
}

func TestBuildStandardImpactEvidenceParsesAllFields(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, filepath.FromSlash(standardImpactReportPath))
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Join([]string{
		"# Standard Impact Report",
		"",
		"- downstream_sync_required: `true`",
		"- context_runtime_change: `false`",
		"- governance_registry_change: `true`",
		"- downstream_release_decision: `required`",
		"- repository_rules_release_decision: `audit_required`",
		"- primary_downstream: `github.com/ZoneCNH/kernel`",
	}, "\n")
	if err := os.WriteFile(reportPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, root)

	got, err := buildStandardImpactEvidence()
	if err != nil {
		t.Fatal(err)
	}

	if got.Status != "present" {
		t.Fatalf("status = %q, want present", got.Status)
	}
	if !strings.HasPrefix(got.ReportSHA256, "sha256:") {
		t.Fatalf("report_sha256 = %q, want sha256 prefix", got.ReportSHA256)
	}
	if !got.DownstreamSyncRequired {
		t.Fatal("downstream_sync_required = false, want true")
	}
	if got.ContextRuntimeChange {
		t.Fatal("context_runtime_change = true, want false")
	}
	if !got.GovernanceRegistryChange {
		t.Fatal("governance_registry_change = false, want true")
	}
	if got.DownstreamReleaseDecision != "required" {
		t.Fatalf("downstream_release_decision = %q, want required", got.DownstreamReleaseDecision)
	}
	if got.RepositoryRulesReleaseDecision != "audit_required" {
		t.Fatalf("repository_rules_release_decision = %q, want audit_required", got.RepositoryRulesReleaseDecision)
	}
	if got.PrimaryDownstream != "github.com/ZoneCNH/kernel" {
		t.Fatalf("primary_downstream = %q, want github.com/ZoneCNH/kernel", got.PrimaryDownstream)
	}
}

func TestBuildDebtEvidenceReturnsDefaultsWhenReportMissing(t *testing.T) {
	chdir(t, t.TempDir())

	got, err := buildDebtEvidence()
	if err != nil {
		t.Fatal(err)
	}

	if got.ReportPath != debtReportPath {
		t.Fatalf("report_path = %q, want %q", got.ReportPath, debtReportPath)
	}
	if got.MarkdownPath != debtMarkdownPath {
		t.Fatalf("markdown_path = %q, want %q", got.MarkdownPath, debtMarkdownPath)
	}
	if got.ChecksumPath != debtChecksumPath {
		t.Fatalf("checksum_path = %q, want %q", got.ChecksumPath, debtChecksumPath)
	}
	if got.Status != "missing" {
		t.Fatalf("status = %q, want missing", got.Status)
	}
	if got.Score != 0 {
		t.Fatalf("score = %f, want 0", got.Score)
	}
	if got.MinScore != 9.8 {
		t.Fatalf("min_score = %f, want 9.8", got.MinScore)
	}
	if got.CheckCount != 0 {
		t.Fatalf("check_count = %d, want 0", got.CheckCount)
	}
	if got.ReportSHA256 != "" {
		t.Fatalf("report_sha256 = %q, want empty", got.ReportSHA256)
	}
}

func TestBuildDebtEvidenceParsesReport(t *testing.T) {
	root := t.TempDir()
	writeDebtReportFixture(t, root)
	chdir(t, root)

	got, err := buildDebtEvidence()
	if err != nil {
		t.Fatal(err)
	}

	if got.Status != "passed" {
		t.Fatalf("status = %q, want passed", got.Status)
	}
	if got.Score != 9.8 {
		t.Fatalf("score = %f, want 9.8", got.Score)
	}
	if got.MinScore != 9.8 {
		t.Fatalf("min_score = %f, want 9.8", got.MinScore)
	}
	if got.CheckCount != 1 {
		t.Fatalf("check_count = %d, want 1", got.CheckCount)
	}
	if !strings.HasPrefix(got.ReportSHA256, "sha256:") {
		t.Fatalf("report_sha256 = %q, want sha256 prefix", got.ReportSHA256)
	}
}

func TestBuildDebtEvidenceReportsDirectoryError(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, filepath.FromSlash(debtReportPath))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, root)

	if _, err := buildDebtEvidence(); err == nil {
		t.Fatal("buildDebtEvidence succeeded for directory report, want error")
	}
}

func TestBuildGeneratorEvidenceReturnsExpectedStructure(t *testing.T) {
	got := buildGeneratorEvidence()

	if got.Command != "GOWORK=off make integration" {
		t.Fatalf("command = %q, want GOWORK=off make integration", got.Command)
	}
	if !got.Required {
		t.Fatal("required = false, want true")
	}
	if len(got.Targets) == 0 {
		t.Fatal("targets is empty, want at least one target")
	}

	for _, want := range generatorEvidenceTargets {
		if !hasGeneratorTarget(got.Targets, want.Name, want.ModulePath, want.PackageName) {
			t.Fatalf("targets = %+v, want %s target", got.Targets, want.Name)
		}
	}
}

func TestBuildGeneratorEvidenceTargetsAreIndependent(t *testing.T) {
	got1 := buildGeneratorEvidence()
	got2 := buildGeneratorEvidence()

	got1.Targets[0].Name = "mutated"
	if got2.Targets[0].Name == "mutated" {
		t.Fatal("generator targets are shared between calls, want independent copy")
	}
}

func TestCopyStatusMapReturnsIndependentCopy(t *testing.T) {
	original := map[string]string{"key1": "passed", "key2": "failed"}
	copied := copyStatusMap(original)

	copied["key1"] = "mutated"
	copied["key3"] = "added"

	if original["key1"] != "passed" {
		t.Fatalf("original key1 = %q, want passed (original was mutated)", original["key1"])
	}
	if _, exists := original["key3"]; exists {
		t.Fatal("original has key3, want independent copy")
	}
}

func TestCopyStatusMapHandlesEmptyMap(t *testing.T) {
	copied := copyStatusMap(map[string]string{})
	if len(copied) != 0 {
		t.Fatalf("copied = %v, want empty", copied)
	}
}

func TestParseReportValueTrimsWhitespaceAndBackticks(t *testing.T) {
	report := "- key: `  trimmed value  `\n"
	got := parseReportValue(report, "key")
	if got != "trimmed value" {
		t.Fatalf("parseReportValue = %q, want 'trimmed value'", got)
	}
}

func TestReportValueDefaultReturnsFallbackForMissing(t *testing.T) {
	got := reportValueDefault("- other: `value`\n", "missing", "fallback")
	if got != "fallback" {
		t.Fatalf("reportValueDefault = %q, want fallback", got)
	}
}

func TestReportValueDefaultReturnsParsedWhenPresent(t *testing.T) {
	got := reportValueDefault("- target: `found`\n", "target", "fallback")
	if got != "found" {
		t.Fatalf("reportValueDefault = %q, want found", got)
	}
}
