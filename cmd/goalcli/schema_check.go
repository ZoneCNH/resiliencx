package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultSchemaCheckReportPath = "reports/schema-check.json"

type schemaCheckReport struct {
	SchemaVersion string            `json:"schema_version"`
	Command       string            `json:"command"`
	Status        string            `json:"status"`
	CheckedAt     string            `json:"checked_at"`
	ReportPath    string            `json:"report_path,omitempty"`
	Checks        []schemaCheckItem `json:"checks"`
	Gaps          []string          `json:"gaps,omitempty"`
}

type schemaCheckItem struct {
	Name     string   `json:"name"`
	Artifact string   `json:"artifact"`
	Schema   string   `json:"schema,omitempty"`
	Status   string   `json:"status"`
	Details  []string `json:"details,omitempty"`
	Gaps     []string `json:"gaps,omitempty"`
}

type jsonSchema struct {
	Schema               string                `json:"$schema"`
	Title                string                `json:"title"`
	Type                 schemaType            `json:"type"`
	Required             []string              `json:"required"`
	Properties           map[string]jsonSchema `json:"properties"`
	Items                *jsonSchema           `json:"items"`
	Enum                 []any                 `json:"enum"`
	Const                any                   `json:"const"`
	Pattern              string                `json:"pattern"`
	MinLength            *int                  `json:"minLength"`
	MinItems             *int                  `json:"minItems"`
	Minimum              *float64              `json:"minimum"`
	AdditionalProperties any                   `json:"additionalProperties"`
}

type schemaType []string

func (t *schemaType) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*t = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*t = many
	return nil
}

func runSchemaCommand(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "validate" {
		write(stderr, "ERROR: schema usage: goalcli schema validate --all|--fixture <dir> [--report <path>] [--json]\n")
		return 2
	}
	return runSchemaValidate(args[1:], stdout, stderr)
}

func runSchemaCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	return runSchemaValidate(withSchemaCheckDefaultAll(args), stdout, stderr)
}

func withSchemaCheckDefaultAll(args []string) []string {
	for _, arg := range args {
		if arg == "--all" || arg == "--fixture" || strings.HasPrefix(arg, "--fixture=") {
			return args
		}
	}
	return append([]string{"--all"}, args...)
}

