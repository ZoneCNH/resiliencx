package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

type auditGoalCheck struct {
	name string
	run  func(stdout io.Writer, stderr io.Writer) int
}

var newAuditGoalChecks = auditGoalDefaultChecks

func runAuditGoal(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli audit-goal", flag.ContinueOnError)
	flags.SetOutput(stderr)
	goalID := flags.String("goal-id", "", "optional goal id to annotate the audit report")
	matrixPath := flags.String("matrix", traceabilityMatrixPath, "path to traceability matrix markdown")
	flags.Bool("json", true, "emit JSON gate report (default true)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		write(stderr, "ERROR: audit-goal invalid arguments: %v\n", err)
		return 2
	}
	if flags.NArg() != 0 {
		write(stderr, "ERROR: audit-goal accepts no positional arguments\n")
		return 2
	}

	details := []string{
		"scope=goal,req,task,issue,evidence,release",
		"mode=local-readonly",
		"write_evidence=false",
	}
	if *goalID != "" {
		details = append(details, "goal_id="+*goalID)
	}
	var gaps []string
	for _, check := range newAuditGoalChecks(*matrixPath) {
		var componentStdout bytes.Buffer
		var componentStderr bytes.Buffer
		code := check.run(&componentStdout, &componentStderr)
		if code == 0 {
			details = append(details, check.name+": passed")
			continue
		}
		summary := auditGoalComponentSummary(componentStdout.String(), componentStderr.String())
		if summary == "" {
			summary = "no component output"
		}
		gaps = append(gaps, fmt.Sprintf("%s: exit code %d: %s", check.name, code, summary))
	}
	if len(gaps) > 0 {
		write(stderr, "ERROR: audit-goal found %d gap(s)\n", len(gaps))
		return emitReport(stdout, "audit-goal", "failed", details, gaps)
	}
	return emitReport(stdout, "audit-goal", "passed", details, nil)
}

func auditGoalDefaultChecks(matrixPath string) []auditGoalCheck {
	checks := []auditGoalCheck{
		{name: "context-check", run: func(stdout io.Writer, stderr io.Writer) int { return runContextCheck(nil, stdout, stderr) }},
		{name: "spec-check", run: func(stdout io.Writer, stderr io.Writer) int { return runSpecCheck(nil, stdout, stderr) }},
		{name: "design-check", run: func(stdout io.Writer, stderr io.Writer) int { return runDesignCheck(nil, stdout, stderr) }},
		{name: "task-check", run: func(stdout io.Writer, stderr io.Writer) int { return runTaskCheck(nil, stdout, stderr) }},
		{name: "evidence-check", run: func(stdout io.Writer, stderr io.Writer) int { return runEvidenceCheck(nil, stdout, stderr) }},
		{name: "cli-contract", run: func(stdout io.Writer, stderr io.Writer) int { return runCLIContract(nil, stdout, stderr) }},
		{name: "issue-registry", run: func(stdout io.Writer, stderr io.Writer) int { return runIssueRegistry(nil, stdout, stderr) }},
		{name: "command-registry", run: func(stdout io.Writer, stderr io.Writer) int { return runCommandRegistry(nil, stdout, stderr) }},
		{name: "makefile-baseline", run: func(stdout io.Writer, stderr io.Writer) int { return runMakefileBaseline(nil, stdout, stderr) }},
		{name: "traceability-check", run: func(stdout io.Writer, stderr io.Writer) int {
			return runTraceabilityCheck([]string{"--matrix", matrixPath}, stdout, stderr)
		}},
	}
	for _, command := range []string{
		"goal-acceptance",
		"goal-delivery",
		"goal-handover",
		"goal-downstream-adoption",
		"goal-certify",
		"goal-runtime-final",
	} {
		checks = append(checks, auditGoalRuntimeDryRunCheck(command))
	}
	return checks
}

func auditGoalRuntimeDryRunCheck(command string) auditGoalCheck {
	return auditGoalCheck{
		name: command + ":dry-run",
		run: func(stdout io.Writer, stderr io.Writer) int {
			return runGoalRuntimeCommand(command, []string{"--dry-run", "--verify"}, stdout, stderr)
		},
	}
}

func auditGoalComponentSummary(stdout string, stderr string) string {
	summary := strings.Join(strings.Fields(stdout), " ")
	if summary == "" {
		summary = strings.Join(strings.Fields(stderr), " ")
	}
	if len(summary) > 300 {
		return summary[:300] + "..."
	}
	return summary
}
