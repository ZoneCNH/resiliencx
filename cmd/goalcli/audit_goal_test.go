package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func overrideAuditGoalChecks(t *testing.T, checks func(matrixPath string) []auditGoalCheck) {
	t.Helper()
	original := newAuditGoalChecks
	newAuditGoalChecks = checks
	t.Cleanup(func() {
		newAuditGoalChecks = original
	})
}

func TestAuditGoalPassesWhenAllComponentsPass(t *testing.T) {
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
	code := runAuditGoal([]string{"--goal-id", "GOAL-1", "--matrix", "custom.md"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	for _, needle := range []string{
		`"command": "audit-goal"`,
		`"status": "passed"`,
		"context-check: passed",
		"traceability-check: passed",
		"write_evidence=false",
		"goal_id=GOAL-1",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("stdout missing %q:\n%s", needle, out)
		}
	}
}

func TestAuditGoalReportsComponentFailures(t *testing.T) {
	overrideAuditGoalChecks(t, func(matrixPath string) []auditGoalCheck {
		return []auditGoalCheck{
			{name: "context-check", run: func(stdout io.Writer, stderr io.Writer) int {
				return emitReport(stdout, "context-check", "passed", nil, nil)
			}},
			{name: "traceability-check", run: func(stdout io.Writer, stderr io.Writer) int {
				return emitReport(stdout, "traceability-check", "failed", nil, []string{"REQ-1 missing evidence"})
			}},
		}
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runAuditGoal(nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "ERROR: audit-goal found 1 gap(s)") {
		t.Fatalf("stderr missing gap count:\n%s", stderr.String())
	}
	out := stdout.String()
	for _, needle := range []string{
		`"status": "failed"`,
		"traceability-check: exit code 1",
		"REQ-1 missing evidence",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("stdout missing %q:\n%s", needle, out)
		}
	}
}

func TestAuditGoalRejectsUnexpectedArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runAuditGoal([]string{"unexpected"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "accepts no positional arguments") {
		t.Fatalf("stderr missing positional argument error:\n%s", stderr.String())
	}
}

func TestAuditGoalDefaultChecksCoverGoalLifecycle(t *testing.T) {
	names := map[string]bool{}
	for _, check := range auditGoalDefaultChecks("matrix.md") {
		names[check.name] = true
	}
	for _, name := range []string{
		"context-check",
		"spec-check",
		"design-check",
		"task-check",
		"evidence-check",
		"cli-contract",
		"issue-registry",
		"command-registry",
		"makefile-baseline",
		"traceability-check",
		"goal-acceptance:dry-run",
		"goal-delivery:dry-run",
		"goal-handover:dry-run",
		"goal-downstream-adoption:dry-run",
		"goal-certify:dry-run",
		"goal-runtime-final:dry-run",
	} {
		if !names[name] {
			t.Fatalf("default checks missing %s", name)
		}
	}
}
