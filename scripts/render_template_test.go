package scripts_test

import (
	"os"
	"strings"
	"testing"
)

func TestRenderTemplateExcludesGeneratedDebtArtifacts(t *testing.T) {
	contents, err := os.ReadFile("render_template.sh")
	if err != nil {
		t.Fatalf("read render_template.sh: %v", err)
	}

	script := string(contents)
	for _, exclude := range []string{
		"--exclude='./release/debt/latest.json'",
		"--exclude='./release/debt/latest.md'",
		"--exclude='./release/debt/latest.json.sha256'",
	} {
		if !strings.Contains(script, exclude) {
			t.Fatalf("render_template.sh missing generated debt artifact exclude %q", exclude)
		}
	}
}

func TestRenderTemplateResetsGeneratedGoalEvidence(t *testing.T) {
	contents, err := os.ReadFile("render_template.sh")
	if err != nil {
		t.Fatalf("read render_template.sh: %v", err)
	}

	script := string(contents)
	if !strings.Contains(script, "--exclude='./release/evidence/goalcli'") {
		t.Fatalf("render_template.sh missing generated goal evidence pack exclude")
	}
	if !strings.Contains(script, "ledger_path=\"$out_dir/.agent/evidence/ledger.jsonl\"") ||
		!strings.Contains(script, "grep -v 'RESILIENCX'") {
		t.Fatalf("render_template.sh must preserve portable ledger entries while removing source goal evidence")
	}
}

func TestRunIntegrationClearsSourceGoalIDForDownstreamSmoke(t *testing.T) {
	contents, err := os.ReadFile("run_integration.sh")
	if err != nil {
		t.Fatalf("read run_integration.sh: %v", err)
	}

	script := string(contents)
	for _, command := range []string{
		"env -u GOAL_ID GOWORK=off go test ./...",
		"env -u GOAL_ID GOWORK=off make contracts",
		"env -u GOAL_ID RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check",
	} {
		if !strings.Contains(script, command) {
			t.Fatalf("run_integration.sh must clear source GOAL_ID for downstream command %q", command)
		}
	}
}
