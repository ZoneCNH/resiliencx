package main

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestDashboardGenerate(t *testing.T) {
	overrideAuditGoalChecks(t, func(matrixPath string) []auditGoalCheck {
		if matrixPath != "custom.md" {
			t.Fatalf("matrix path = %q, want custom.md", matrixPath)
		}
		return []auditGoalCheck{
			{name: "context-check", run: func(stdout io.Writer, stderr io.Writer) int {
				return emitReport(stdout, "context-check", "passed", nil, nil)
			}},
			{name: "traceability-check", run: func(stdout io.Writer, stderr io.Writer) int {
				return emitReport(stdout, "traceability-check", "passed", nil, nil)
			}},
		}
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runDashboardGenerate([]string{"--goal-id", "GOAL-1", "--matrix", "custom.md", "--format", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var report dashboardReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("parse dashboard JSON: %v\n%s", err, stdout.String())
	}
	if report.SchemaVersion != dashboardGenerateSchemaVersion {
		t.Fatalf("schema_version = %q, want %q", report.SchemaVersion, dashboardGenerateSchemaVersion)
	}
	if report.Command != "dashboard-generate" || report.Status != "passed" {
		t.Fatalf("unexpected command/status: %#v", report)
	}
	if report.GoalID != "GOAL-1" || report.Matrix != "custom.md" {
		t.Fatalf("unexpected goal or matrix: %#v", report)
	}
	if !reflect.DeepEqual(report.Scope, []string{"goal", "req", "task", "issue", "evidence", "release"}) {
		t.Fatalf("scope = %#v", report.Scope)
	}
	if report.Mode != "local-readonly" || report.WriteEvidence {
		t.Fatalf("unexpected mode/write evidence: %#v", report)
	}
	if got, want := componentStatuses(report.Components), []string{"context-check:passed", "traceability-check:passed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("components = %#v, want %#v", got, want)
	}
	if len(report.Gaps) != 0 {
		t.Fatalf("gaps = %#v, want none", report.Gaps)
	}
	for _, needle := range []string{
		`"schema_version": "1.0"`,
		`"command": "dashboard-generate"`,
		`"goal_id": "GOAL-1"`,
		`"write_evidence": false`,
	} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("stdout missing %q:\n%s", needle, stdout.String())
		}
	}
}

func TestDashboardGenerateMarkdownReportsGaps(t *testing.T) {
	overrideAuditGoalChecks(t, func(matrixPath string) []auditGoalCheck {
		return []auditGoalCheck{
			{name: "context-check", run: func(stdout io.Writer, stderr io.Writer) int {
				return emitReport(stdout, "context-check", "passed", nil, nil)
			}},
			{name: "command-registry", run: func(stdout io.Writer, stderr io.Writer) int {
				write(stderr, "component | details\n")
				return 1
			}},
		}
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runDashboardGenerate([]string{"--goal-id", "GOAL-1", "--format", "markdown"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "ERROR: dashboard-generate found 1 gap(s)") {
		t.Fatalf("stderr missing gap count:\n%s", stderr.String())
	}
	expected := `# Goal Dashboard

| 字段 | 值 |
| --- | --- |
| command | dashboard-generate |
| status | failed |
| goal_id | GOAL-1 |
| matrix | .agent/traceability/traceability-matrix.md |
| scope | goal,req,task,issue,evidence,release |
| mode | local-readonly |
| write_evidence | false |

## Components

| component | status | summary |
| --- | --- | --- |
| context-check | passed |  |
| command-registry | failed | component \| details |

## Gaps

- command-registry: exit code 1: component \| details
`
	if stdout.String() != expected {
		t.Fatalf("markdown mismatch:\n--- got ---\n%s\n--- want ---\n%s", stdout.String(), expected)
	}
}

func TestDashboardGenerateRejectsInvalidArgs(t *testing.T) {
	for _, args := range [][]string{
		{"unexpected"},
		{"--format", "yaml"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := runDashboardGenerate(args, &stdout, &stderr)
		if code != 2 {
			t.Fatalf("runDashboardGenerate(%v) exit code = %d, want 2; stderr=%s", args, code, stderr.String())
		}
	}
}

func componentStatuses(components []dashboardComponent) []string {
	statuses := make([]string, 0, len(components))
	for _, component := range components {
		statuses = append(statuses, component.Name+":"+component.Status)
	}
	return statuses
}
