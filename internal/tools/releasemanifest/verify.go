// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/ZoneCNH/resiliencx/internal/releasequality"
)

// verifyManifest 验证已有的发布清单。
func verifyManifest(path string, requirePassed bool, requireClean bool, expectVersion string, minScore float64) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var got Manifest
	if err := json.Unmarshal(data, &got); err != nil {
		return err
	}

	current, err := buildManifest()
	if err != nil {
		return err
	}

	var failures []string
	requireNonEmpty(&failures, "module", got.Module)
	requireNonEmpty(&failures, "version", got.Version)
	requireNonEmpty(&failures, "commit", got.Commit)
	requireNonEmpty(&failures, "tree_sha", got.TreeSHA)
	requireNonEmpty(&failures, "source_digest", got.SourceDigest)
	requireNonEmpty(&failures, "go_version", got.GoVersion)
	requireNonEmpty(&failures, "generated_at", got.GeneratedAt)
	requireNonEmpty(&failures, "generated_by", got.GeneratedBy)
	requireNonEmpty(&failures, "tree_state", got.TreeState)
	requireNonEmpty(&failures, "workflow.workflow_run_id", got.Workflow.WorkflowRunID)
	requireNonEmpty(&failures, "workflow.artifact_name", got.Workflow.ArtifactName)
	requireNonEmpty(&failures, "workflow.artifact_url", got.Workflow.ArtifactURL)

	expectVersion = strings.TrimSpace(expectVersion)
	if _, err := time.Parse(time.RFC3339, got.GeneratedAt); err != nil {
		failures = append(failures, "generated_at must be RFC3339")
	}
	if expectVersion != "" && got.Version != expectVersion {
		failures = append(failures, fmt.Sprintf("version mismatch: got %q, want %q", got.Version, expectVersion))
	}
	if got.Module != current.Module {
		failures = append(failures, fmt.Sprintf("module mismatch: got %q, want %q", got.Module, current.Module))
	}
	if got.Commit != current.Commit {
		failures = append(failures, fmt.Sprintf("commit mismatch: got %q, want %q", got.Commit, current.Commit))
	}
	if got.TreeSHA != current.TreeSHA {
		failures = append(failures, fmt.Sprintf("tree_sha mismatch: got %q, want %q", got.TreeSHA, current.TreeSHA))
	}
	if got.SourceDigest != current.SourceDigest {
		failures = append(failures, "source_digest does not match current tracked file contents")
	}
	if got.TrackedFileCount != current.TrackedFileCount {
		failures = append(failures, fmt.Sprintf("tracked_file_count mismatch: got %d, want %d", got.TrackedFileCount, current.TrackedFileCount))
	}
	if got.TreeState != current.TreeState {
		failures = append(failures, fmt.Sprintf("tree_state mismatch: got %q, want %q", got.TreeState, current.TreeState))
	}
	failures = append(failures, validateDockerEvidence(got.Docker, requirePassed)...)
	if got.Score.Value != current.Score.Value {
		failures = append(failures, fmt.Sprintf("score.value mismatch: got %.1f, want %.1f", got.Score.Value, current.Score.Value))
	}
	if got.Score.Status == "" {
		failures = append(failures, "score.status is required")
	}
	if got.Score.Threshold == 0 {
		failures = append(failures, "score.threshold is required")
	}
	if minScore > 0 {
		if err := releasequality.Verify(got.Score, minScore); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if requireClean && got.TreeState != "clean" {
		failures = append(failures, fmt.Sprintf("tree_state must be clean, got %q", got.TreeState))
	}
	if !reflect.DeepEqual(got.Contracts, current.Contracts) {
		failures = append(failures, "contract fingerprints do not match current contract files")
	}
	if !reflect.DeepEqual(got.Dependencies, current.Dependencies) {
		failures = append(failures, "dependency inventory does not match go list -m -json all")
	}
	if !reflect.DeepEqual(got.StandardImpact, current.StandardImpact) {
		failures = append(failures, "standard_impact does not match current standard impact evidence")
	}
	if !reflect.DeepEqual(got.Debt, current.Debt) {
		failures = append(failures, "debt does not match current debt evidence")
	}
	if !reflect.DeepEqual(got.GovernanceRuntime, current.GovernanceRuntime) {
		failures = append(failures, "governance_runtime does not match current context runtime evidence")
	}
	if governanceFailures := validateGovernanceRuntimeEvidence(got.GovernanceRuntime); len(governanceFailures) > 0 {
		failures = append(failures, "governance_runtime does not match current governance runtime evidence")
		failures = append(failures, governanceFailures...)
	}
	if got.DownstreamSyncRequired != current.DownstreamSyncRequired {
		failures = append(failures, fmt.Sprintf("downstream_sync_required mismatch: got %t, want %t", got.DownstreamSyncRequired, current.DownstreamSyncRequired))
	}
	if got.DownstreamSyncRequired != got.StandardImpact.DownstreamSyncRequired {
		failures = append(failures, "downstream_sync_required must match standard_impact.downstream_sync_required")
	}
	requireNonEmpty(&failures, "standard_impact.report_path", got.StandardImpact.ReportPath)
	requireNonEmpty(&failures, "standard_impact.status", got.StandardImpact.Status)
	requireEnumValue(&failures, "standard_impact.downstream_release_decision", got.StandardImpact.DownstreamReleaseDecision, downstreamReleaseDecisionValues)
	requireEnumValue(&failures, "standard_impact.repository_rules_release_decision", got.StandardImpact.RepositoryRulesReleaseDecision, repositoryRulesReleaseDecisionValues)
	if requirePassed {
		if got.StandardImpact.Status != "present" {
			failures = append(failures, fmt.Sprintf("standard_impact.status must be present, got %q", got.StandardImpact.Status))
		}
		requireNonEmpty(&failures, "standard_impact.report_sha256", got.StandardImpact.ReportSHA256)
		requireNonEmpty(&failures, "standard_impact.primary_downstream", got.StandardImpact.PrimaryDownstream)
	}
	requireNonEmpty(&failures, "debt.report_path", got.Debt.ReportPath)
	requireNonEmpty(&failures, "debt.markdown_path", got.Debt.MarkdownPath)
	requireNonEmpty(&failures, "debt.checksum_path", got.Debt.ChecksumPath)
	requireNonEmpty(&failures, "debt.status", got.Debt.Status)
	if requirePassed {
		if got.Debt.Status != "passed" {
			failures = append(failures, fmt.Sprintf("debt.status must be passed, got %q", got.Debt.Status))
		}
		requireNonEmpty(&failures, "debt.report_sha256", got.Debt.ReportSHA256)
		if got.Debt.Score < got.Debt.MinScore {
			failures = append(failures, fmt.Sprintf("debt.score %.1f is below minimum %.1f", got.Debt.Score, got.Debt.MinScore))
		}
		if got.Debt.CheckCount == 0 {
			failures = append(failures, "debt.check_count must be greater than zero")
		}
	}
	requireNonEmpty(&failures, "governance_runtime.runtime", got.GovernanceRuntime.Runtime)
	requireNonEmpty(&failures, "governance_runtime.schema_version", got.GovernanceRuntime.SchemaVersion)
	requireNonEmpty(&failures, "governance_runtime.status", got.GovernanceRuntime.Status)
	requireNonEmpty(&failures, "governance_runtime.profile_check", got.GovernanceRuntime.ProfileCheck)
	requireNonEmpty(&failures, "governance_runtime.release_target", got.GovernanceRuntime.ReleaseTarget)
	if len(got.GovernanceRuntime.Profiles) == 0 {
		failures = append(failures, "governance_runtime.profiles is required")
	}
	if len(got.GovernanceRuntime.LegacyAliases) == 0 {
		failures = append(failures, "governance_runtime.legacy_aliases is required")
	}
	if !reflect.DeepEqual(got.GeneratorEvidence, current.GeneratorEvidence) {
		failures = append(failures, "generator_evidence does not match current integration evidence")
	}
	requireNonEmpty(&failures, "generator_evidence.command", got.GeneratorEvidence.Command)
	if !got.GeneratorEvidence.Required {
		failures = append(failures, "generator_evidence.required must be true")
	}
	if len(got.GeneratorEvidence.Targets) == 0 {
		failures = append(failures, "generator_evidence.targets is required")
	}
	for _, artifact := range requiredArtifacts {
		if !contains(got.Artifacts, artifact) {
			failures = append(failures, fmt.Sprintf("artifacts must include %s", artifact))
		}
	}
	if got.Tools["go"] == "" {
		failures = append(failures, "tools.go must be recorded")
	}
	failures = append(failures, validateChecks(got.Checks, requirePassed)...)

	if len(failures) > 0 {
		return errors.New("release evidence verification failed:\n - " + strings.Join(failures, "\n - "))
	}
	return nil
}

