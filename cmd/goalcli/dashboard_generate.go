package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

const dashboardGenerateSchemaVersion = "1.0"

type dashboardComponent struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
}

type dashboardReport struct {
	SchemaVersion string               `json:"schema_version"`
	Command       string               `json:"command"`
	Status        string               `json:"status"`
	GoalID        string               `json:"goal_id,omitempty"`
	Matrix        string               `json:"matrix"`
	Scope         []string             `json:"scope"`
	Mode          string               `json:"mode"`
	WriteEvidence bool                 `json:"write_evidence"`
	Components    []dashboardComponent `json:"components"`
	Gaps          []string             `json:"gaps,omitempty"`
}

func runDashboardGenerate(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli dashboard-generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	goalID := flags.String("goal-id", "", "optional goal id to annotate the dashboard")
	matrixPath := flags.String("matrix", traceabilityMatrixPath, "path to traceability matrix markdown")
	format := flags.String("format", "json", "dashboard output format: json or markdown")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		write(stderr, "ERROR: dashboard-generate invalid arguments: %v\n", err)
		return 2
	}
	if flags.NArg() != 0 {
		write(stderr, "ERROR: dashboard-generate accepts no positional arguments\n")
		return 2
	}
	if *format != "json" && *format != "markdown" {
		write(stderr, "ERROR: dashboard-generate invalid format %q; want json or markdown\n", *format)
		return 2
	}

	report := buildDashboardReport(*goalID, *matrixPath)
	if len(report.Gaps) > 0 {
		write(stderr, "ERROR: dashboard-generate found %d gap(s)\n", len(report.Gaps))
	}
	if *format == "markdown" {
		write(stdout, "%s", renderDashboardMarkdown(report))
	} else {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			write(stderr, "ERROR: dashboard-generate failed to marshal JSON: %v\n", err)
			return 1
		}
		write(stdout, "%s\n", data)
	}
	if report.Status == "passed" {
		return 0
	}
	return 1
}

func buildDashboardReport(goalID, matrixPath string) dashboardReport {
	report := dashboardReport{
		SchemaVersion: dashboardGenerateSchemaVersion,
		Command:       "dashboard-generate",
		Status:        "passed",
		GoalID:        goalID,
		Matrix:        matrixPath,
		Scope:         []string{"goal", "req", "task", "issue", "evidence", "release"},
		Mode:          "local-readonly",
		WriteEvidence: false,
	}
	for _, check := range newAuditGoalChecks(matrixPath) {
		var componentStdout bytes.Buffer
		var componentStderr bytes.Buffer
		code := check.run(&componentStdout, &componentStderr)
		component := dashboardComponent{Name: check.name, Status: "passed"}
		if code != 0 {
			component.Status = "failed"
			component.Summary = auditGoalComponentSummary(componentStdout.String(), componentStderr.String())
			if component.Summary == "" {
				component.Summary = "no component output"
			}
			report.Gaps = append(report.Gaps, fmt.Sprintf("%s: exit code %d: %s", check.name, code, component.Summary))
		}
		report.Components = append(report.Components, component)
	}
	if len(report.Gaps) > 0 {
		report.Status = "failed"
	}
	return report
}

func renderDashboardMarkdown(report dashboardReport) string {
	var b strings.Builder
	b.WriteString("# Goal Dashboard\n\n")
	b.WriteString("| 字段 | 值 |\n")
	b.WriteString("| --- | --- |\n")
	b.WriteString("| command | ")
	b.WriteString(markdownCell(report.Command))
	b.WriteString(" |\n")
	b.WriteString("| status | ")
	b.WriteString(markdownCell(report.Status))
	b.WriteString(" |\n")
	if report.GoalID != "" {
		b.WriteString("| goal_id | ")
		b.WriteString(markdownCell(report.GoalID))
		b.WriteString(" |\n")
	}
	b.WriteString("| matrix | ")
	b.WriteString(markdownCell(report.Matrix))
	b.WriteString(" |\n")
	b.WriteString("| scope | ")
	b.WriteString(markdownCell(strings.Join(report.Scope, ",")))
	b.WriteString(" |\n")
	b.WriteString("| mode | ")
	b.WriteString(markdownCell(report.Mode))
	b.WriteString(" |\n")
	b.WriteString("| write_evidence | ")
	b.WriteString(fmt.Sprintf("%t", report.WriteEvidence))
	b.WriteString(" |\n\n")
	b.WriteString("## Components\n\n")
	b.WriteString("| component | status | summary |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, component := range report.Components {
		b.WriteString("| ")
		b.WriteString(markdownCell(component.Name))
		b.WriteString(" | ")
		b.WriteString(markdownCell(component.Status))
		b.WriteString(" | ")
		b.WriteString(markdownCell(component.Summary))
		b.WriteString(" |\n")
	}
	b.WriteString("\n## Gaps\n\n")
	if len(report.Gaps) == 0 {
		b.WriteString("- None\n")
		return b.String()
	}
	for _, gap := range report.Gaps {
		b.WriteString("- ")
		b.WriteString(markdownCell(gap))
		b.WriteString("\n")
	}
	return b.String()
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}