func runSchemaValidate(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("goalcli schema validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	all := flags.Bool("all", false, "validate repo schema-bearing artifacts")
	fixture := flags.String("fixture", "", "validate a fixture directory containing schemas/, valid/, and invalid/")
	reportPath := flags.String("report", defaultSchemaCheckReportPath, "write schema-check report")
	flags.Bool("json", false, "emit JSON report")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		write(stderr, "ERROR: schema validate invalid arguments: unexpected positional argument %q\n", flags.Arg(0))
		return 2
	}
	if *all == (*fixture != "") {
		write(stderr, "ERROR: schema validate requires exactly one of --all or --fixture <dir>\n")
		return 2
	}

	report := schemaCheckReport{
		SchemaVersion: "schema-check/v1",
		Command:       "schema-check",
		Status:        "passed",
		CheckedAt:     time.Now().UTC().Format(time.RFC3339),
		ReportPath:    *reportPath,
	}
	if *all {
		report.Checks = append(report.Checks, validateRepoSchemaArtifacts()...)
	} else {
		report.Checks = append(report.Checks, validateFixtureSchemas(*fixture)...)
	}
	for _, check := range report.Checks {
		if check.Status != "passed" {
			report.Status = "failed"
			report.Gaps = append(report.Gaps, check.Gaps...)
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		write(stderr, "ERROR: schema-check marshal report: %v\n", err)
		return 1
	}
	data = append(data, '\n')
	write(stdout, "%s", data)
	if err := writeSchemaCheckReport(*reportPath, data); err != nil {
		write(stderr, "ERROR: schema-check write report: %v\n", err)
		return 1
	}
	if report.Status != "passed" {
		write(stderr, "ERROR: schema-check found %d gap(s)\n", len(report.Gaps))
		return 1
	}
	return 0
}

func writeSchemaCheckReport(path string, data []byte) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func validateRepoSchemaArtifacts() []schemaCheckItem {
	var checks []schemaCheckItem
	paths, err := filepath.Glob("contracts/*.schema.json")
	if err != nil || len(paths) == 0 {
		return []schemaCheckItem{
			{
				Name:     "contract schemas discovered",
				Artifact: "contracts/*.schema.json",
				Status:   "failed",
				Gaps:     []string{"no contract schemas discovered"},
			},
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		check := schemaCheckItem{Name: "contract schema parses", Artifact: path, Status: "passed"}
		schema, gaps := loadJSONSchema(path)
		if len(gaps) > 0 {
			check.Status = "failed"
			check.Gaps = gaps
		} else {
			if schema.Schema == "" {
				check.Gaps = append(check.Gaps, path+" missing $schema")
			}
			if !schemaAllowsType(schema, "object") {
				check.Gaps = append(check.Gaps, path+" root type is not object")
			}
			if len(check.Gaps) > 0 {
				check.Status = "failed"
			} else {
				check.Details = []string{"schema JSON parsed", "root object contract declared"}
			}
		}
		checks = append(checks, check)
	}
	checks = append(checks,
		validateYAMLArtifactAgainstSchema(".agent/registries/command-registry.yaml", "contracts/command-registry.schema.json"),
		validateYAMLArtifactAgainstSchema(".agent/registries/issue-registry.yaml", "contracts/issue-registry.schema.json"),
		validateYAMLArtifactAgainstSchema(".agent/policies/layer-governance.yaml", "contracts/layer-governance.schema.json"),
		validateLayerGovernanceSemantics(".agent/policies/layer-governance.yaml"),
	)
	return checks
}

func validateYAMLArtifactAgainstSchema(artifactPath, schemaPath string) schemaCheckItem {
	check := schemaCheckItem{Name: "repo artifact matches schema", Artifact: artifactPath, Schema: schemaPath, Status: "passed"}
	schema, gaps := loadJSONSchema(schemaPath)
	if len(gaps) > 0 {
		check.Status = "failed"
		check.Gaps = gaps
		return check
	}
	value, err := parseBaselineYAMLFile(artifactPath)
	if err != nil {
		check.Status = "failed"
		check.Gaps = []string{err.Error()}
		return check
	}
	gaps = validateValueAgainstSchema(value, schema, artifactPath)
	if len(gaps) > 0 {
		check.Status = "failed"
		check.Gaps = gaps
		return check
	}
	check.Details = []string{"artifact parsed", "required schema fields validated"}
	return check
}

func validateLayerGovernanceSemantics(artifactPath string) schemaCheckItem {
	check := schemaCheckItem{Name: "layer governance semantics", Artifact: artifactPath, Status: "passed"}
	value, err := parseBaselineYAMLFile(artifactPath)
	if err != nil {
		check.Status = "failed"
		check.Gaps = []string{err.Error()}
		return check
	}
	gaps := layerGovernanceSemanticGaps(artifactPath, value)
	if len(gaps) > 0 {
		check.Status = "failed"
		check.Gaps = gaps
		return check
	}
	check.Details = []string{
		"layer visibility and dependency direction validated",
		"public and private repository boundaries validated",
		"rule evidence coverage validated",
	}
	return check
}

func layerGovernanceSemanticGaps(artifactPath string, value map[string]any) []string {
	var gaps []string
	if got := schemaCheckStringField(value, "dependency_direction"); got != "L3>L2>L1>L0>Standard" {
		gaps = append(gaps, fmt.Sprintf("%s dependency_direction = %q; want L3>L2>L1>L0>Standard", artifactPath, got))
	}

	layerSpecs := map[string]struct {
		visibility string
		repos      []string
		deps       []string
		forbids    []string
	}{
		"Standard": {
			visibility: "public",
			repos:      []string{"resiliencx"},
			deps:       []string{},
			forbids:    []string{"business_semantics", "production_secrets"},
		},
		"L0": {
			visibility: "public",
			repos:      []string{"kernel"},
			deps:       []string{"Standard"},
			forbids:    []string{"l1_imports", "l2_imports", "l3_imports", "production_secrets"},
		},
		"L1": {
			visibility: "public",
			repos:      []string{"configx", "observex", "testkitx"},
			deps:       []string{"L0", "Standard"},
			forbids:    []string{"l2_imports", "l3_imports", "business_semantics", "production_secrets"},
		},
		"L2": {
			visibility: "public",
			repos:      []string{"redisx", "kafkax", "postgresx", "taosx", "ossx", "clickhousex", "natsx"},
			deps:       []string{"L1", "L0", "Standard"},
			forbids:    []string{"l3_imports", "business_repository", "business_schema", "production_secrets"},
		},
		"L3": {
			visibility: "private",
			repos:      []string{"x.go", "market-data", "market-engine", "macro-data", "macro-engine", "regime-engine"},
			deps:       []string{"L2", "L1", "L0", "Standard"},
			forbids:    []string{"public_release", "public_business_semantics", "committed_production_secrets"},
		},
	}
	layerIDs := []string{"Standard", "L0", "L1", "L2", "L3"}
	layers := schemaCheckMapList(value["layers"])
	layersByID := map[string]map[string]any{}
	for _, layer := range layers {
		id := schemaCheckStringField(layer, "id")
		if id == "" {
			gaps = append(gaps, artifactPath+" layers contains item missing id")
			continue
		}
		if _, exists := layersByID[id]; exists {
			gaps = append(gaps, fmt.Sprintf("%s duplicate layer %s", artifactPath, id))
			continue
		}
		layersByID[id] = layer
	}
	for id := range layersByID {
		if _, ok := layerSpecs[id]; !ok {
			gaps = append(gaps, fmt.Sprintf("%s unexpected layer %s", artifactPath, id))
		}
	}
	privateRepos := schemaCheckStringSet(layerSpecs["L3"].repos)
	for _, id := range layerIDs {
		spec := layerSpecs[id]
		layer, ok := layersByID[id]
		if !ok {
			gaps = append(gaps, fmt.Sprintf("%s missing required layer %s", artifactPath, id))
			continue
		}
		if got := schemaCheckStringField(layer, "visibility"); got != spec.visibility {
			gaps = append(gaps, fmt.Sprintf("%s %s visibility must be %s", artifactPath, id, spec.visibility))
		}
		repos := schemaCheckStringList(layer["repos"])
		schemaCheckExactStringSetGaps(&gaps, artifactPath, id, "repos", repos, spec.repos)
		schemaCheckExactStringSetGaps(&gaps, artifactPath, id, "may_depend_on", schemaCheckStringList(layer["may_depend_on"]), spec.deps)
		schemaCheckRequiredStringSetGaps(&gaps, artifactPath, id, "forbids", schemaCheckStringList(layer["forbids"]), spec.forbids)
		if id != "L3" {
			for _, repo := range repos {
				if privateRepos[repo] {
					gaps = append(gaps, fmt.Sprintf("%s public layer %s must not include private repo %s", artifactPath, id, repo))
				}
			}
		}
	}

	ruleEvidence := map[string][]string{
		"LAYER-P0-PRIVATE-BOUNDARY":     {"docs-check", "schema-check", "boundary", "standard-impact-check"},
		"LAYER-P0-DEPENDENCY-DIRECTION": {"schema-check", "boundary", "contracts"},
		"LAYER-P1-DOWNSTREAM-UPDATE":    {"standard-impact-check", "downstream-sync-policy", "release-manifest"},
		"LAYER-P2-ITERATION-EVIDENCE":   {"adr", "release-notes", "migration-guide"},
	}
	ruleLevels := map[string]string{
		"LAYER-P0-PRIVATE-BOUNDARY":     "P0",
		"LAYER-P0-DEPENDENCY-DIRECTION": "P0",
		"LAYER-P1-DOWNSTREAM-UPDATE":    "P1",
		"LAYER-P2-ITERATION-EVIDENCE":   "P2",
	}
	ruleIDs := []string{
		"LAYER-P0-PRIVATE-BOUNDARY",
		"LAYER-P0-DEPENDENCY-DIRECTION",
		"LAYER-P1-DOWNSTREAM-UPDATE",
		"LAYER-P2-ITERATION-EVIDENCE",
	}
	rulesByID := map[string]map[string]any{}
	for _, rule := range schemaCheckMapList(value["rules"]) {
		id := schemaCheckStringField(rule, "id")
		if id == "" {
			gaps = append(gaps, artifactPath+" rules contains item missing id")
			continue
		}
		if _, exists := rulesByID[id]; exists {
			gaps = append(gaps, fmt.Sprintf("%s duplicate rule %s", artifactPath, id))
			continue
		}
		rulesByID[id] = rule
	}
	for id := range rulesByID {
		if _, ok := ruleEvidence[id]; !ok {
			gaps = append(gaps, fmt.Sprintf("%s unexpected rule %s", artifactPath, id))
		}
	}
	for _, id := range ruleIDs {
		rule, ok := rulesByID[id]
		if !ok {
			gaps = append(gaps, fmt.Sprintf("%s missing required rule %s", artifactPath, id))
			continue
		}
		if got := schemaCheckStringField(rule, "level"); got != ruleLevels[id] {
			gaps = append(gaps, fmt.Sprintf("%s %s level must be %s", artifactPath, id, ruleLevels[id]))
		}
		schemaCheckRequiredStringSetGaps(&gaps, artifactPath, id, "evidence", schemaCheckStringList(rule["evidence"]), ruleEvidence[id])
	}
	return gaps
}

func schemaCheckMapList(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	maps := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if typed, ok := item.(map[string]any); ok {
			maps = append(maps, typed)
		}
	}
	return maps
}

func schemaCheckStringField(value map[string]any, key string) string {
	typed, _ := value[key].(string)
	return typed
}

func schemaCheckStringList(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	strings := make([]string, 0, len(items))
	for _, item := range items {
		if typed, ok := item.(string); ok {
			strings = append(strings, typed)
		}
	}
	return strings
}

func schemaCheckExactStringSetGaps(gaps *[]string, artifactPath, owner, field string, got, want []string) {
	schemaCheckRequiredStringSetGaps(gaps, artifactPath, owner, field, got, want)
	wantSet := schemaCheckStringSet(want)
	for _, item := range sortedStringSetDifference(got, wantSet) {
		*gaps = append(*gaps, fmt.Sprintf("%s %s %s unexpected %s", artifactPath, owner, field, item))
	}
}

func schemaCheckRequiredStringSetGaps(gaps *[]string, artifactPath, owner, field string, got, want []string) {
	gotSet := schemaCheckStringSet(got)
	for _, item := range sortedStringSetDifference(want, gotSet) {
		*gaps = append(*gaps, fmt.Sprintf("%s %s %s missing %s", artifactPath, owner, field, item))
	}
}

func schemaCheckStringSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}

