package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ZoneCNH/xlib-standard/internal/debtcheck"
)

func TestDebtPatchSuggestionsWithFindings(t *testing.T) {
	report := debtcheck.Report{Sections: []debtcheck.SectionReport{
		{
			Name: "architecture",
			Findings: []debtcheck.Finding{
				{Severity: "P0", ID: "debt.arch.legacy", Path: "cmd/foo.go", Message: "legacy import"},
				{Severity: "P1", ID: "debt.arch.drift", Path: "cmd/bar.go", Message: "drift"},
			},
		},
	}}
	suggestions := debtPatchSuggestions(report)
	if len(suggestions) != 2 {
		t.Fatalf("suggestions = %d, want 2", len(suggestions))
	}
}

func TestDebtPatchSuggestionsWithNoFindings(t *testing.T) {
	report := debtcheck.Report{}
	suggestions := debtPatchSuggestions(report)
	if len(suggestions) != 1 || suggestions[0] != "no patch suggestions; current debt report has no findings" {
		t.Fatalf("suggestions = %v, want no findings message", suggestions)
	}
}

func TestDebtPatchSuggestionsCapsAt20(t *testing.T) {
	var findings []debtcheck.Finding
	for i := 0; i < 25; i++ {
		findings = append(findings, debtcheck.Finding{
			Severity: "P1", ID: "debt.test." + string(rune('A'+i)), Path: "file.go", Message: "msg",
		})
	}
	section := debtcheck.SectionReport{Name: "section", Findings: findings}
	report := debtcheck.Report{Sections: []debtcheck.SectionReport{section}}
	suggestions := debtPatchSuggestions(report)
	if len(suggestions) != 20 {
		t.Fatalf("suggestions = %d, want 20 (cap)", len(suggestions))
	}
}

func TestDebtTrendDetailsWithPriorReport(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "release", "debt")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	previous := `{"status":"passed","score":9.90,"min_score":9.80}`
	if err := os.WriteFile(filepath.Join(dir, "latest.json"), []byte(previous), 0o644); err != nil {
		t.Fatal(err)
	}
	origDir, _ := os.Getwd()
	_ = os.Chdir(root)
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	report := debtcheck.Report{}
	report.Status = "passed"
	report.Score = 9.80
	details := debtTrendDetails(report)
	if len(details) != 4 {
		t.Fatalf("details = %d, want 4; %v", len(details), details)
	}
}

func TestRunDebtEvidenceRejectsArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := runDebtEvidence([]string{"extra"}, &stdout, &stderr)
	if got != 2 {
		t.Fatalf("exit = %d, want 2", got)
	}
}

func TestVerifyArtifactExistsDirectoryGlobMissing(t *testing.T) {
	err := verifyArtifactExists("/nonexistent/dir/*")
	if err == nil {
		t.Fatal("expected error for missing directory glob")
	}
}

func TestVerifyArtifactExistsGlobNoMatch(t *testing.T) {
	dir := t.TempDir()
	err := verifyArtifactExists(filepath.Join(dir, "*.nonexistent_extension_xyz"))
	if err == nil {
		t.Fatal("expected error for glob with no matches")
	}
}

func TestVerifyArtifactExistsMissingFile(t *testing.T) {
	err := verifyArtifactExists("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for missing artifact")
	}
}

func TestEnvDefault(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		t.Setenv("TEST_ENV_DEFAULT_KEY", "from_env")
		if got := envDefault("TEST_ENV_DEFAULT_KEY", "fallback"); got != "from_env" {
			t.Fatalf("got %q, want from_env", got)
		}
	})
	t.Run("returns fallback when unset", func(t *testing.T) {
		_ = os.Unsetenv("TEST_ENV_DEFAULT_KEY_MISSING")
		if got := envDefault("TEST_ENV_DEFAULT_KEY_MISSING", "fallback"); got != "fallback" {
			t.Fatalf("got %q, want fallback", got)
		}
	})
}

func TestFallback(t *testing.T) {
	if got := fallback("", "default"); got != "default" {
		t.Fatalf("fallback empty = %q, want default", got)
	}
	if got := fallback("value", "default"); got != "value" {
		t.Fatalf("fallback value = %q, want value", got)
	}
}

