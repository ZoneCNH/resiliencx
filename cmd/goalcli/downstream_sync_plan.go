package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultDownstreamImpactReport = "release/standard-impact/latest.md"
	defaultDownstreamSyncPlan     = "release/downstream-sync/latest.md"
)

var protectedDownstreamSyncPlanOutputPaths = map[string]struct{}{
	".agent/registries/downstream-adoption-status.yaml": {},
	".agent/evidence/truth-state.yaml":                  {},
}

var downstreamImpactCategories = []string{
	"contracts",
	"context_runtime",
	"governance_registry",
	"harness",
	"repository_rules",
	"generator",
	"downstream_context",
	"evidence",
	"docs",
	"other",
}

type downstreamSyncPlan struct {
	SchemaVersion                  string                       `json:"schema_version"`
	Command                        string                       `json:"command"`
	GeneratedBy                    string                       `json:"generated_by"`
	ImpactReport                   string                       `json:"impact_report"`
	Output                         string                       `json:"output,omitempty"`
	DownstreamSyncRequired         bool                         `json:"downstream_sync_required"`
	DownstreamReleaseDecision      string                       `json:"downstream_release_decision"`
	RepositoryRulesReleaseDecision string                       `json:"repository_rules_release_decision"`
	PrimaryDownstream              string                       `json:"primary_downstream"`
	ChangedFileCount               int                          `json:"changed_file_count"`
	AdoptionClaim                  string                       `json:"adoption_claim"`
	CategoryFileCounts             map[string]int               `json:"category_file_counts"`
	Targets                        []downstreamSyncPlanTarget   `json:"targets"`
	ConsumerReview                 downstreamSyncConsumerReview `json:"consumer_review"`
	TruthSources                   []string                     `json:"truth_sources"`
	ForbiddenInterpretations       []string                     `json:"forbidden_interpretations"`
}

type downstreamSyncPlanTarget struct {
	Name        string   `json:"name"`
	ModulePath  string   `json:"module_path"`
	PackageName string   `json:"package_name"`
	Layer       string   `json:"layer"`
	Priority    string   `json:"priority"`
	Action      string   `json:"action"`
	Status      string   `json:"status"`
	Commands    []string `json:"commands,omitempty"`
}

type downstreamSyncConsumerReview struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Action string `json:"action"`
	Status string `json:"status"`
}

type downstreamImpactReport struct {
	Path                           string
	DownstreamSyncRequired         bool
	DownstreamReleaseDecision      string
	RepositoryRulesReleaseDecision string
	PrimaryDownstream              string
	ChangedFileCount               int
	CategoryFileCounts             map[string]int
}