func sortedStringSetDifference(items []string, contains map[string]bool) []string {
	var diff []string
	for _, item := range items {
		if !contains[item] {
			diff = append(diff, item)
		}
	}
	sort.Strings(diff)
	return diff
}

func validateFixtureSchemas(fixtureDir string) []schemaCheckItem {
	paths, err := filepath.Glob(filepath.Join(fixtureDir, "schemas", "*.schema.json"))
	if err != nil || len(paths) == 0 {
		return []schemaCheckItem{
			{
				Name:     "fixture schemas discovered",
				Artifact: filepath.Join(fixtureDir, "schemas"),
				Status:   "failed",
				Gaps:     []string{"no fixture schemas discovered"},
			},
		}
	}
	sort.Strings(paths)
	schemas := make(map[string]jsonSchema, len(paths))
	var checks []schemaCheckItem
	for _, path := range paths {
		schema, gaps := loadJSONSchema(path)
		if len(gaps) > 0 {
			checks = append(checks, schemaCheckItem{Name: "fixture schema parses", Artifact: path, Status: "failed", Gaps: gaps})
			continue
		}
		schemas[schemaFixtureKey(path)] = schema
		checks = append(checks, schemaCheckItem{Name: "fixture schema parses", Artifact: path, Status: "passed", Details: []string{"schema JSON parsed"}})
	}
	if len(schemas) == 0 {
		return checks
	}
	checks = append(checks, validateFixtureJSONGroup(filepath.Join(fixtureDir, "valid", "*.json"), schemas, true)...)
	checks = append(checks, validateFixtureJSONGroup(filepath.Join(fixtureDir, "invalid", "*.json"), schemas, false)...)
	return checks
}

