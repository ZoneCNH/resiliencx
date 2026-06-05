// SPDX-License-Identifier: Apache-2.0
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/ZoneCNH/resiliencx/internal/debtcheck"
)

// PlaceholderImageDigest 是未构建镜像时的默认 digest（空字符串的 SHA256）。
// 构建后由 docker_gate.sh 用真实 digest 覆盖。
const PlaceholderImageDigest = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// buildDockerEvidence 构建 Docker 工具链证据。
func buildDockerEvidence() DockerEvidence {
	workflow := buildWorkflowEvidence()
	toolchainImage := envDefault("DOCKER_TOOLCHAIN_IMAGE", envDefault("DOCKER_IMAGE", "resiliencx-toolchain:local"))
	return DockerEvidence{
		Enabled:              envBool("DOCKER_TOOLCHAIN_ENABLED", true),
		ContractVersion:      envDefault("DOCKER_CONTRACT_VERSION", "docker-toolchain/v2"),
		GoVersion:            envDefault("DOCKER_GO_VERSION", "1.23"),
		GolangCILintVersion:  envDefault("DOCKER_GOLANGCI_LINT_VERSION", "golangci-lint v2.1.6"),
		GovulncheckVersion:   envDefault("DOCKER_GOVULNCHECK_VERSION", "govulncheck v1.1.4"),
		BuildKitRequired:     envBool("DOCKER_BUILDKIT_REQUIRED", true),
		CacheMounts:          envCSVDefault("DOCKER_CACHE_MOUNTS", []string{"go-build", "go-mod", "golangci-lint"}),
		BaseImage:            envDefault("DOCKER_BASE_IMAGE", "golang:1.23-bookworm"),
		BaseImageDigest:      envDefault("DOCKER_BASE_IMAGE_DIGEST", PlaceholderImageDigest),
		ToolchainImage:       toolchainImage,
		ToolchainImageDigest: envDefault("DOCKER_TOOLCHAIN_IMAGE_DIGEST", PlaceholderImageDigest),
		RuntimeImage:         envDefault("DOCKER_RUNTIME_IMAGE", "resiliencx-goalcli-runtime:local"),
		RuntimeImageDigest:   envDefault("DOCKER_RUNTIME_IMAGE_DIGEST", PlaceholderImageDigest),
		ValidatedBy:          envCSVDefault("DOCKER_VALIDATED_BY", dockerEvidenceValidators),
		WorkflowRunID:        workflow.WorkflowRunID,
		ArtifactName:         envDefault("DOCKER_ARTIFACT_NAME", workflow.ArtifactName),
		ArtifactURL:          envDefault("DOCKER_ARTIFACT_URL", workflow.ArtifactURL),
	}
}

// buildStandardImpactEvidence 构建标准影响证据。
func buildStandardImpactEvidence() (StandardImpactEvidence, error) {
	evidence := StandardImpactEvidence{
		ReportPath:                     standardImpactReportPath,
		Status:                         "missing",
		DownstreamReleaseDecision:      "not_required",
		RepositoryRulesReleaseDecision: "not_required",
	}

	data, err := os.ReadFile(standardImpactReportPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return evidence, nil
		}
		return StandardImpactEvidence{}, err
	}

	sum := sha256.Sum256(data)
	report := string(data)
	evidence.ReportSHA256 = "sha256:" + hex.EncodeToString(sum[:])
	evidence.Status = "present"
	evidence.DownstreamSyncRequired = strings.EqualFold(parseReportValue(report, "downstream_sync_required"), "true")
	evidence.ContextRuntimeChange = strings.EqualFold(parseReportValue(report, "context_runtime_change"), "true")
	evidence.GovernanceRegistryChange = strings.EqualFold(parseReportValue(report, "governance_registry_change"), "true")
	evidence.PrimaryDownstream = parseReportValue(report, "primary_downstream")
	evidence.DownstreamReleaseDecision = reportValueDefault(report, "downstream_release_decision", "not_required")
	evidence.RepositoryRulesReleaseDecision = reportValueDefault(report, "repository_rules_release_decision", "not_required")
	return evidence, nil
}

// buildDebtEvidence 构建债务证据。
func buildDebtEvidence() (DebtEvidence, error) {
	evidence := DebtEvidence{
		ReportPath:   debtReportPath,
		MarkdownPath: debtMarkdownPath,
		ChecksumPath: debtChecksumPath,
		Status:       "missing",
		MinScore:     debtcheck.DefaultMinScore,
	}

	data, err := os.ReadFile(debtReportPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return evidence, nil
		}
		return DebtEvidence{}, err
	}

	sum := sha256.Sum256(data)
	evidence.ReportSHA256 = "sha256:" + hex.EncodeToString(sum[:])

	var report struct {
		Status   string            `json:"status"`
		Score    float64           `json:"score"`
		MinScore float64           `json:"min_score"`
		Checks   []json.RawMessage `json:"checks"`
		Sections []json.RawMessage `json:"sections"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return DebtEvidence{}, err
	}

	evidence.Status = report.Status
	evidence.Score = report.Score
	if report.MinScore != 0 {
		evidence.MinScore = report.MinScore
	}
	evidence.CheckCount = len(report.Checks)
	if evidence.CheckCount == 0 {
		evidence.CheckCount = len(report.Sections)
	}
	return evidence, nil
}

// buildGovernanceRuntime 构建治理运行时证据。
func buildGovernanceRuntime() GovernanceRuntime {
	evidence := buildGovernanceRuntimeEvidence()
	evidence.Runtime = "context-runtime-v4.0"
	evidence.Status = "present"
	evidence.Profiles = []string{
		"context-lite",
		"context-standard",
		"context-full",
		"context-release",
	}
	evidence.ProfileCheck = "context-profile-check"
	evidence.ReleaseTarget = "context-release"
	evidence.LegacyAliases = []string{
		"context-fast-check",
		"context-standard-check",
		"context-full-check",
	}
	return evidence
}

// buildGovernanceRuntimeEvidence 构建治理运行时证据基础数据。
func buildGovernanceRuntimeEvidence() GovernanceRuntimeEvidence {
	return GovernanceRuntimeEvidence{
		SchemaVersion:   governanceRuntimeVersion,
		RuntimeVersion:  governanceRuntimeVersion,
		GateStatuses:    copyStatusMap(governanceRuntimeGateStatuses),
		ProfileStatuses: copyStatusMap(governanceRuntimeProfileStatuses),
	}
}

// copyStatusMap 深拷贝状态映射。
func copyStatusMap(statuses map[string]string) map[string]string {
	copied := make(map[string]string, len(statuses))
	for name, status := range statuses {
		copied[name] = status
	}
	return copied
}

// buildGeneratorEvidence 构建生成器证据。
func buildGeneratorEvidence() GeneratorEvidence {
	return GeneratorEvidence{
		Command:  "GOWORK=off make integration",
		Required: true,
		Targets:  append([]GeneratorTarget(nil), generatorEvidenceTargets...),
	}
}

// parseReportValue 从报告文本中解析指定键的值。
func parseReportValue(report string, key string) string {
	prefix := key + ":"
	for _, line := range strings.Split(report, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			value = strings.Trim(value, "`")
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// reportValueDefault 从报告中解析值，若为空则返回默认值。
func reportValueDefault(report string, key string, fallback string) string {
	value := parseReportValue(report, key)
	if value == "" {
		return fallback
	}
	return value
}