func runDownstreamSyncPlan(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli downstream-sync-plan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	impactReportPath := flags.String("impact-report", defaultDownstreamImpactReport, "standard impact report path")
	outputPath := flags.String("output", defaultDownstreamSyncPlan, "sync plan output path, or - for stdout")
	workspaceRoot := flags.String("workspace-root", "..", "downstream workspace root used in rendered commands")
	format := flags.String("format", "markdown", "plan format: markdown or json")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		write(stderr, "ERROR: downstream-sync-plan does not accept positional args: %s\n", strings.Join(flags.Args(), ", "))
		return 2
	}
	if *format != "markdown" && *format != "json" {
		write(stderr, "ERROR: unsupported downstream-sync-plan format %q\n", *format)
		return 2
	}
	if err := validateDownstreamSyncPlanOutputPath(*outputPath, *workspaceRoot); err != nil {
		write(stderr, "ERROR: invalid downstream sync plan output: %v\n", err)
		return emitReport(stdout, "downstream-sync-plan", "failed", nil, []string{err.Error()})
	}

	impact, err := parseDownstreamImpactReport(*impactReportPath)
	if err != nil {
		gap := fmt.Sprintf("standard impact report unavailable or invalid: %v", err)
		write(stderr, "ERROR: %s; run GOWORK=off make standard-impact-check first\n", gap)
		return emitReport(stdout, "downstream-sync-plan", "failed", nil, []string{gap})
	}

	plan := buildDownstreamSyncPlan(impact, *outputPath, *workspaceRoot)
	var rendered []byte
	if *format == "json" {
		rendered, err = json.MarshalIndent(plan, "", "  ")
		if err == nil {
			rendered = append(rendered, '\n')
		}
	} else {
		rendered = []byte(renderDownstreamSyncPlanMarkdown(plan))
	}
	if err != nil {
		write(stderr, "ERROR: render downstream sync plan: %v\n", err)
		return emitReport(stdout, "downstream-sync-plan", "failed", nil, []string{err.Error()})
	}

	if *outputPath == "-" {
		write(stdout, "%s", rendered)
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		write(stderr, "ERROR: create downstream sync plan directory: %v\n", err)
		return emitReport(stdout, "downstream-sync-plan", "failed", nil, []string{err.Error()})
	}
	if err := os.WriteFile(*outputPath, rendered, 0o644); err != nil {
		write(stderr, "ERROR: write downstream sync plan: %v\n", err)
		return emitReport(stdout, "downstream-sync-plan", "failed", nil, []string{err.Error()})
	}

	requiredText := strconv.FormatBool(plan.DownstreamSyncRequired)
	return emitReport(stdout, "downstream-sync-plan", "passed", []string{
		"downstream sync plan written to " + *outputPath,
		"impact report: " + *impactReportPath,
		"downstream_sync_required=" + requiredText,
		fmt.Sprintf("target_count=%d", len(plan.Targets)),
		"adoption_claim=not_claimed",
	}, nil)
}

func validateDownstreamSyncPlanOutputPath(outputPath string, workspaceRoot string) error {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "-" {
		return nil
	}
	if outputPath == "" {
		return errors.New("output path must not be empty")
	}
	if filepath.IsAbs(outputPath) {
		return fmt.Errorf("output path must be repository-relative or -: %s", outputPath)
	}
	rawSlash := filepath.ToSlash(outputPath)
	for _, segment := range strings.Split(rawSlash, "/") {
		if segment == ".." {
			return fmt.Errorf("output path must not contain parent traversal: %s", outputPath)
		}
	}

	cleaned := filepath.Clean(outputPath)
	slash := filepath.ToSlash(cleaned)
	if slash == "." {
		return errors.New("output path must reference a file")
	}
	if _, ok := protectedDownstreamSyncPlanOutputPaths[slash]; ok {
		return fmt.Errorf("output path must not target adoption truth file: %s", slash)
	}
	if isWithinDownstreamWorkspace(slash, workspaceRoot) {
		return fmt.Errorf("output path must not be inside downstream workspace root: %s", slash)
	}
	if info, err := os.Stat(cleaned); err == nil && info.IsDir() {
		return fmt.Errorf("output path must reference a file, got directory: %s", slash)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check output path: %w", err)
	}
	return nil
}

func isWithinDownstreamWorkspace(outputSlash string, workspaceRoot string) bool {
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" || workspaceRoot == "." || filepath.IsAbs(workspaceRoot) {
		return false
	}
	rawWorkspaceSlash := filepath.ToSlash(workspaceRoot)
	for _, segment := range strings.Split(rawWorkspaceSlash, "/") {
		if segment == ".." {
			return false
		}
	}
	workspaceSlash := filepath.ToSlash(filepath.Clean(workspaceRoot))
	if workspaceSlash == "." {
		return false
	}
	return outputSlash == workspaceSlash || strings.HasPrefix(outputSlash, workspaceSlash+"/")
}

