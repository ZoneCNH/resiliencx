package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTemplateExcludesGeneratedReleaseArtifacts(t *testing.T) {
	contents, err := os.ReadFile("render_template.sh")
	if err != nil {
		t.Fatalf("read render_template.sh: %v", err)
	}

	script := string(contents)
	for _, required := range []string{
		"--exclude='./.omc'",
		"--exclude='./.omx'",
		"--exclude='./.worktree'",
		"--exclude='./release/manifest/latest.json'",
		"--exclude='./release/manifest/latest.json.sha256'",
		"--exclude='./release/standard-impact/latest.md'",
		"--exclude='./release/downstream-sync/latest.md'",
		"--exclude='./release/debt/latest.json'",
		"--exclude='./release/debt/latest.md'",
		"--exclude='./release/debt/latest.json.sha256'",
		`rm -rf "$out_dir/.omc"`,
		`rm -rf "$out_dir/.omx"`,
		`rm -rf "$out_dir/.worktree"`,
		`rm -f "$out_dir/release/manifest/latest.json"`,
		`rm -f "$out_dir/release/manifest/latest.json.sha256"`,
		`rm -f "$out_dir/release/standard-impact/latest.md"`,
		`rm -f "$out_dir/release/downstream-sync/latest.md"`,
		`rm -f "$out_dir/release/debt/latest.json"`,
		`rm -f "$out_dir/release/debt/latest.md"`,
		`rm -f "$out_dir/release/debt/latest.json.sha256"`,
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("render_template.sh missing render omission rule %q", required)
		}
	}
}

func TestRenderTemplateIncludesGoalcliControlPlane(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "configx")
	cmd := exec.Command(
		"bash",
		"render_template.sh",
		"--module-name",
		"configx",
		"--module-path",
		"github.com/ZoneCNH/configx",
		"--package-name",
		"configx",
		"--out",
		outDir,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render template: %v\n%s", err, output)
	}

	for _, required := range []string{
		filepath.Join("cmd", "goalcli", "main.go"),
		filepath.Join("cmd", "goalcli", "main_test.go"),
		filepath.Join("cmd", "goalcli", "governance.go"),
		filepath.Join("internal", "goalcli", "README.md"),
		"Makefile",
		filepath.Join(".agent", "index.yaml"),
		filepath.Join(".agent", "harness", "harness.yaml"),
		filepath.Join(".agent", "harness", "gates.md"),
		filepath.Join(".agent", "registries", "command-registry.yaml"),
		filepath.Join(".agent", "registries", "command-implementation-status.yaml"),
		filepath.Join(".agent", "registries", "makefile-baseline.yaml"),
		filepath.Join(".agent", "registries", "makefile-target-registry.yaml"),
		filepath.Join("contracts", "goalcli-report.schema.json"),
		filepath.Join("docs", "standard", "goalcli-cli-contract.md"),
		filepath.Join("docs", "standard", "goalcli-runtime.md"),
	} {
		if _, err := os.Stat(filepath.Join(outDir, required)); err != nil {
			t.Fatalf("rendered template missing goalcli control-plane path %s: %v", required, err)
		}
	}

	makefile, err := os.ReadFile(filepath.Join(outDir, "Makefile"))
	if err != nil {
		t.Fatalf("read rendered Makefile: %v", err)
	}
	if !strings.Contains(string(makefile), "GOALCLI ?= go run ./cmd/goalcli") {
		t.Fatalf("rendered Makefile missing GOALCLI entrypoint")
	}
}

