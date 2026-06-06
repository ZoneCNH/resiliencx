// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/ZoneCNH/resiliencx/internal/releasequality"
)

// Manifest 是发布清单的顶层结构。
type Manifest struct {
	Module                 string                 `json:"module"`
	Version                string                 `json:"version"`
	Commit                 string                 `json:"commit"`
	TreeSHA                string                 `json:"tree_sha"`
	SourceDigest           string                 `json:"source_digest"`
	TrackedFileCount       int                    `json:"tracked_file_count"`
	GoVersion              string                 `json:"go_version"`
	GeneratedAt            string                 `json:"generated_at"`
	GeneratedBy            string                 `json:"generated_by"`
	TreeState              string                 `json:"tree_state"`
	Checks                 map[string]string      `json:"checks"`
	Workflow               WorkflowEvidence       `json:"workflow"`
	Docker                 DockerEvidence         `json:"docker"`
	Score                  releasequality.Report  `json:"score"`
	Contracts              []FileDigest           `json:"contracts"`
	Dependencies           []ModuleDigest         `json:"dependencies"`
	StandardImpact         StandardImpactEvidence `json:"standard_impact"`
	Debt                   DebtEvidence           `json:"debt"`
	GovernanceRuntime      GovernanceRuntime      `json:"governance_runtime"`
	DownstreamSyncRequired bool                   `json:"downstream_sync_required"`
	GeneratorEvidence      GeneratorEvidence      `json:"generator_evidence"`
	Tools                  map[string]string      `json:"tools"`
	Artifacts              []string               `json:"artifacts"`
	Notes                  Notes                  `json:"notes"`
}

type WorkflowEvidence struct {
	WorkflowRunID string `json:"workflow_run_id"`
	ArtifactName  string `json:"artifact_name"`
	ArtifactURL   string `json:"artifact_url"`
}

type DockerEvidence struct {
	Enabled              bool     `json:"enabled"`
	ContractVersion      string   `json:"contract_version"`
	GoVersion            string   `json:"go_version"`
	GolangCILintVersion  string   `json:"golangci_lint_version"`
	GovulncheckVersion   string   `json:"govulncheck_version"`
	BuildKitRequired     bool     `json:"buildkit_required"`
	CacheMounts          []string `json:"cache_mounts"`
	BaseImage            string   `json:"base_image"`
	BaseImageDigest      string   `json:"base_image_digest"`
	ToolchainImage       string   `json:"toolchain_image"`
	ToolchainImageDigest string   `json:"toolchain_image_digest"`
	RuntimeImage         string   `json:"runtime_image"`
	RuntimeImageDigest   string   `json:"runtime_image_digest"`
	ValidatedBy          []string `json:"validated_by"`
	WorkflowRunID        string   `json:"workflow_run_id"`
	ArtifactName         string   `json:"artifact_name"`
	ArtifactURL          string   `json:"artifact_url"`
}

type FileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type ModuleDigest struct {
	Path    string         `json:"path"`
	Version string         `json:"version,omitempty"`
	Main    bool           `json:"main,omitempty"`
	Replace *ModuleReplace `json:"replace,omitempty"`
}

type ModuleReplace struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

type StandardImpactEvidence struct {
	ReportPath                     string `json:"report_path"`
	ReportSHA256                   string `json:"report_sha256"`
	Status                         string `json:"status"`
	DownstreamSyncRequired         bool   `json:"downstream_sync_required"`
	ContextRuntimeChange           bool   `json:"context_runtime_change"`
	GovernanceRegistryChange       bool   `json:"governance_registry_change"`
	DownstreamReleaseDecision      string `json:"downstream_release_decision"`
	RepositoryRulesReleaseDecision string `json:"repository_rules_release_decision"`
	PrimaryDownstream              string `json:"primary_downstream"`
}

type DebtEvidence struct {
	ReportPath   string  `json:"report_path"`
	MarkdownPath string  `json:"markdown_path"`
	ChecksumPath string  `json:"checksum_path"`
	ReportSHA256 string  `json:"report_sha256"`
	Status       string  `json:"status"`
	Score        float64 `json:"score"`
	MinScore     float64 `json:"min_score"`
	CheckCount   int     `json:"check_count"`
}

type GovernanceRuntime struct {
	Runtime         string            `json:"runtime"`
	SchemaVersion   string            `json:"schema_version"`
	RuntimeVersion  string            `json:"runtime_version"`
	Status          string            `json:"status"`
	Profiles        []string          `json:"profiles"`
	ProfileCheck    string            `json:"profile_check"`
	ReleaseTarget   string            `json:"release_target"`
	LegacyAliases   []string          `json:"legacy_aliases"`
	GateStatuses    map[string]string `json:"gate_statuses"`
	ProfileStatuses map[string]string `json:"profile_statuses"`
}

type GovernanceRuntimeEvidence = GovernanceRuntime

type GeneratorEvidence struct {
	Command  string            `json:"command"`
	Required bool              `json:"required"`
	Targets  []GeneratorTarget `json:"targets"`
}

type GeneratorTarget struct {
	Name        string `json:"name"`
	ModulePath  string `json:"module_path"`
	PackageName string `json:"package_name"`
}

type Notes struct {
	BreakingChanges string   `json:"breaking_changes"`
	KnownRisks      []string `json:"known_risks"`
}