func parseDownstreamImpactReport(path string) (downstreamImpactReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return downstreamImpactReport{}, err
	}
	report := downstreamImpactReport{
		Path:               path,
		CategoryFileCounts: map[string]int{},
	}
	metadata := map[string]string{}
	currentSection := ""
	metadataLine := regexp.MustCompile(`^- ([a-zA-Z0-9_]+): ` + "`" + `([^` + "`" + `]*)` + "`" + `$`)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			continue
		}
		if currentSection == "" {
			if match := metadataLine.FindStringSubmatch(strings.TrimSpace(line)); len(match) == 3 {
				metadata[match[1]] = match[2]
			}
			continue
		}
		if !isDownstreamImpactCategory(currentSection) {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "- `") {
			report.CategoryFileCounts[currentSection]++
		}
	}

	requiredRaw, ok := metadata["downstream_sync_required"]
	if !ok {
		return downstreamImpactReport{}, errors.New("missing downstream_sync_required")
	}
	required, err := strconv.ParseBool(requiredRaw)
	if err != nil {
		return downstreamImpactReport{}, fmt.Errorf("invalid downstream_sync_required %q", requiredRaw)
	}
	report.DownstreamSyncRequired = required

	report.DownstreamReleaseDecision = metadata["downstream_release_decision"]
	if report.DownstreamReleaseDecision != "required" && report.DownstreamReleaseDecision != "not_required" {
		return downstreamImpactReport{}, fmt.Errorf("invalid downstream_release_decision %q", report.DownstreamReleaseDecision)
	}
	if report.DownstreamSyncRequired && report.DownstreamReleaseDecision != "required" {
		return downstreamImpactReport{}, errors.New("downstream_sync_required=true requires downstream_release_decision=required")
	}
	if !report.DownstreamSyncRequired && report.DownstreamReleaseDecision != "not_required" {
		return downstreamImpactReport{}, errors.New("downstream_sync_required=false requires downstream_release_decision=not_required")
	}

	report.RepositoryRulesReleaseDecision = metadata["repository_rules_release_decision"]
	if report.RepositoryRulesReleaseDecision != "audit_required" && report.RepositoryRulesReleaseDecision != "not_required" {
		return downstreamImpactReport{}, fmt.Errorf("invalid repository_rules_release_decision %q", report.RepositoryRulesReleaseDecision)
	}
	report.PrimaryDownstream = metadata["primary_downstream"]
	if report.PrimaryDownstream == "" {
		return downstreamImpactReport{}, errors.New("missing primary_downstream")
	}
	changedRaw := metadata["changed_file_count"]
	if changedRaw == "" {
		return downstreamImpactReport{}, errors.New("missing changed_file_count")
	}
	changedFileCount, err := strconv.Atoi(changedRaw)
	if err != nil {
		return downstreamImpactReport{}, fmt.Errorf("invalid changed_file_count %q", changedRaw)
	}
	report.ChangedFileCount = changedFileCount

	for _, category := range downstreamImpactCategories {
		if _, ok := report.CategoryFileCounts[category]; !ok {
			report.CategoryFileCounts[category] = 0
		}
	}
	return report, nil
}

func isDownstreamImpactCategory(section string) bool {
	for _, category := range downstreamImpactCategories {
		if section == category {
			return true
		}
	}
	return false
}

