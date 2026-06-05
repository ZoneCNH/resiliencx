package scripts

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePreflightRejectsNonMainBranch(t *testing.T) {
	repo := newReleasePreflightRepo(t)
	runGit(t, repo, "switch", "-c", "feature/branch-governance")

	result := runReleasePreflight(t, repo)

	if result.code == 0 {
		t.Fatalf("release preflight passed on non-main branch; output:\n%s", result.output)
	}
	if !strings.Contains(result.output, "release preflight must run on main") {
		t.Fatalf("output missing non-main rejection:\n%s", result.output)
	}
}

func TestReleasePreflightRejectsDirtyMain(t *testing.T) {
	repo := newReleasePreflightRepo(t)
	writeFixtureFile(t, repo, "uncommitted.txt")

	result := runReleasePreflight(t, repo)

	if result.code == 0 {
		t.Fatalf("release preflight passed with dirty worktree; output:\n%s", result.output)
	}
	if !strings.Contains(result.output, "requires a clean git worktree") {
		t.Fatalf("output missing dirty worktree rejection:\n%s", result.output)
	}
}

func TestReleasePreflightRejectsOriginMainMismatch(t *testing.T) {
	repo := newReleasePreflightRepo(t)
	advanceOriginMain(t, repo)

	result := runReleasePreflight(t, repo)

	if result.code == 0 {
		t.Fatalf("release preflight passed when local main lagged origin/main; output:\n%s", result.output)
	}
	if !strings.Contains(result.output, "local main is not aligned with origin/main") {
		t.Fatalf("output missing origin/main alignment rejection:\n%s", result.output)
	}
}

func TestReleasePreflightPassesCleanAlignedMain(t *testing.T) {
	repo := newReleasePreflightRepo(t)

	result := runReleasePreflight(t, repo)

	if result.code != 0 {
		t.Fatalf("release preflight failed for clean aligned main: exit %d\n%s", result.code, result.output)
	}
	if !strings.Contains(result.output, "release preflight metadata checks passed for v0.9.0") {
		t.Fatalf("output missing success message:\n%s", result.output)
	}
}

type releasePreflightResult struct {
	code   int
	output string
}

func newReleasePreflightRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	runGit(t, root, "init", "--bare", origin)

	repo := filepath.Join(root, "repo")
	runGit(t, root, "clone", origin, repo)
	runGit(t, repo, "switch", "-c", "main")
	runGit(t, repo, "config", "user.name", "Release Preflight Test")
	runGit(t, repo, "config", "user.email", "release-preflight@example.com")
	writeFixtureFile(t, repo, "README.md")
	writeFile(t, filepath.Join(repo, "CHANGELOG.md"), "# Changelog\n\n## [v0.9.0] - 2026-06-05\n\n- Test fixture release.\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial release fixture")
	runGit(t, repo, "push", "-u", "origin", "main")
	return repo
}

func advanceOriginMain(t *testing.T, repo string) {
	t.Helper()

	remoteURL := strings.TrimSpace(gitOutput(t, repo, "remote", "get-url", "origin"))
	clone := filepath.Join(t.TempDir(), "origin-advancer")
	runGit(t, filepath.Dir(clone), "clone", remoteURL, clone)
	runGit(t, clone, "switch", "main")
	runGit(t, clone, "config", "user.name", "Release Preflight Test")
	runGit(t, clone, "config", "user.email", "release-preflight@example.com")
	writeFixtureFile(t, clone, "origin-only.txt")
	runGit(t, clone, "add", ".")
	runGit(t, clone, "commit", "-m", "advance origin main")
	runGit(t, clone, "push", "origin", "main")
}

func runReleasePreflight(t *testing.T, repo string) releasePreflightResult {
	t.Helper()

	scriptsDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get scripts directory: %v", err)
	}
	fakeBin := filepath.Join(t.TempDir(), "bin")
	writeReleasePreflightExecutable(t, fakeBin, "golangci-lint", "#!/bin/sh\nexit 0\n")

	cmd := exec.Command("bash", filepath.Join(scriptsDir, "check_release_preflight.sh"), "v0.9.0")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	if err == nil {
		return releasePreflightResult{code: 0, output: string(output)}
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("release preflight failed without exit status: %v\n%s", err, output)
	}
	return releasePreflightResult{code: exitErr.ExitCode(), output: string(output)}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func writeReleasePreflightExecutable(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create fake command directory: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake command %s: %v", name, err)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