func TestFlagProvided(t *testing.T) {
	tests := []struct {
		args []string
		name string
		want bool
	}{
		{[]string{"--repo", "kernel"}, "repo", true},
		{[]string{"--repo=kernel"}, "repo", true},
		{[]string{"--mode", "patch-only"}, "repo", false},
		{nil, "repo", false},
	}
	for _, tt := range tests {
		if got := flagProvided(tt.args, tt.name); got != tt.want {
			t.Errorf("flagProvided(%v, %q) = %v, want %v", tt.args, tt.name, got, tt.want)
		}
	}
}

func TestTrimYAMLScalar(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"value # comment", "value"},
		{`"quoted"`, "quoted"},
		{`'single'`, "single"},
		{"plain", "plain"},
	}
	for _, tt := range tests {
		if got := trimYAMLScalar(tt.input); got != tt.want {
			t.Errorf("trimYAMLScalar(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBlockHasYAMLListItem(t *testing.T) {
	block := "evidence:\n  - go test ./cmd/goalcli\n"
	if !blockHasYAMLListItem(block, "evidence") {
		t.Fatal("expected evidence list item")
	}
	empty := "evidence:\nstatus: implemented\n"
	if blockHasYAMLListItem(empty, "evidence") {
		t.Fatal("expected no evidence list item for empty list")
	}
}

func TestValidContextProfileName(t *testing.T) {
	valid := []string{"lite", "standard", "full", "release"}
	for _, p := range valid {
		if !validContextProfileName(p) {
			t.Errorf("validContextProfileName(%q) = false", p)
		}
	}
	if validContextProfileName("unknown") {
		t.Error("validContextProfileName(\"unknown\") = true")
	}
}

func TestNormalizeContextProfile(t *testing.T) {
	if got := normalizeContextProfile("fast"); got != "lite" {
		t.Fatalf("normalizeContextProfile(\"fast\") = %q, want lite", got)
	}
	if got := normalizeContextProfile("standard"); got != "standard" {
		t.Fatalf("normalizeContextProfile(\"standard\") = %q, want standard", got)
	}
}

func TestAppendMakefileDuplicateGaps(t *testing.T) {
	makefile := "target-a:\n\t@true\ntarget-a:\n\t@true\n"
	var gaps []string
	appendMakefileDuplicateGaps(makefile, []string{"target-a"}, &gaps)
	if len(gaps) == 0 {
		t.Fatal("expected duplicate gap")
	}
}

func TestAppendMakefileTargetForbiddenReferenceGapsMissingTarget(t *testing.T) {
	var gaps []string
	appendMakefileTargetForbiddenReferenceGaps("other:\n\t@true\n", "context-release", []string{"release-final-check"}, &gaps)
	if len(gaps) == 0 {
		t.Fatal("expected missing target block gap")
	}
}

func TestAppendReleaseFinalDelegationGapsMissingBlock(t *testing.T) {
	var gaps []string
	appendReleaseFinalDelegationGaps("other:\n\t@true\n", &gaps)
	if len(gaps) == 0 {
		t.Fatal("expected missing target block gap")
	}
}

func TestRunExternalSequenceStopsOnError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := runExternalSequence(&bytes.Buffer{}, &stdout, &stderr,
		externalCommand{name: "false"},
		externalCommand{name: "echo", args: []string{"should not run"}},
	)
	if got == 0 {
		t.Fatal("expected non-zero exit from failing sequence")
	}
}

func TestFlagValue(t *testing.T) {
	tests := []struct {
		args []string
		name string
		def  string
		want string
	}{
		{[]string{"--repo", "kernel"}, "repo", "", "kernel"},
		{[]string{"--repo=kernel"}, "repo", "", "kernel"},
		{[]string{"--other", "val"}, "repo", "default", "default"},
	}
	for _, tt := range tests {
		if got := flagValue(tt.args, tt.name, tt.def); got != tt.want {
			t.Errorf("flagValue(%v, %q, %q) = %q, want %q", tt.args, tt.name, tt.def, got, tt.want)
		}
	}
}