func buildDownstreamSyncPlan(impact downstreamImpactReport, outputPath string, workspaceRoot string) downstreamSyncPlan {
	plan := downstreamSyncPlan{
		SchemaVersion:                  "1.0",
		Command:                        "downstream-sync-plan",
		GeneratedBy:                    "goalcli downstream-sync-plan",
		ImpactReport:                   impact.Path,
		Output:                         outputPath,
		DownstreamSyncRequired:         impact.DownstreamSyncRequired,
		DownstreamReleaseDecision:      impact.DownstreamReleaseDecision,
		RepositoryRulesReleaseDecision: impact.RepositoryRulesReleaseDecision,
		PrimaryDownstream:              impact.PrimaryDownstream,
		ChangedFileCount:               impact.ChangedFileCount,
		AdoptionClaim:                  "not_claimed",
		CategoryFileCounts:             impact.CategoryFileCounts,
		TruthSources: []string{
			".agent/registries/downstream-adoption-status.yaml",
			".agent/evidence/truth-state.yaml",
		},
		ForbiddenInterpretations: []string{
			"registered != adopted",
			"baseline_scanned != adopted",
			"patch_only != proof_based_adoption",
			"not_run != passed",
		},
	}
	for _, target := range downstreamSyncPlanTargets() {
		if impact.DownstreamSyncRequired {
			target.Status = "blocked_pending_downstream_workspace"
			target.Action = "sync_required"
			if target.ModulePath == impact.PrimaryDownstream {
				target.Action = "primary_sync_required"
			}
			target.Commands = downstreamSyncCommands(target, workspaceRoot)
		} else {
			target.Action = "sync_not_required"
			target.Status = "not_required_by_standard_impact"
			target.Commands = nil
		}
		plan.Targets = append(plan.Targets, target)
	}
	plan.ConsumerReview = downstreamSyncConsumerReview{
		Name: "x.go",
		Role: "consumer_only",
	}
	if impact.DownstreamSyncRequired {
		plan.ConsumerReview.Action = "consumer_only_review_required"
		plan.ConsumerReview.Status = "review_pending_no_standard_write"
	} else {
		plan.ConsumerReview.Action = "consumer_only_no_write"
		plan.ConsumerReview.Status = "not_required_by_standard_impact"
	}
	return plan
}

func downstreamSyncPlanTargets() []downstreamSyncPlanTarget {
	return []downstreamSyncPlanTarget{
		{Name: "kernel", ModulePath: "github.com/ZoneCNH/kernel", PackageName: "kernel", Layer: "L0", Priority: "P0"},
		{Name: "configx", ModulePath: "github.com/ZoneCNH/configx", PackageName: "configx", Layer: "L1", Priority: "P1"},
		{Name: "observex", ModulePath: "github.com/ZoneCNH/observex", PackageName: "observex", Layer: "L1", Priority: "P1"},
		{Name: "testkitx", ModulePath: "github.com/ZoneCNH/testkitx", PackageName: "testkitx", Layer: "L1", Priority: "P1"},
		{Name: "postgresx", ModulePath: "github.com/ZoneCNH/postgresx", PackageName: "postgresx", Layer: "L2", Priority: "P2"},
		{Name: "redisx", ModulePath: "github.com/ZoneCNH/redisx", PackageName: "redisx", Layer: "L2", Priority: "P2"},
		{Name: "kafkax", ModulePath: "github.com/ZoneCNH/kafkax", PackageName: "kafkax", Layer: "L2", Priority: "P2"},
		{Name: "natsx", ModulePath: "github.com/ZoneCNH/natsx", PackageName: "natsx", Layer: "L2", Priority: "P2"},
		{Name: "taosx", ModulePath: "github.com/ZoneCNH/taosx", PackageName: "taosx", Layer: "L2", Priority: "P2"},
		{Name: "ossx", ModulePath: "github.com/ZoneCNH/ossx", PackageName: "ossx", Layer: "L2", Priority: "P2"},
		{Name: "clickhousex", ModulePath: "github.com/ZoneCNH/clickhousex", PackageName: "clickhousex", Layer: "L2", Priority: "P2"},
	}
}

func downstreamSyncCommands(target downstreamSyncPlanTarget, workspaceRoot string) []string {
	out := filepath.Join(workspaceRoot, target.Name)
	quotedOut := shellQuote(out)
	return []string{
		fmt.Sprintf("scripts/render_template.sh --module-name %s --module-path %s --package-name %s --out %s", shellQuote(target.Name), shellQuote(target.ModulePath), shellQuote(target.PackageName), quotedOut),
		fmt.Sprintf("cd %s && GOWORK=off go mod tidy", quotedOut),
		fmt.Sprintf("cd %s && GOWORK=off go test ./...", quotedOut),
		fmt.Sprintf("cd %s && GOWORK=off make contracts", quotedOut),
		fmt.Sprintf("cd %s && GOWORK=off make boundary", quotedOut),
		fmt.Sprintf("cd %s && CHECK_STATUS=passed GOWORK=off make evidence", quotedOut),
		fmt.Sprintf("cd %s && RELEASE_EVIDENCE_REQUIRE_PASSED=1 GOWORK=off make release-evidence-check", quotedOut),
	}
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("_-./:", r) {
			continue
		}
		return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
	}
	return value
}