func validateFixtureJSONGroup(pattern string, schemas map[string]jsonSchema, shouldPass bool) []schemaCheckItem {
	paths, err := filepath.Glob(pattern)
	if err != nil || len(paths) == 0 {
		status := "valid"
		if !shouldPass {
			status = "invalid"
		}
		return []schemaCheckItem{
			{
				Name:     status + " fixtures discovered",
				Artifact: pattern,
				Status:   "failed",
				Gaps:     []string{"no " + status + " fixtures discovered"},
			},
		}
	}
	sort.Strings(paths)
	var checks []schemaCheckItem
	for _, path := range paths {
		schemaPath, schema := selectFixtureSchema(path, schemas)
		check := schemaCheckItem{Name: "valid fixture accepted", Artifact: path, Schema: schemaPath, Status: "passed"}
		if !shouldPass {
			check.Name = "invalid fixture rejected"
		}
		value, err := readJSONValue(path)
		if err != nil {
			check.Status = "failed"
			check.Gaps = []string{err.Error()}
			checks = append(checks, check)
			continue
		}
		gaps := validateValueAgainstSchema(value, schema, path)
		if shouldPass && len(gaps) > 0 {
			check.Status = "failed"
			check.Gaps = gaps
		} else if !shouldPass && len(gaps) == 0 {
			check.Status = "failed"
			check.Gaps = []string{path + " invalid fixture unexpectedly passed"}
		} else if !shouldPass {
			check.Details = []string{"invalid fixture rejected: " + gaps[0]}
		} else {
			check.Details = []string{"valid fixture accepted"}
		}
		checks = append(checks, check)
	}
	return checks
}