func TestRenderTemplateIncludesGovernancePack(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "kernel")
	cmd := exec.Command(
		"bash",
		"render_template.sh",
		"--module-name",
		"kernel",
		"--module-path",
		"github.com/ZoneCNH/kernel",
		"--package-name",
		"kernel",
		"--layer",
		"L0",
		"--enable-governance",
		"--standard-version",
		"v0.5.0",
		"--standard-commit",
		"abcdef1234567890",
		"--out",
		outDir,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render template with governance pack: %v\n%s", err, output)
	}

	for _, required := range []string{
		"xlib-standard.lock",
		filepath.Join(".githooks", "pre-commit"),
		filepath.Join(".githooks", "pre-push"),
		filepath.Join(".github", "workflows", "adoption-check.yml"),
		filepath.Join(".github", "rulesets", "protect-main.json"),
		filepath.Join("mk", "governance.mk"),
		filepath.Join(".agent", "harness", "harness.yaml"),
	} {
		if _, err := os.Stat(filepath.Join(outDir, required)); err != nil {
			t.Fatalf("rendered governance pack missing %s: %v", required, err)
		}
	}

	lock, err := os.ReadFile(filepath.Join(outDir, "xlib-standard.lock"))
	if err != nil {
		t.Fatalf("read governance lock: %v", err)
	}
	for _, needle := range []string{
		`standard_version: "v0.5.0"`,
		`standard_commit: "abcdef1234567890"`,
		`module_name: "kernel"`,
		`module_path: "github.com/ZoneCNH/kernel"`,
		`package_name: "kernel"`,
		`layer: "L0"`,
		`adoption_check: "GOWORK=off make adoption-check"`,
	} {
		if !strings.Contains(string(lock), needle) {
			t.Fatalf("governance lock missing %q:\n%s", needle, lock)
		}
	}

	adoption := exec.Command("make", "adoption-check")
	adoption.Dir = outDir
	adoption.Env = append(os.Environ(), "GOWORK=off")
	output, err = adoption.CombinedOutput()
	if err != nil {
		t.Fatalf("rendered governance adoption-check: %v\n%s", err, output)
	}
}

func TestRenderTemplateIncludesDockerContract(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "kernel")
	cmd := exec.Command(
		"bash",
		"render_template.sh",
		"--module-name",
		"kernel",
		"--module-path",
		"github.com/ZoneCNH/kernel",
		"--package-name",
		"kernel",
		"--out",
		outDir,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render template: %v\n%s", err, output)
	}

	for _, required := range []string{
		"Dockerfile",
		"docker-compose.yml",
		".dockerignore",
		filepath.Join(".devcontainer", "devcontainer.json"),
		filepath.Join("scripts", "docker", "check_toolchain.sh"),
		filepath.Join("scripts", "docker", "docker_gate.sh"),
	} {
		if _, err := os.Stat(filepath.Join(outDir, required)); err != nil {
			t.Fatalf("rendered template missing Docker contract path %s: %v", required, err)
		}
	}

	dockerfile, err := os.ReadFile(filepath.Join(outDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("read rendered Dockerfile: %v", err)
	}
	for _, required := range []string{
		"GOLANGCI_LINT_VERSION",
		"GOVULNCHECK_VERSION",
		"python3-yaml",
		"github.com/golangci/golangci-lint/v2/cmd/golangci-lint",
		"golang.org/x/vuln/cmd/govulncheck",
		"safe.directory /workspace",
	} {
		if !strings.Contains(string(dockerfile), required) {
			t.Fatalf("rendered Dockerfile missing toolchain marker %s", required)
		}
	}

	makefile, err := os.ReadFile(filepath.Join(outDir, "Makefile"))
	if err != nil {
		t.Fatalf("read rendered Makefile: %v", err)
	}
	if !strings.Contains(string(makefile), `GITHUB_ACTIONS=$${GITHUB_ACTIONS:-}`) {
		t.Fatalf("rendered Makefile missing Docker CI environment passthrough")
	}
	if !strings.Contains(string(makefile), `GOLANGCI_LINT_VERSION=$${GOLANGCI_LINT_VERSION:-v2.1.6}`) {
		t.Fatalf("rendered Makefile missing Docker lint toolchain build arg")
	}
	if !strings.Contains(string(makefile), `GIT_CONFIG_VALUE_0=/workspace`) {
		t.Fatalf("rendered Makefile missing Docker Git workspace trust config")
	}
	for _, target := range []string{
		"docker-toolchain-check",
		"docker-build",
		"docker-build-check",
		"docker-shell",
		"docker-ci",
		"docker-release-check",
		"docker-release-final-check",
		"docker-goalcli",
		"docker-goalcli-image",
		"docker-goalcli-version",
		"docker-runtime-check",
		"docker-drift-check",
		"docker-contract",
	} {
		if !strings.Contains(string(makefile), target) {
			t.Fatalf("rendered Makefile missing Docker contract target %s", target)
		}
	}

	dockerGate, err := os.ReadFile(filepath.Join(outDir, "scripts", "docker", "docker_gate.sh"))
	if err != nil {
		t.Fatalf("read rendered Docker gate: %v", err)
	}
	if !strings.Contains(string(dockerGate), `GITHUB_ACTIONS=${GITHUB_ACTIONS:-}`) {
		t.Fatalf("rendered Docker gate missing GitHub Actions environment passthrough")
	}
	if !strings.Contains(string(dockerGate), `GOLANGCI_LINT_VERSION:-v2.1.6`) {
		t.Fatalf("rendered Docker gate missing lint toolchain build arg")
	}
	if !strings.Contains(string(dockerGate), `GOVULNCHECK_VERSION:-v1.1.4`) {
		t.Fatalf("rendered Docker gate missing govulncheck toolchain build arg")
	}
	if !strings.Contains(string(dockerGate), `GIT_CONFIG_VALUE_0=/workspace`) {
		t.Fatalf("rendered Docker gate missing Git workspace trust config")
	}
}

