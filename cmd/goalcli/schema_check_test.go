package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaValidateFixtureAcceptsValidAndRejectsInvalid(t *testing.T) {
	fixture := writeSchemaCheckFixture(t, false)
	report := filepath.Join(t.TempDir(), "schema-check.json")
	var stdout, stderr bytes.Buffer

	got := run([]string{"schema", "validate", "--fixture", fixture, "--report", report}, strings.NewReader(""), &stdout, &stderr)

	if got != 0 {
		t.Fatalf("schema validate fixture exit = %d, stderr %q", got, stderr.String())
	}
	for _, needle := range []string{`"schema_version": "schema-check/v1"`, `"status": "passed"`, "valid fixture accepted", "invalid fixture rejected"} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("stdout missing %q in:\n%s", needle, stdout.String())
		}
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read schema-check report: %v", err)
	}
	if !strings.Contains(string(data), `"status": "passed"`) {
		t.Fatalf("report = %s; want passed status", string(data))
	}
}

func TestSchemaValidateFixtureFailsWhenInvalidFixturePasses(t *testing.T) {
	fixture := writeSchemaCheckFixture(t, true)
	report := filepath.Join(t.TempDir(), "schema-check.json")
	var stdout, stderr bytes.Buffer

	got := run([]string{"schema-check", "--fixture", fixture, "--report", report}, strings.NewReader(""), &stdout, &stderr)

	if got != 1 {
		t.Fatalf("schema-check invalid fixture exit = %d; want 1", got)
	}
	for _, needle := range []string{"schema-check found", "invalid fixture unexpectedly passed"} {
		combined := stdout.String() + stderr.String()
		if !strings.Contains(combined, needle) {
			t.Fatalf("output missing %q; stdout=%q stderr=%q", needle, stdout.String(), stderr.String())
		}
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read schema-check report: %v", err)
	}
	if !strings.Contains(string(data), `"status": "failed"`) {
		t.Fatalf("report = %s; want failed status", string(data))
	}
}

func TestSchemaCheckAllWritesRepoReport(t *testing.T) {
	chdir(t, repoRoot(t))
	report := filepath.Join(t.TempDir(), "schema-check.json")
	var stdout, stderr bytes.Buffer

	got := run([]string{"schema-check", "--all", "--report", report}, strings.NewReader(""), &stdout, &stderr)

	if got != 0 {
		t.Fatalf("schema-check --all exit = %d, stderr %q, stdout:\n%s", got, stderr.String(), stdout.String())
	}
	for _, needle := range []string{`"status": "passed"`, ".agent/registries/command-registry.yaml", ".agent/registries/issue-registry.yaml", ".agent/policies/layer-governance.yaml", "layer governance semantics"} {
		if !strings.Contains(stdout.String(), needle) {
			t.Fatalf("stdout missing %q in:\n%s", needle, stdout.String())
		}
	}
	if _, err := os.Stat(report); err != nil {
		t.Fatalf("schema-check report missing: %v", err)
	}
}

func TestSchemaValidateRequiresExactlyOneMode(t *testing.T) {
	var stdout, stderr bytes.Buffer

	got := run([]string{"schema", "validate"}, strings.NewReader(""), &stdout, &stderr)

	if got != 2 {
		t.Fatalf("schema validate without mode exit = %d; want 2", got)
	}
	if !strings.Contains(stderr.String(), "requires exactly one of --all or --fixture") {
		t.Fatalf("stderr = %q; want mode error", stderr.String())
	}
}

func TestLayerGovernanceSemanticCheckAcceptsRepoFixture(t *testing.T) {
	chdir(t, repoRoot(t))

	check := validateLayerGovernanceSemantics(".agent/policies/layer-governance.yaml")

	if check.Status != "passed" {
		t.Fatalf("layer governance semantics status = %s, gaps = %v", check.Status, check.Gaps)
	}
}

func TestLayerGovernanceSemanticCheckRejectsBoundaryDrift(t *testing.T) {
	chdir(t, repoRoot(t))
	data, err := os.ReadFile(".agent/policies/layer-governance.yaml")
	if err != nil {
		t.Fatalf("read layer governance fixture: %v", err)
	}
	text := string(data)
	text = strings.Replace(text,
		"repos: [redisx, kafkax, postgresx, taosx, ossx, clickhousex, natsx]",
		"repos: [redisx, kafkax, postgresx, taosx, ossx, clickhousex, market-data]",
		1,
	)
	text = strings.Replace(text,
		"  - id: L3\n    visibility: private",
		"  - id: L3\n    visibility: public",
		1,
	)
	path := filepath.Join(t.TempDir(), "layer-governance.yaml")
	writeSchemaCheckText(t, path, text)

	check := validateLayerGovernanceSemantics(path)

	if check.Status != "failed" {
		t.Fatalf("layer governance semantics status = %s; want failed", check.Status)
	}
	combined := strings.Join(check.Gaps, "\n")
	for _, needle := range []string{
		"L2 repos missing natsx",
		"public layer L2 must not include private repo market-data",
		"L3 visibility must be private",
	} {
		if !strings.Contains(combined, needle) {
			t.Fatalf("gaps missing %q in:\n%s", needle, combined)
		}
	}
}

func writeSchemaCheckFixture(t *testing.T, invalidFixturePasses bool) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"schemas", "valid", "invalid"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeSchemaCheckText(t, filepath.Join(root, "schemas", "sample.schema.json"), `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["name", "count", "tags"],
  "properties": {
    "name": {"type": "string", "minLength": 1},
    "count": {"type": "integer", "minimum": 0},
    "tags": {"type": "array", "minItems": 1, "items": {"type": "string"}}
  }
}
`)
	writeSchemaCheckText(t, filepath.Join(root, "valid", "sample.json"), `{"name":"ok","count":1,"tags":["fixture"]}`)
	invalid := `{"count":1,"tags":["fixture"]}`
	if invalidFixturePasses {
		invalid = `{"name":"ok","count":1,"tags":["fixture"]}`
	}
	writeSchemaCheckText(t, filepath.Join(root, "invalid", "sample.json"), invalid)
	return root
}

func writeSchemaCheckText(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