func loadJSONSchema(path string) (jsonSchema, []string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return jsonSchema{}, []string{"missing " + path}
	}
	var schema jsonSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return jsonSchema{}, []string{path + " invalid schema JSON: " + err.Error()}
	}
	return schema, nil
}

func readJSONValue(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("missing %s", path)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("%s invalid JSON: %w", path, err)
	}
	return value, nil
}

func schemaFixtureKey(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(strings.TrimSuffix(base, ".json"), ".schema")
}

func selectFixtureSchema(path string, schemas map[string]jsonSchema) (string, jsonSchema) {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if schema, ok := schemas[stem]; ok {
		return filepath.Join("schemas", stem+".schema.json"), schema
	}
	keys := make([]string, 0, len(schemas))
	for key := range schemas {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return filepath.Join("schemas", keys[0]+".schema.json"), schemas[keys[0]]
}

func validateValueAgainstSchema(value any, schema jsonSchema, path string) []string {
	var gaps []string
	appendSchemaValidationGaps(value, schema, path, &gaps)
	return gaps
}

func appendSchemaValidationGaps(value any, schema jsonSchema, path string, gaps *[]string) {
	if len(schema.Type) > 0 && !valueMatchesAnyType(value, schema.Type) {
		*gaps = append(*gaps, fmt.Sprintf("%s expected type %s", path, strings.Join(schema.Type, "|")))
		return
	}
	if schema.Const != nil && !reflect.DeepEqual(value, schema.Const) {
		*gaps = append(*gaps, fmt.Sprintf("%s expected const %v", path, schema.Const))
	}
	if len(schema.Enum) > 0 {
		matched := false
		for _, candidate := range schema.Enum {
			if reflect.DeepEqual(value, candidate) {
				matched = true
				break
			}
		}
		if !matched {
			*gaps = append(*gaps, fmt.Sprintf("%s expected enum %v", path, schema.Enum))
		}
	}
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range schema.Required {
			if _, ok := typed[key]; !ok {
				*gaps = append(*gaps, fmt.Sprintf("%s missing required field %q", path, key))
			}
		}
		for key, childSchema := range schema.Properties {
			if childValue, ok := typed[key]; ok {
				appendSchemaValidationGaps(childValue, childSchema, path+"."+key, gaps)
			}
		}
	case []any:
		if schema.MinItems != nil && len(typed) < *schema.MinItems {
			*gaps = append(*gaps, fmt.Sprintf("%s expected at least %d item(s)", path, *schema.MinItems))
		}
		if schema.Items != nil {
			for index, item := range typed {
				appendSchemaValidationGaps(item, *schema.Items, fmt.Sprintf("%s[%d]", path, index), gaps)
			}
		}
	case string:
		if schema.MinLength != nil && len(typed) < *schema.MinLength {
			*gaps = append(*gaps, fmt.Sprintf("%s expected minLength %d", path, *schema.MinLength))
		}
		if schema.Pattern != "" {
			re, err := regexp.Compile(schema.Pattern)
			if err != nil {
				*gaps = append(*gaps, fmt.Sprintf("%s invalid schema pattern %q", path, schema.Pattern))
			} else if !re.MatchString(typed) {
				*gaps = append(*gaps, fmt.Sprintf("%s value does not match pattern %q", path, schema.Pattern))
			}
		}
	case float64:
		if schema.Minimum != nil && typed < *schema.Minimum {
			*gaps = append(*gaps, fmt.Sprintf("%s expected minimum %v", path, *schema.Minimum))
		}
	}
}