func TestRenderTemplatePrunesOmittedAgentInboxIndexEntries(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "kernel")
	cmd := exec.Command(
		"bash",
		"render_template.sh",
		"--module-name",
		"kernel",
		"--module-path",
		"github.com/ZoneCNH/kernel",
		"--package-name",
		"kernel",
		"--out",
		outDir,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render template: %v\n%s", err, output)
	}

	index, err := os.ReadFile(filepath.Join(outDir, ".agent", "index.yaml"))
	if err != nil {
		t.Fatalf("read rendered agent index: %v", err)
	}
	if strings.Contains(string(index), ".agent/inbox/") {
		t.Fatalf("rendered agent index still references omitted inbox entries")
	}
	if _, err := os.Stat(filepath.Join(outDir, ".agent", "inbox")); !os.IsNotExist(err) {
		t.Fatalf("rendered agent inbox should be omitted, stat err=%v", err)
	}
}

func TestRenderTemplateGitArchivePrunesRuntimeState(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "redisx")
	cmd := exec.Command(
		"bash",
		"render_template.sh",
		"--module-name",
		"redisx",
		"--module-path",
		"github.com/ZoneCNH/redisx",
		"--package-name",
		"redisx",
		"--out",
		outDir,
	)
	cmd.Env = append(os.Environ(), "XLIB_RENDER_FORCE_GIT_ARCHIVE=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render template from git archive: %v\n%s", err, output)
	}

	for _, omitted := range []string{
		".omc",
		".omx",
		".worktree",
		filepath.Join("release", "manifest", "latest.json"),
		filepath.Join("release", "manifest", "latest.json.sha256"),
		filepath.Join("release", "standard-impact", "latest.md"),
		filepath.Join("release", "downstream-sync", "latest.md"),
		filepath.Join("release", "debt", "latest.json"),
		filepath.Join("release", "debt", "latest.md"),
		filepath.Join("release", "debt", "latest.json.sha256"),
	} {
		if _, err := os.Stat(filepath.Join(outDir, omitted)); !os.IsNotExist(err) {
			t.Fatalf("rendered git archive should omit %s, stat err=%v", omitted, err)
		}
	}
}

func TestRenderTemplateGitArchiveSkipsUntrackedFiles(t *testing.T) {
	markerPath := filepath.Join("..", ".xlib-render-untracked-marker-test")
	if err := os.WriteFile(markerPath, []byte("untracked marker"), 0o600); err != nil {
		t.Fatalf("write untracked marker: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(markerPath)
	})

	outDir := filepath.Join(t.TempDir(), "kernel")
	cmd := exec.Command(
		"bash",
		"render_template.sh",
		"--module-name",
		"kernel",
		"--module-path",
		"github.com/ZoneCNH/kernel",
		"--package-name",
		"kernel",
		"--out",
		outDir,
	)
	cmd.Env = append(os.Environ(), "XLIB_RENDER_FORCE_GIT_ARCHIVE=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render template: %v\n%s", err, output)
	}

	if _, err := os.Stat(filepath.Join(outDir, ".xlib-render-untracked-marker-test")); !os.IsNotExist(err) {
		t.Fatalf("expected git archive render to skip untracked marker, stat err=%v", err)
	}
}
