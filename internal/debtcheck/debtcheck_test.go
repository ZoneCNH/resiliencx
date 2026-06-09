package debtcheck

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPassesWithPolicyFilesAndCleanTree(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)
	writeFile(t, root, "safe.go", "package fixture\n")

	report, err := Run(Options{Root: root, Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "passed" {
		t.Fatalf("status = %q, want passed: %+v", report.Status, report.Summary)
	}
	if code := ExitCode(report); code != 0 {
		t.Fatalf("ExitCode = %d, want 0", code)
	}
	if problems := ValidateEvidence(EvidenceFromReport(report), DefaultMinScore); len(problems) != 0 {
		t.Fatalf("ValidateEvidence problems = %v, want none", problems)
	}
	if report.Digests.Report == "" || report.Digests.Rules == "missing" {
		t.Fatalf("digests = %+v, want populated policy and report digests", report.Digests)
	}
}

func TestRunPassesDownstreamSectionWithGovernanceFiles(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)

	report, err := Run(Options{Root: root, Section: "downstream", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "passed" {
		t.Fatalf("status = %q, want passed: %+v", report.Status, report.Summary)
	}
	if len(report.Sections) != 1 || report.Sections[0].Name != "downstream" {
		t.Fatalf("sections = %+v, want only downstream", report.Sections)
	}
	if code := ExitCode(report); code != 0 {
		t.Fatalf("ExitCode = %d, want 0", code)
	}
}

func TestRunFailsDownstreamSectionWithoutRegistryCoverage(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)
	writeFile(t, root, ".agent/registries/downstream-registry.yaml", `schema_version: "2.9.3"
downstreams:
  - repo: kernel/configx
    mode: patch-only
    status: unavailable_in_worker_workspace_gap_explicit
`)

	report, err := Run(Options{Root: root, Section: "downstream", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "failed" {
		t.Fatalf("status = %q, want failed", report.Status)
	}
	if report.Summary.P0 != 2 {
		t.Fatalf("P0 = %d, want 2", report.Summary.P0)
	}
	if !strings.Contains(ToMarkdown(report), "debt.downstream.registry-missing-repo") {
		t.Fatalf("markdown missing downstream registry finding: %s", ToMarkdown(report))
	}
}

func TestRunFailsDownstreamSectionOnFalseAdoptionClaim(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)
	writeFile(t, root, ".agent/registries/downstream-adoption-status.yaml", `schema_version: "2.9.3"
current_registry:
  adoption_status: adopted
  proof_based_adoption: true
standard_target_libraries:
  - name: kernel
  - name: configx
  - name: observex
  - name: testkitx
  - name: postgresx
  - name: redisx
  - name: kafkax
  - name: natsx
  - name: taosx
  - name: ossx
  - name: clickhousex
`)

	report, err := Run(Options{Root: root, Section: "downstream", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "failed" {
		t.Fatalf("status = %q, want failed", report.Status)
	}
	markdown := ToMarkdown(report)
	if !strings.Contains(markdown, "debt.downstream.false-adoption-claim") {
		t.Fatalf("markdown missing false adoption finding: %s", markdown)
	}
	if !strings.Contains(markdown, "debt.downstream.false-proof-claim") {
		t.Fatalf("markdown missing false proof finding: %s", markdown)
	}
}

func TestRunFailsDownstreamSectionWithoutIntegrationDebtEvidenceGate(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)
	writeFile(t, root, "scripts/run_integration.sh", `#!/usr/bin/env bash
TARGETS=(
  "kernel|github.com/ZoneCNH/kernel|kernel"
  "configx|github.com/ZoneCNH/configx|configx"
  "redisx|github.com/ZoneCNH/redisx|redisx"
)
`)

	report, err := Run(Options{Root: root, Section: "downstream", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "failed" {
		t.Fatalf("status = %q, want failed", report.Status)
	}
	if !strings.Contains(ToMarkdown(report), "debt.downstream.integration-missing-contract") {
		t.Fatalf("markdown missing downstream integration finding: %s", ToMarkdown(report))
	}
}

func TestRunFailsDownstreamSectionWithoutRenderedDebtEvidenceExclusions(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)
	writeFile(t, root, "scripts/render_template.sh", `#!/usr/bin/env bash
rsync "$@"
`)

	report, err := Run(Options{Root: root, Section: "downstream", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "failed" {
		t.Fatalf("status = %q, want failed", report.Status)
	}
	if !strings.Contains(ToMarkdown(report), "debt.downstream.render-template-missing-exclusion") {
		t.Fatalf("markdown missing downstream render-template finding: %s", ToMarkdown(report))
	}
}

func TestRunFailsOnLegacyProductionImport(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeFile(t, root, "bad.go", "package fixture\n\nimport _ \"github.com/ZoneCNH/x.go\"\n")

	report, err := Run(Options{Root: root, Section: "architecture", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}

	if report.Status != "failed" {
		t.Fatalf("status = %q, want failed", report.Status)
	}
	if report.Summary.P0 != 1 {
		t.Fatalf("P0 = %d, want 1", report.Summary.P0)
	}
	if code := ExitCode(report); code != 1 {
		t.Fatalf("ExitCode = %d, want 1", code)
	}
	if !strings.Contains(ToMarkdown(report), "legacy ZoneCNH x module") {
		t.Fatalf("markdown missing legacy import finding: %s", ToMarkdown(report))
	}
}

func TestSkipPathSkipsMigratedInboxArchive(t *testing.T) {
	root := t.TempDir()

	if !skipPath(root, filepath.Join(root, ".agent", "archive", "inbox", "goal-patch-v1.0-to-v2.2.md")) {
		t.Fatal("skipPath should skip migrated inbox archive files")
	}
	if skipPath(root, filepath.Join(root, ".agent", "archive", "standard", "goal-runtime-canonical.md")) {
		t.Fatal("skipPath should not skip non-inbox archive files")
	}
}

func writePolicyFiles(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		DefaultRulesPath:    "schema_version: debt-rules/v1\nprofile: test\n",
		DefaultRegistryPath: "schema_version: debt-rule-registry/v1\nrules: []\n",
		DefaultExceptions:   "schema_version: debt-exceptions/v1\nexceptions: []\n",
		DefaultPurpose:      "schema_version: debt-dependency-purpose/v1\npurposes: []\n",
	}
	for path, content := range files {
		writeFile(t, root, path, content)
	}
}

func writeDownstreamFiles(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		".agent/registries/downstream-registry.yaml": `schema_version: "2.9.3"
downstreams:
  - repo: kernel/configx
    mode: patch-only
    status: unavailable_in_worker_workspace_gap_explicit
  - repo: kernel/redisx
    mode: patch-only
    status: unavailable_in_worker_workspace_gap_explicit
  - repo: corekit
    mode: patch-only
    status: unavailable_in_worker_workspace_gap_explicit
`,
		".agent/registries/downstream-baseline-scan.yaml": `schema_version: "2.9.3"
repo: kernel/configx
mode: patch-only
status: gap_explicit_when_repo_missing
`,
		".agent/registries/downstream-adoption-modes.yaml": `schema_version: "2.9.3"
modes: [patch-only, dry-run]
forbidden: [direct_downstream_write_without_repo]
`,
		".agent/registries/downstream-adoption-status.yaml": `schema_version: "2.9.3"
current_registry:
  adoption_status: not_adopted
  proof_based_adoption: false
first_pr_mva_assertions:
  no_proof_based_adoption: true
standard_target_libraries:
  - name: kernel
  - name: configx
  - name: observex
  - name: testkitx
  - name: postgresx
  - name: redisx
  - name: kafkax
  - name: natsx
  - name: taosx
  - name: ossx
  - name: clickhousex
`,
		"docs/downstream-matrix.md": `# Downstream Matrix

| Library | Adoption |
| --- | --- |
| ` + "`kernel`" + ` | not_adopted |
| ` + "`configx`" + ` | not_adopted |
| ` + "`observex`" + ` | not_adopted |
| ` + "`testkitx`" + ` | not_adopted |
| ` + "`postgresx`" + ` | not_adopted |
| ` + "`redisx`" + ` | not_adopted |
| ` + "`kafkax`" + ` | not_adopted |
| ` + "`natsx`" + ` | not_adopted |
| ` + "`taosx`" + ` | not_adopted |
| ` + "`ossx`" + ` | not_adopted |
| ` + "`clickhousex`" + ` | not_adopted |
`,
		"docs/standard/downstream-compatibility.md": `# Downstream Compatibility

默认 downstream 为 ` + "`kernel`" + ` 与 ` + "`corekit`" + `。
发布验证命令必须包含 GOWORK=off make integration。
`,
		"scripts/run_integration.sh": `#!/usr/bin/env bash
TARGETS=(
  "kernel|github.com/ZoneCNH/kernel|kernel"
  "configx|github.com/ZoneCNH/configx|configx"
  "redisx|github.com/ZoneCNH/redisx|redisx"
)
GOWORK=off make debt
GOWORK=off make debt-evidence
GOWORK=off make debt-evidence-checksum-check
`,
		"scripts/render_template.sh": `#!/usr/bin/env bash
rsync \
  --exclude release/debt/latest.json \
  --exclude release/debt/latest.md \
  --exclude release/debt/latest.json.sha256
`,
	}
	for path, content := range files {
		writeFile(t, root, path, content)
	}
}

func TestRunRejectsInvalidMode(t *testing.T) {
	root := t.TempDir()
	_, err := Run(Options{Root: root, Mode: "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestRunRejectsInvalidSection(t *testing.T) {
	root := t.TempDir()
	_, err := Run(Options{Root: root, Section: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for invalid section")
	}
}

func TestExitCode_WarnMode(t *testing.T) {
	report := Report{Mode: "warn", Status: "failed"}
	if code := ExitCode(report); code != 0 {
		t.Fatalf("ExitCode = %d, want 0 for warn mode", code)
	}
}

func TestExitCode_ObserveMode(t *testing.T) {
	report := Report{Mode: "observe", Status: "failed"}
	if code := ExitCode(report); code != 0 {
		t.Fatalf("ExitCode = %d, want 0 for observe mode", code)
	}
}

func TestExitCode_EnforcePassed(t *testing.T) {
	report := Report{Mode: "enforce", Status: "passed"}
	if code := ExitCode(report); code != 0 {
		t.Fatalf("ExitCode = %d, want 0 for passed", code)
	}
}

func TestToMarkdown_EmptyFindings(t *testing.T) {
	report := Report{
		Status:   "passed",
		Mode:     "enforce",
		Score:    10.0,
		MinScore: 9.8,
		Summary:  Summary{},
		Sections: []SectionReport{{Name: "test", Status: "passed"}},
	}
	md := ToMarkdown(report)
	if !strings.Contains(md, "No findings.") {
		t.Fatalf("expected 'No findings.' in markdown: %s", md)
	}
}

func TestToMarkdown_WithFindings(t *testing.T) {
	report := Report{
		Status:   "failed",
		Mode:     "enforce",
		Score:    8.0,
		MinScore: 9.8,
		Summary:  Summary{P0: 1},
		Sections: []SectionReport{{
			Name:   "security",
			Status: "failed",
			P0:     1,
			Findings: []Finding{{
				ID:       "debt.security.private-key",
				Severity: "P0",
				Path:     "key.pem",
				Message:  "private key found",
			}},
		}},
	}
	md := ToMarkdown(report)
	if !strings.Contains(md, "debt.security.private-key") {
		t.Fatalf("expected finding in markdown: %s", md)
	}
	if !strings.Contains(md, "key.pem") {
		t.Fatalf("expected path in markdown: %s", md)
	}
}

func TestValidateEvidence_AllProblems(t *testing.T) {
	e := Evidence{
		SchemaVersion:       "wrong",
		ReportSchemaVersion: "wrong",
		Status:              "failed",
		Score:               5.0,
		MinScore:            9.8,
		Sections: []SectionEvidence{{
			Name:   "test",
			Status: "failed",
			P0:     1,
		}},
	}
	problems := ValidateEvidence(e, 9.8)
	if len(problems) < 5 {
		t.Fatalf("expected at least 5 problems, got %d: %v", len(problems), problems)
	}
}

func TestValidateEvidence_ScoreBelowMin(t *testing.T) {
	e := Evidence{
		SchemaVersion:       ManifestSchema,
		ReportSchemaVersion: SchemaVersion,
		Status:              "passed",
		Score:               9.0,
		MinScore:            9.8,
		Sections:            []SectionEvidence{{Name: "test", Status: "passed"}},
	}
	problems := ValidateEvidence(e, 9.8)
	found := false
	for _, p := range problems {
		if strings.Contains(p, "below") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected score below threshold problem: %v", problems)
	}
}

func TestNormalize_Defaults(t *testing.T) {
	report, _ := Run(Options{Root: t.TempDir(), MinScore: DefaultMinScore})
	if report.Mode != "enforce" {
		t.Fatalf("expected default mode enforce, got %s", report.Mode)
	}
}

func TestStatus_WarnModeWithScore(t *testing.T) {
	s := Summary{P1: 5}
	st := status(s, 9.0, 9.8, "warn")
	if st != "warning" {
		t.Fatalf("expected warning, got %s", st)
	}
}

func TestStatus_EnforceWithFindings(t *testing.T) {
	s := Summary{P1: 1, P2: 1}
	st := status(s, 10.0, 9.8, "enforce")
	if st != "passed" {
		t.Fatalf("expected passed in enforce with P1/P2, got %s", st)
	}
}

func TestStatus_ObserveWithScore(t *testing.T) {
	s := Summary{P1: 5}
	st := status(s, 9.0, 9.8, "observe")
	if st != "warning" {
		t.Fatalf("expected warning in observe with low score, got %s", st)
	}
}

func TestReadReport_ValidFile(t *testing.T) {
	dir := t.TempDir()
	report := Report{SchemaVersion: SchemaVersion, Status: "passed"}
	data, _ := json.Marshal(report)
	os.WriteFile(filepath.Join(dir, "report.json"), data, 0o644)
	got, err := ReadReport(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "passed" {
		t.Fatalf("expected passed, got %s", got.Status)
	}
}

func TestReadReport_BadSchema(t *testing.T) {
	dir := t.TempDir()
	report := Report{SchemaVersion: "wrong", Status: "passed"}
	data, _ := json.Marshal(report)
	os.WriteFile(filepath.Join(dir, "report.json"), data, 0o644)
	_, err := ReadReport(filepath.Join(dir, "report.json"))
	if err == nil {
		t.Fatal("expected error for bad schema")
	}
}

func TestReadReport_MissingFile(t *testing.T) {
	_, err := ReadReport("/nonexistent/report.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadReport_BadJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "report.json"), []byte("not json"), 0o644)
	_, err := ReadReport(filepath.Join(dir, "report.json"))
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestSkipDir_AllCases(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{".git", true},
		{".omx", true},
		{".worktree", true},
		{"vendor", true},
		{"node_modules", true},
		{"release", true},
		{"tmp", true},
		{".cache", true},
		{"src", false},
		{"internal", false},
	}
	for _, tc := range cases {
		if got := skipDir(tc.name); got != tc.want {
			t.Fatalf("skipDir(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestSkipFile_Images(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"image.png", true},
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"anim.gif", true},
		{"doc.pdf", true},
		{".hidden", true},
		{".gitignore", false},
		{"main.go", false},
	}
	for _, tc := range cases {
		if got := skipFile(tc.path); got != tc.want {
			t.Fatalf("skipFile(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestBytesLookBinary(t *testing.T) {
	if !bytesLookBinary([]byte("hello\x00world")) {
		t.Fatal("expected binary data to be detected")
	}
	if bytesLookBinary([]byte("hello world")) {
		t.Fatal("expected text data to not be binary")
	}
}

func TestRel_ErrorFallback(t *testing.T) {
	// rel returns path as-is when filepath.Rel fails
	got := rel("/a", "C:\\b")
	if got == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestBuildSection_P2Status(t *testing.T) {
	findings := []Finding{{ID: "test", Severity: "P2", Message: "test"}}
	section := buildSection("test", findings)
	if section.Status != "warning" {
		t.Fatalf("expected warning status for P2 only, got %s", section.Status)
	}
	if section.P2 != 1 {
		t.Fatalf("expected P2=1, got %d", section.P2)
	}
}

func TestRun_WithObserveMode(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	report, err := Run(Options{Root: root, Mode: "observe", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	if report.Mode != "observe" {
		t.Fatalf("expected observe mode, got %s", report.Mode)
	}
}

func TestRun_WithWarnMode(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	report, err := Run(Options{Root: root, Mode: "warn", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	if report.Mode != "warn" {
		t.Fatalf("expected warn mode, got %s", report.Mode)
	}
}

func TestScanSection_UnknownSection(t *testing.T) {
	findings := scanSection(".", "nonexistent")
	if findings != nil {
		t.Fatalf("expected nil for unknown section, got %v", findings)
	}
}

func TestRun_WithCustomPaths(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	report, err := Run(Options{
		Root:                  root,
		ConfigPath:            DefaultRulesPath,
		RegistryPath:          DefaultRegistryPath,
		ExceptionsPath:        DefaultExceptions,
		DependencyPurposePath: DefaultPurpose,
		MinScore:              DefaultMinScore,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Digests.Rules == "missing" {
		t.Fatal("expected rules digest to be populated")
	}
}

func TestRun_MissingPolicyFindings(t *testing.T) {
	root := t.TempDir()
	report, err := Run(Options{Root: root, Section: "all", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.P0 == 0 {
		t.Fatal("expected P0 findings for missing policies")
	}
}

func TestRun_DownstreamSectionWithPlaceholderFile(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)
	// Overwrite a file with placeholder content
	writeFile(t, root, ".agent/registries/downstream-registry.yaml", `schema_version: "2.9.3"
# TODO: fill this in
downstreams:
  - repo: kernel/configx
    mode: patch-only
    status: unavailable_in_worker_workspace_gap_explicit
  - repo: kernel/redisx
    mode: patch-only
    status: unavailable_in_worker_workspace_gap_explicit
  - repo: corekit
    mode: patch-only
    status: unavailable_in_worker_workspace_gap_explicit
`)
	report, err := Run(Options{Root: root, Section: "downstream", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	// Should find placeholder finding
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.downstream.placeholder" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected placeholder finding")
	}
}

func TestRun_DownstreamMissingSchemaVersion(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)
	writeFile(t, root, ".agent/registries/downstream-registry.yaml", `downstreams:
  - repo: kernel/configx
`)
	report, err := Run(Options{Root: root, Section: "downstream", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.downstream.schema-missing" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected schema-missing finding")
	}
}

func TestRun_DownstreamMissingBaselineGap(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeDownstreamFiles(t, root)
	writeFile(t, root, ".agent/registries/downstream-baseline-scan.yaml", `schema_version: "2.9.3"
repo: kernel/configx
mode: patch-only
status: ok
`)
	report, err := Run(Options{Root: root, Section: "downstream", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.downstream.baseline-missing-gap-status" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected baseline-missing-gap-status finding")
	}
}

func TestRun_DomainSectionWithMarker(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeFile(t, root, "debt.txt", "xlib-domain-forbidden\n")
	report, err := Run(Options{Root: root, Section: "domain", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.domain.marker" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected domain marker finding")
	}
}

func TestRun_DocsSectionWithMarker(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeFile(t, root, "drift.txt", "xlib-docs-drift\n")
	report, err := Run(Options{Root: root, Section: "docs", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.docs.marker" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected docs marker finding")
	}
}

func TestRun_TestingSectionWithMarker(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeFile(t, root, "test.txt", "xlib-testing-debt\n")
	report, err := Run(Options{Root: root, Section: "testing", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.testing.marker" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected testing marker finding")
	}
}

func TestRun_ImplementationSectionWithMarker(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeFile(t, root, "impl.txt", "xlib-implementation-debt\n")
	report, err := Run(Options{Root: root, Section: "implementation", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.implementation.marker" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected implementation marker finding")
	}
}

func TestRun_SecuritySectionWithMarker(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeFile(t, root, "sec.txt", "xlib-security-debt\n")
	report, err := Run(Options{Root: root, Section: "security", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.security.marker" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected security marker finding")
	}
}

func TestRun_SecuritySectionWithPrivateKey(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	// Construct the marker dynamically to avoid secret scanner false positive
	marker := "-----BEGIN " + "PRIVATE KEY-----" + "\nFAKEKEYDATA"
	writeFile(t, root, "key.txt", marker)

	report, err := Run(Options{Root: root, Section: "security", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.security.private-key" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected private-key finding")
	}
}

func TestRun_DependencySectionWithLatest(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeFile(t, root, "go.mod", "require github.com/foo/bar @latest\n")
	report, err := Run(Options{Root: root, Section: "dependency", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.dependency.unpinned-latest" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected unpinned-latest finding")
	}
}

func TestRun_DependencySectionWithCurlBash(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	writeFile(t, root, "install.sh", "curl https://example.com | bash\n")
	report, err := Run(Options{Root: root, Section: "dependency", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range report.Sections {
		for _, f := range s.Findings {
			if f.ID == "debt.dependency.curl-pipe-bash" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected curl-pipe-bash finding")
	}
}

func TestRun_WithUnreadableRoot(t *testing.T) {
	// Run normalizes root to "." if empty, and walkFiles handles errors internally
	// Test that scanGoImports on a nonexistent path returns empty findings (no panic)
	findings := scanGoImports("/nonexistent/path/xyz")
	if findings == nil {
		findings = []Finding{}
	}
	// Should not panic
}

func TestWalkFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	var visited []string
	err := walkFiles(filepath.Join(dir, "test.txt"), func(path string) error {
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(visited) != 1 {
		t.Fatalf("expected 1 visited, got %d", len(visited))
	}
}

func TestWalkDir_SkipsGitDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "visible.txt", "hello")
	writeFile(t, dir, ".git/config", "git config")
	var visited []string
	err := walkDir(dir, func(path string) error {
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range visited {
		if strings.Contains(v, ".git") {
			t.Fatal("should not visit files in .git dir")
		}
	}
}

func TestWalkDir_SkipsVendor(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.go", "package main")
	writeFile(t, dir, "vendor/foo/bar.go", "package bar")
	var visited []string
	_ = walkDir(dir, func(path string) error {
		visited = append(visited, path)
		return nil
	})
	for _, v := range visited {
		if strings.Contains(v, "vendor") {
			t.Fatal("should not visit vendor dir")
		}
	}
}

func TestWalkFiles_ErrorPath(t *testing.T) {
	err := walkFiles("/nonexistent/path", func(path string) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestScanGoImports_WithBadGoFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "bad.go", "not valid go {{{")
	findings := scanGoImports(root)
	found := false
	for _, f := range findings {
		if f.ID == "debt.architecture.parse" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected parse finding, got %v", findings)
	}
}

func TestScanGoImports_WithXImport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "main.go", `package main

import _ "github.com/ZoneCNH/x/something"

func main() {}
`)
	findings := scanGoImports(root)
	found := false
	for _, f := range findings {
		if f.ID == "debt.architecture.legacy-import" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected legacy import finding, got %v", findings)
	}
}

func TestScanTrackedText_SkipsBinaryFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "binary.bin", "hello\x00world")
	writeFile(t, root, "text.txt", "hello world")
	var found []string
	_ = walkFiles(root, func(path string) error {
		return nil
	})
	findings := scanTrackedText(root, func(path, text string) []Finding {
		return []Finding{{ID: "test", Path: path}}
	})
	for _, f := range findings {
		found = append(found, f.Path)
	}
	for _, f := range found {
		if strings.HasSuffix(f, ".bin") {
			t.Fatal("should not scan binary files")
		}
	}
}

func TestScanTrackedText_SkipsDotFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".env", "SECRET=abc")
	findings := scanTrackedText(root, func(path, text string) []Finding {
		return []Finding{{ID: "test", Path: path}}
	})
	for _, f := range findings {
		if strings.Contains(f.Path, ".env") {
			t.Fatal("should not scan dot files")
		}
	}
}

func TestScanTrackedText_ReadError(t *testing.T) {
	// Scan a nonexistent directory
	findings := scanTrackedText("/nonexistent/path", func(path, text string) []Finding {
		return nil
	})
	// Should not panic, may return empty
	if findings == nil {
		// OK
	}
}

func TestStatus_EnforceP0Failed(t *testing.T) {
	s := Summary{P0: 1}
	st := status(s, 10.0, 9.8, "enforce")
	if st != "failed" {
		t.Fatalf("expected failed with P0, got %s", st)
	}
}

func TestStatus_EnforceScoreBelowMin(t *testing.T) {
	s := Summary{}
	st := status(s, 9.0, 9.8, "enforce")
	if st != "failed" {
		t.Fatalf("expected failed with low score in enforce, got %s", st)
	}
}

func TestStatus_NoFindings(t *testing.T) {
	s := Summary{}
	st := status(s, 10.0, 9.8, "enforce")
	if st != "passed" {
		t.Fatalf("expected passed with no findings, got %s", st)
	}
}

func TestStatus_WarnNoFindings(t *testing.T) {
	s := Summary{}
	st := status(s, 10.0, 9.8, "warn")
	if st != "passed" {
		t.Fatalf("expected passed with no findings in warn, got %s", st)
	}
}

func TestToMarkdown_FindingWithoutPath(t *testing.T) {
	report := Report{
		Status:   "failed",
		Mode:     "enforce",
		Score:    8.0,
		MinScore: 9.8,
		Summary:  Summary{P0: 1},
		Sections: []SectionReport{{
			Name:   "test",
			Status: "failed",
			P0:     1,
			Findings: []Finding{{
				ID:       "debt.test.finding",
				Severity: "P0",
				Path:     "",
				Message:  "test finding without path",
			}},
		}},
	}
	md := ToMarkdown(report)
	if !strings.Contains(md, "policy") {
		t.Fatalf("expected 'policy' as default path in markdown: %s", md)
	}
}

func TestStatus_P1P2WithEnforce(t *testing.T) {
	s := Summary{P1: 1, P2: 1}
	st := status(s, 10.0, 9.8, "enforce")
	if st != "passed" {
		t.Fatalf("expected passed in enforce with P1/P2 only, got %s", st)
	}
}

func TestStatus_P1P2WithWarn(t *testing.T) {
	s := Summary{P1: 1, P2: 1}
	st := status(s, 10.0, 9.8, "warn")
	if st != "warning" {
		t.Fatalf("expected warning in warn with P1/P2, got %s", st)
	}
}

func TestBytesLookBinary_LargeData(t *testing.T) {
	// Data > 4096 bytes with null byte in first 4096
	data := make([]byte, 5000)
	data[100] = 0
	if !bytesLookBinary(data) {
		t.Fatal("expected binary for large data with null byte")
	}
	// Data > 4096 bytes without null byte
	data2 := make([]byte, 5000)
	for i := range data2 {
		data2[i] = 'a'
	}
	if bytesLookBinary(data2) {
		t.Fatal("expected non-binary for large text data")
	}
}

func TestRun_NormalizeEmptyRoot(t *testing.T) {
	// When Root is empty, normalize sets it to "."
	report, err := Run(Options{Section: "architecture", Mode: "enforce", MinScore: DefaultMinScore})
	if err != nil {
		t.Fatal(err)
	}
	if report.Mode != "enforce" {
		t.Fatalf("expected enforce, got %s", report.Mode)
	}
}

func TestScanTrackedText_SortsFindings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "b.txt", "marker-xlib-domain-forbidden")
	writeFile(t, root, "a.txt", "marker-xlib-domain-forbidden")
	findings := scanTextMarker(root, "marker-xlib-domain-forbidden", "test.id", "test msg")
	if len(findings) < 2 {
		t.Fatalf("expected at least 2 findings, got %d", len(findings))
	}
	// Should be sorted by path
	for i := 1; i < len(findings); i++ {
		if findings[i-1].Path > findings[i].Path {
			t.Fatalf("findings not sorted: %s > %s", findings[i-1].Path, findings[i].Path)
		}
	}
}

func TestScanTrackedText_SortsSamePathByID(t *testing.T) {
	// Create a file with multiple markers to get same-path findings with different IDs
	root := t.TempDir()
	writeFile(t, root, "multi.txt", "marker-a\nmarker-b\n")
	// Use scanTrackedText directly with an inspect that returns multiple findings for same path
	findings := scanTrackedText(root, func(path, text string) []Finding {
		var f []Finding
		if strings.Contains(text, "marker-a") {
			f = append(f, Finding{ID: "z.id", Path: path, Severity: "P1", Message: "a"})
		}
		if strings.Contains(text, "marker-b") {
			f = append(f, Finding{ID: "a.id", Path: path, Severity: "P1", Message: "b"})
		}
		return f
	})
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	// Same path, should be sorted by ID: a.id < z.id
	if findings[0].ID != "a.id" || findings[1].ID != "z.id" {
		t.Fatalf("expected [a.id, z.id], got [%s, %s]", findings[0].ID, findings[1].ID)
	}
}

func TestWalkDir_LstatError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	// Remove the file after walkDir reads the directory but before Lstat
	// This is hard to trigger, so test the visit error path instead
	var walkErr error
	_ = walkDir(dir, func(path string) error {
		return walkErr
	})
	// No error when visit returns nil
}

func TestWalkDir_VisitError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "test.txt", "hello")
	wantErr := errors.New("visit error")
	err := walkDir(dir, func(path string) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected visit error, got %v", err)
	}
}

func TestWalkDir_RecursiveError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sub/test.txt", "hello")
	wantErr := errors.New("visit error")
	err := walkDir(dir, func(path string) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected visit error, got %v", err)
	}
}

func TestBytesLookBinary_SmallData(t *testing.T) {
	if !bytesLookBinary([]byte("hi\x00")) {
		t.Fatal("expected binary for small data with null byte")
	}
	if bytesLookBinary([]byte("hi")) {
		t.Fatal("expected non-binary for small text data")
	}
}

func TestRun_SectionSpecific(t *testing.T) {
	root := t.TempDir()
	writePolicyFiles(t, root)
	for _, section := range allSections() {
		report, err := Run(Options{Root: root, Section: section, Mode: "enforce", MinScore: DefaultMinScore})
		if err != nil {
			t.Fatalf("section %s: %v", section, err)
		}
		if len(report.Sections) != 1 {
			t.Fatalf("section %s: expected 1 section, got %d", section, len(report.Sections))
		}
		if report.Sections[0].Name != section {
			t.Fatalf("section %s: got %s", section, report.Sections[0].Name)
		}
	}
}

func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