func valueMatchesAnyType(value any, types []string) bool {
	for _, typ := range types {
		if valueMatchesType(value, typ) {
			return true
		}
	}
	return false
}

func valueMatchesType(value any, typ string) bool {
	switch typ {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == float64(int64(number))
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	default:
		return true
	}
}

func schemaAllowsType(schema jsonSchema, typ string) bool {
	if len(schema.Type) == 0 {
		return true
	}
	for _, candidate := range schema.Type {
		if candidate == typ {
			return true
		}
	}
	return false
}

func parseBaselineYAMLFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("missing %s", path)
	}
	return parseBaselineYAML(string(data), path)
}

func parseBaselineYAML(text, path string) (map[string]any, error) {
	result := map[string]any{}
	var listKey string
	var current map[string]any
	for lineNumber, raw := range strings.Split(text, "\n") {
		line := stripYAMLComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if indent == 0 {
			key, value, ok := strings.Cut(trimmed, ":")
			if !ok {
				return nil, fmt.Errorf("%s:%d invalid YAML line", path, lineNumber+1)
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			current = nil
			if value == "" {
				result[key] = []any{}
				listKey = key
				continue
			}
			result[key] = parseYAMLScalar(value)
			listKey = ""
			continue
		}
		if strings.HasPrefix(trimmed, "- ") && listKey != "" {
			item := map[string]any{}
			list, _ := result[listKey].([]any)
			list = append(list, item)
			result[listKey] = list
			current = item
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if rest != "" {
				key, value, ok := strings.Cut(rest, ":")
				if !ok {
					return nil, fmt.Errorf("%s:%d invalid YAML list item", path, lineNumber+1)
				}
				item[strings.TrimSpace(key)] = parseYAMLScalar(strings.TrimSpace(value))
			}
			continue
		}
		if current != nil {
			key, value, ok := strings.Cut(trimmed, ":")
			if !ok {
				return nil, fmt.Errorf("%s:%d invalid YAML mapping", path, lineNumber+1)
			}
			current[strings.TrimSpace(key)] = parseYAMLScalar(strings.TrimSpace(value))
		}
	}
	return result, nil
}

func stripYAMLComment(line string) string {
	inQuote := false
	quote := rune(0)
	for index, r := range line {
		if r == '\'' || r == '"' {
			if !inQuote {
				inQuote = true
				quote = r
			} else if quote == r {
				inQuote = false
			}
		}
		if r == '#' && !inQuote {
			return line[:index]
		}
	}
	return line
}

func parseYAMLScalar(value string) any {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"'")
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
		if inner == "" {
			return []any{}
		}
		parts := strings.Split(inner, ",")
		items := make([]any, 0, len(parts))
		for _, part := range parts {
			items = append(items, parseYAMLScalar(part))
		}
		return items
	}
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	if number, err := strconv.ParseFloat(value, 64); err == nil && value != "" && !strings.ContainsAny(value, "-_") {
		return number
	}
	return value
}
