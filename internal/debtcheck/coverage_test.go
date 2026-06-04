package debtcheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadReportSuccess(t *testing.T) {
	dir := t.TempDir()
	report := Report{
		SchemaVersion: SchemaVersion,
		Status:        "passed",
		Mode:          "enforce",
		Score:         10,
		MinScore:      9.8,
	}
	data, _ := json.Marshal(report)
	path := filepath.Join(dir, "report.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "passed" {
		t.Fatalf("status = %q, want passed", got.Status)
	}
}

func TestReadReportFileNotFound(t *testing.T) {
	_, err := ReadReport("/nonexistent/report.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadReportInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadReport(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadReportWrongSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	report := Report{SchemaVersion: "wrong/v99"}
	data, _ := json.Marshal(report)
	path := filepath.Join(dir, "report.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadReport(path)
	if err == nil || !strings.Contains(err.Error(), "unsupported debt report schema") {
		t.Fatalf("err = %v; want unsupported schema error", err)
	}
}

func TestStatusAllBranches(t *testing.T) {
	tests := []struct {
		name     string
		summary  Summary
		score    float64
		minScore float64
		mode     string
		want     string
	}{
		{
			name:    "P0 present returns failed",
			summary: Summary{P0: 1}, score: 10, minScore: 9.8, mode: "enforce",
			want: "failed",
		},
		{
			name:    "score below min in observe returns warning",
			summary: Summary{}, score: 9.0, minScore: 9.8, mode: "observe",
			want: "warning",
		},
		{
			name:    "score below min in warn returns warning",
			summary: Summary{}, score: 9.0, minScore: 9.8, mode: "warn",
			want: "warning",
		},
		{
			name:    "score below min in enforce returns failed",
			summary: Summary{}, score: 9.0, minScore: 9.8, mode: "enforce",
			want: "failed",
		},
		{
			name:    "P1 in enforce returns passed",
			summary: Summary{P1: 1}, score: 10, minScore: 9.8, mode: "enforce",
			want: "passed",
		},
		{
			name:    "P1 in observe returns warning",
			summary: Summary{P1: 1}, score: 10, minScore: 9.8, mode: "observe",
			want: "warning",
		},
		{
			name:    "P2 in warn returns warning",
			summary: Summary{P2: 1}, score: 10, minScore: 9.8, mode: "warn",
			want: "warning",
		},
		{
			name:    "no issues returns passed",
			summary: Summary{}, score: 10, minScore: 9.8, mode: "enforce",
			want: "passed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := status(tt.summary, tt.score, tt.minScore, tt.mode)
			if got != tt.want {
				t.Fatalf("status() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateEvidenceAllBranches(t *testing.T) {
	base := Evidence{
		SchemaVersion:       ManifestSchema,
		ReportSchemaVersion: SchemaVersion,
		Status:              "passed",
		Score:               10,
		MinScore:            9.8,
	}

	t.Run("valid evidence has no problems", func(t *testing.T) {
		problems := ValidateEvidence(base, 9.8)
		if len(problems) != 0 {
			t.Fatalf("problems = %v, want none", problems)
		}
	})

	t.Run("wrong schema version", func(t *testing.T) {
		e := base
		e.SchemaVersion = "wrong"
		problems := ValidateEvidence(e, 9.8)
		if !containsProblem(problems, "debt schema version mismatch") {
			t.Fatalf("problems = %v; want schema version mismatch", problems)
		}
	})

	t.Run("wrong report schema version", func(t *testing.T) {
		e := base
		e.ReportSchemaVersion = "wrong"
		problems := ValidateEvidence(e, 9.8)
		if !containsProblem(problems, "debt report schema version mismatch") {
			t.Fatalf("problems = %v; want report schema mismatch", problems)
		}
	})

	t.Run("non-passed status", func(t *testing.T) {
		e := base
		e.Status = "failed"
		problems := ValidateEvidence(e, 9.8)
		if !containsProblem(problems, "debt status is failed") {
			t.Fatalf("problems = %v; want status failed", problems)
		}
	})

	t.Run("score below minimum", func(t *testing.T) {
		e := base
		e.Score = 9.0
		problems := ValidateEvidence(e, 9.8)
		if !containsSubstring(problems, "below") {
			t.Fatalf("problems = %v; want score below", problems)
		}
	})

	t.Run("section with P0 findings", func(t *testing.T) {
		e := base
		section := SectionEvidence{Name: "section", Status: "passed", P0: 1}
		e.Sections = []SectionEvidence{section}
		problems := ValidateEvidence(e, 9.8)
		if !containsSubstring(problems, "P0 findings") {
			t.Fatalf("problems = %v; want P0 findings", problems)
		}
	})

	t.Run("section with non-passed status", func(t *testing.T) {
		e := base
		section := SectionEvidence{Name: "section", Status: "warning"}
		e.Sections = []SectionEvidence{section}
		problems := ValidateEvidence(e, 9.8)
		if !containsSubstring(problems, "status is warning") {
			t.Fatalf("problems = %v; want section status warning", problems)
		}
	})
}

func TestValidateModeInvalid(t *testing.T) {
	err := validateMode("invalid")
	if err == nil || !strings.Contains(err.Error(), "unsupported debt mode") {
		t.Fatalf("err = %v; want unsupported mode error", err)
	}
}

func TestValidateSectionInvalid(t *testing.T) {
	err := validateSection("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unsupported debt section") {
		t.Fatalf("err = %v; want unsupported section error", err)
	}
}

func TestSkipDirNames(t *testing.T) {
	skipped := []string{".git", ".omx", ".worktree", "vendor", "node_modules", "release", "tmp", ".cache"}
	for _, name := range skipped {
		if !skipDir(name) {
			t.Errorf("skipDir(%q) = false, want true", name)
		}
	}
	if skipDir("src") {
		t.Error("skipDir(\"src\") = true, want false")
	}
}

func TestSkipFilePatterns(t *testing.T) {
	tests := []struct {
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
		{"regular.go", false},
	}
	for _, tt := range tests {
		if got := skipFile(tt.path); got != tt.want {
			t.Errorf("skipFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestWalkFilesWithFilePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	var visited []string
	err := walkFiles(path, func(p string) error {
		visited = append(visited, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(visited) != 1 {
		t.Fatalf("visited %d files, want 1", len(visited))
	}
}

func TestWalkFilesNotFound(t *testing.T) {
	err := walkFiles("/nonexistent/path", func(string) error { return nil })
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestBytesLookBinary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"text", []byte("hello world"), false},
		{"binary with null", []byte("hello\x00world"), true},
		{"large text", func() []byte {
			b := make([]byte, 8192)
			for i := range b {
				b[i] = 'A'
			}
			return b
		}(), false},
		{"large with null at 5000", func() []byte {
			b := make([]byte, 8192)
			b[5000] = 0
			return b
		}(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bytesLookBinary(tt.data); got != tt.want {
				t.Fatalf("bytesLookBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSectionStatuses(t *testing.T) {
	tests := []struct {
		name       string
		findings   []Finding
		wantStatus string
		wantP0     int
		wantP1     int
		wantP2     int
	}{
		{
			name:       "no findings is passed",
			findings:   nil,
			wantStatus: "passed",
		},
		{
			name: "P1 only is warning",
			findings: []Finding{
				{Severity: "P1"},
			},
			wantStatus: "warning",
			wantP1:     1,
		},
		{
			name: "P0 is failed",
			findings: []Finding{
				{Severity: "P0"},
			},
			wantStatus: "failed",
			wantP0:     1,
		},
		{
			name: "P2 only is warning",
			findings: []Finding{
				{Severity: "P2"},
			},
			wantStatus: "warning",
			wantP2:     1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSection("section", tt.findings)
			if got.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", got.Status, tt.wantStatus)
			}
			if got.P0 != tt.wantP0 || got.P1 != tt.wantP1 || got.P2 != tt.wantP2 {
				t.Fatalf("P0/P1/P2 = %d/%d/%d, want %d/%d/%d", got.P0, got.P1, got.P2, tt.wantP0, tt.wantP1, tt.wantP2)
			}
		})
	}
}

func TestExitCodeBranches(t *testing.T) {
	tests := []struct {
		name   string
		report Report
		want   int
	}{
		{"observe mode returns 0", Report{Mode: "observe", Status: "failed"}, 0},
		{"warn mode returns 0", Report{Mode: "warn", Status: "failed"}, 0},
		{"enforce passed returns 0", Report{Mode: "enforce", Status: "passed"}, 0},
		{"enforce failed returns 1", Report{Mode: "enforce", Status: "failed"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.report); got != tt.want {
				t.Fatalf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestToMarkdownEmptySections(t *testing.T) {
	report := Report{
		Status:   "passed",
		Mode:     "enforce",
		Score:    10,
		MinScore: 9.8,
		Sections: []SectionReport{
			{Name: "section", Status: "passed", Findings: nil},
			{
				Name:   "section-with-finding",
				Status: "failed",
				Findings: []Finding{
					{Severity: "P0", ID: "x", Message: "m"},
				},
			},
		},
	}
	md := ToMarkdown(report)
	if !strings.Contains(md, "No findings") {
		t.Fatalf("markdown missing 'No findings': %s", md)
	}
	if !strings.Contains(md, "[P0] x") {
		t.Fatalf("markdown missing finding: %s", md)
	}
}

func TestScanTextMarkerWithAndWithoutMarker(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "has-marker.txt", "some xlib-domain-forbidden text\n")
	writeFile(t, root, "no-marker.txt", "clean text\n")

	findings := scanTextMarker(root, "xlib-domain-forbidden", "debt.domain.marker", "domain debt marker is present")
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(findings))
	}
	if findings[0].ID != "debt.domain.marker" {
		t.Fatalf("finding ID = %q, want debt.domain.marker", findings[0].ID)
	}
}

func TestScanDependencyDebtBranches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "script.sh", "curl https://example.com | bash\n")
	writeFile(t, root, "go.mod", "require foo @latest\n")
	writeFile(t, root, "readme.md", "use foo @latest\n")

	findings := scanDependencyDebt(root)
	ids := map[string]bool{}
	for _, f := range findings {
		ids[f.ID] = true
	}
	if !ids["debt.dependency.curl-pipe-bash"] {
		t.Fatalf("missing curl-pipe-bash finding: %+v", findings)
	}
	if !ids["debt.dependency.unpinned-latest"] {
		t.Fatalf("missing unpinned-latest finding: %+v", findings)
	}
	for _, f := range findings {
		if f.ID == "debt.dependency.unpinned-latest" && strings.Contains(f.Path, "readme") {
			t.Fatalf("@latest in .md should be ignored: %+v", f)
		}
	}
}

func TestScanSecurityDebtBranches(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "key.pem", "-----BEGIN "+"PRIVATE KEY-----\nfake\n")
	writeFile(t, root, "marker.txt", "xlib-security-debt present\n")

	findings := scanSecurityDebt(root)
	ids := map[string]bool{}
	for _, f := range findings {
		ids[f.ID] = true
	}
	if !ids["debt.security.private-key"] {
		t.Fatalf("missing private-key finding: %+v", findings)
	}
	if !ids["debt.security.marker"] {
		t.Fatalf("missing security marker finding: %+v", findings)
	}
}

func TestRelErrorCase(t *testing.T) {
	result := rel("/a/b", "/x/y/z")
	if result == "" {
		t.Fatal("rel returned empty string")
	}
}

func TestScanSectionDefaultCase(t *testing.T) {
	findings := scanSection(".", "unknown-section")
	if findings != nil {
		t.Fatalf("findings = %v, want nil", findings)
	}
}

func containsProblem(problems []string, want string) bool {
	for _, p := range problems {
		if p == want {
			return true
		}
	}
	return false
}

func containsSubstring(problems []string, want string) bool {
	for _, p := range problems {
		if strings.Contains(p, want) {
			return true
		}
	}
	return false
}