// validateDockerEvidence 验证 Docker 证据。
func validateDockerEvidence(evidence DockerEvidence, requireDigests bool) []string {
	var failures []string
	requireNonEmpty(&failures, "docker.contract_version", evidence.ContractVersion)
	requireNonEmpty(&failures, "docker.go_version", evidence.GoVersion)
	requireNonEmpty(&failures, "docker.golangci_lint_version", evidence.GolangCILintVersion)
	requireNonEmpty(&failures, "docker.govulncheck_version", evidence.GovulncheckVersion)
	if evidence.ContractVersion != "" && evidence.ContractVersion != "docker-toolchain/v2" {
		failures = append(failures, fmt.Sprintf("docker.contract_version must be docker-toolchain/v2, got %q", evidence.ContractVersion))
	}
	if !evidence.Enabled {
		failures = append(failures, "docker.enabled must be true")
	}
	if !evidence.BuildKitRequired {
		failures = append(failures, "docker.buildkit_required must be true")
	}
	for _, mount := range []string{"go-build", "go-mod", "golangci-lint"} {
		if !contains(evidence.CacheMounts, mount) {
			failures = append(failures, fmt.Sprintf("docker.cache_mounts must include %s", mount))
		}
	}
	for _, validator := range dockerEvidenceValidators {
		if !contains(evidence.ValidatedBy, validator) {
			failures = append(failures, fmt.Sprintf("docker.validated_by must include %s", validator))
		}
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"docker.base_image", evidence.BaseImage},
		{"docker.toolchain_image", evidence.ToolchainImage},
		{"docker.runtime_image", evidence.RuntimeImage},
		{"docker.workflow_run_id", evidence.WorkflowRunID},
		{"docker.artifact_name", evidence.ArtifactName},
		{"docker.artifact_url", evidence.ArtifactURL},
	} {
		requireNonEmpty(&failures, field.name, field.value)
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"docker.base_image_digest", evidence.BaseImageDigest},
		{"docker.toolchain_image_digest", evidence.ToolchainImageDigest},
		{"docker.runtime_image_digest", evidence.RuntimeImageDigest},
	} {
		if requireDigests && field.value == "" {
			failures = append(failures, field.name+" is required")
			continue
		}
		if field.value != "" && !strings.HasPrefix(field.value, "sha256:") {
			failures = append(failures, fmt.Sprintf("%s must start with sha256:", field.name))
		}
	}
	return failures
}