func renderDownstreamSyncPlanMarkdown(plan downstreamSyncPlan) string {
	var builder strings.Builder
	write(&builder, "# Downstream Sync Plan\n\n")
	write(&builder, "- generated_by: `%s`\n", plan.GeneratedBy)
	write(&builder, "- impact_report: `%s`\n", plan.ImpactReport)
	write(&builder, "- downstream_sync_required: `%t`\n", plan.DownstreamSyncRequired)
	write(&builder, "- downstream_release_decision: `%s`\n", plan.DownstreamReleaseDecision)
	write(&builder, "- repository_rules_release_decision: `%s`\n", plan.RepositoryRulesReleaseDecision)
	write(&builder, "- primary_downstream: `%s`\n", plan.PrimaryDownstream)
	write(&builder, "- changed_file_count: `%d`\n", plan.ChangedFileCount)
	write(&builder, "- adoption_claim: `%s`\n\n", plan.AdoptionClaim)

	write(&builder, "## Impact Categories\n\n")
	write(&builder, "| Category | Files |\n")
	write(&builder, "| --- | ---: |\n")
	for _, category := range sortedDownstreamImpactCategories(plan.CategoryFileCounts) {
		write(&builder, "| `%s` | %d |\n", category, plan.CategoryFileCounts[category])
	}

	write(&builder, "\n## Target Plan\n\n")
	write(&builder, "| Target | Layer | Priority | Action | Status |\n")
	write(&builder, "| --- | --- | --- | --- | --- |\n")
	for _, target := range plan.Targets {
		write(&builder, "| `%s` | `%s` | `%s` | `%s` | `%s` |\n", target.Name, target.Layer, target.Priority, target.Action, target.Status)
	}
	write(&builder, "| `%s` | `%s` | `%s` | `%s` | `%s` |\n", plan.ConsumerReview.Name, plan.ConsumerReview.Role, "review", plan.ConsumerReview.Action, plan.ConsumerReview.Status)

	write(&builder, "\n## Sync Commands\n\n")
	if !plan.DownstreamSyncRequired {
		write(&builder, "No downstream write commands are generated because standard impact does not require sync.\n\n")
	} else {
		for _, target := range plan.Targets {
			write(&builder, "### %s\n\n", target.Name)
			write(&builder, "```bash\n")
			for _, command := range target.Commands {
				write(&builder, "%s\n", command)
			}
			write(&builder, "```\n\n")
		}
	}

	write(&builder, "## Consumer Review\n\n")
	write(&builder, "- `%s`: `%s` / `%s`.\n\n", plan.ConsumerReview.Name, plan.ConsumerReview.Action, plan.ConsumerReview.Status)

	write(&builder, "## Evidence Rules\n\n")
	write(&builder, "- This plan is not proof-based adoption.\n")
	write(&builder, "- Adoption truth remains `.agent/registries/downstream-adoption-status.yaml` and `.agent/evidence/truth-state.yaml`.\n")
	write(&builder, "- This command must not modify downstream repositories or adoption truth files.\n")
	write(&builder, "- Generated output is local evidence and is ignored by git.\n")
	return builder.String()
}

func sortedDownstreamImpactCategories(counts map[string]int) []string {
	categories := make([]string, 0, len(counts))
	seen := map[string]bool{}
	for _, category := range downstreamImpactCategories {
		if _, ok := counts[category]; ok {
			categories = append(categories, category)
			seen[category] = true
		}
	}
	var extras []string
	for category := range counts {
		if !seen[category] {
			extras = append(extras, category)
		}
	}
	sort.Strings(extras)
	return append(categories, extras...)
}