// validateChecks 验证检查项状态。
func validateChecks(checks map[string]string, requirePassed bool) []string {
	var failures []string
	for _, name := range checkNames {
		status := strings.TrimSpace(checks[name])
		if status == "" {
			failures = append(failures, "checks."+name+" is required")
			continue
		}
		if requirePassed && status != "passed" {
			failures = append(failures, fmt.Sprintf("checks.%s must be passed, got %q", name, status))
		}
	}
	return failures
}

// validateGovernanceRuntimeEvidence 验证治理运行时证据。
func validateGovernanceRuntimeEvidence(evidence GovernanceRuntimeEvidence) []string {
	var failures []string
	if evidence.SchemaVersion != governanceRuntimeVersion {
		failures = append(failures, fmt.Sprintf("governance_runtime.schema_version must be %q, got %q", governanceRuntimeVersion, evidence.SchemaVersion))
	}
	if evidence.RuntimeVersion != governanceRuntimeVersion {
		failures = append(failures, fmt.Sprintf("governance_runtime.runtime_version must be %q, got %q", governanceRuntimeVersion, evidence.RuntimeVersion))
	}
	appendRequiredPassedStatuses(&failures, "governance_runtime.gate_statuses", evidence.GateStatuses, governanceRuntimeGateStatuses)
	appendRequiredPassedStatuses(&failures, "governance_runtime.profile_statuses", evidence.ProfileStatuses, governanceRuntimeProfileStatuses)
	return failures
}

// appendRequiredPassedStatuses 追加必需的已通过状态验证失败信息。
func appendRequiredPassedStatuses(failures *[]string, field string, got map[string]string, required map[string]string) {
	for name, want := range required {
		status := strings.TrimSpace(got[name])
		if status == "" {
			*failures = append(*failures, fmt.Sprintf("%s.%s is required", field, name))
			continue
		}
		if status != want {
			*failures = append(*failures, fmt.Sprintf("%s.%s must be %s, got %q", field, name, want, status))
		}
	}
}

// requireNonEmpty 验证值非空。
func requireNonEmpty(failures *[]string, field string, value string) {
	if strings.TrimSpace(value) == "" {
		*failures = append(*failures, field+" is required")
	}
}

// requireEnumValue 验证值在允许列表中。
func requireEnumValue(failures *[]string, field string, value string, allowed []string) {
	if strings.TrimSpace(value) == "" {
		*failures = append(*failures, field+" is required")
		return
	}
	if contains(allowed, value) {
		return
	}
	*failures = append(*failures, fmt.Sprintf("%s must be one of %s, got %q", field, strings.Join(allowed, ", "), value))
}

// contains 检查字符串切片是否包含指定值。
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
