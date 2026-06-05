// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ZoneCNH/resiliencx/internal/releasequality"
)

// sourceDigest 计算所有跟踪文件的源码摘要。
func sourceDigest() (string, int, error) {
	raw, err := runRawCommand("git", "ls-files", "-z")
	if err != nil {
		return "", 0, err
	}
	parts := strings.Split(string(raw), "\x00")
	files := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			files = append(files, part)
		}
	}
	sort.Strings(files)

	digest := sha256.New()
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", 0, err
		}
		fileSum := sha256.Sum256(data)
		digest.Write([]byte(path))
		digest.Write([]byte{0})
		digest.Write([]byte(hex.EncodeToString(fileSum[:])))
		digest.Write([]byte{0})
	}

	return "sha256:" + hex.EncodeToString(digest.Sum(nil)), len(files), nil
}

// contractDigests 计算所有契约文件的摘要。
func contractDigests() ([]FileDigest, error) {
	digests := make([]FileDigest, 0, len(contractFiles))
	for _, path := range contractFiles {
		digest, err := fileDigest(path)
		if err != nil {
			return nil, err
		}
		digests = append(digests, digest)
	}
	return digests, nil
}

// fileDigest 计算单个文件的 SHA256 摘要。
func fileDigest(path string) (FileDigest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FileDigest{}, err
	}
	sum := sha256.Sum256(data)
	return FileDigest{
		Path:   path,
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
	}, nil
}

// moduleDigests 获取所有模块的摘要信息。
func moduleDigests() ([]ModuleDigest, error) {
	raw, err := runRawCommand("go", "list", "-m", "-json", "all")
	if err != nil {
		return nil, err
	}

	type goModule struct {
		Path    string
		Version string
		Main    bool
		Replace *struct {
			Path    string
			Version string
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	var modules []ModuleDigest
	for {
		var module goModule
		if err := decoder.Decode(&module); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		digest := ModuleDigest{
			Path:    module.Path,
			Version: module.Version,
			Main:    module.Main,
		}
		if module.Replace != nil {
			digest.Replace = &ModuleReplace{
				Path:    module.Replace.Path,
				Version: module.Replace.Version,
			}
		}
		modules = append(modules, digest)
	}
	return modules, nil
}

// treeState 获取 git 工作树状态。
func treeState() string {
	status, err := runTrimmed("git", "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return "unknown"
	}
	if status == "" {
		return "clean"
	}
	return "dirty"
}

// toolVersion 获取工具版本信息。
func toolVersion(name string, args ...string) string {
	if _, err := exec.LookPath(name); err != nil {
		return "missing"
	}
	output, err := runTrimmed(name, args...)
	if err != nil {
		return "error: " + firstLine(err.Error())
	}
	return firstLine(output)
}

// runTrimmedDefault 运行命令并返回修剪后的输出，失败时返回默认值。
func runTrimmedDefault(fallback string, name string, args ...string) string {
	output, err := runTrimmed(name, args...)
	if err != nil {
		return fallback
	}
	return output
}

// runTrimmed 运行命令并返回修剪后的输出。
func runTrimmed(name string, args ...string) (string, error) {
	output, err := runRawCommand(name, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// runRaw 运行命令并返回原始输出。
func runRaw(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

// runRawCommand 是 runRaw 的可替换版本，便于测试。
var runRawCommand = runRaw

// envBool 从环境变量读取布尔值。
func envBool(name string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}

// envCSVDefault 从环境变量读取逗号分隔的值列表。
func envCSVDefault(name string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	if len(items) == 0 {
		return append([]string(nil), fallback...)
	}
	return items
}

// envDefault 从环境变量读取字符串值。
func envDefault(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

// firstLine 返回字符串的第一行。
func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return value[:idx]
	}
	return value
}

// writeManifest 将清单写入文件。
func writeManifest(path string, manifest Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := encodeManifestFunc(&buf, manifest); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// encodeManifest 将清单编码为 JSON。
func encodeManifest(w io.Writer, manifest Manifest) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(manifest)
}

// encodeManifestFunc 是 encodeManifest 的可替换版本，便于测试。
var encodeManifestFunc = encodeManifest

// buildChecks 构建检查状态映射。
func buildChecks() map[string]string {
	defaultStatus := envDefault("CHECK_STATUS", "unknown")
	checks := make(map[string]string, len(checkNames))
	for _, name := range checkNames {
		checks[name] = envDefault(checkEnvNames[name], defaultStatus)
	}
	return checks
}

// buildWorkflowEvidence 构建工作流证据。
func buildWorkflowEvidence() WorkflowEvidence {
	runID := envDefault("WORKFLOW_RUN_ID", envDefault("GITHUB_RUN_ID", "local"))
	artifactName := envDefault("ARTIFACT_NAME", "release-manifest-"+runID)
	artifactURL := envDefault("ARTIFACT_URL", "")
	if artifactURL == "" {
		server := strings.TrimRight(envDefault("GITHUB_SERVER_URL", ""), "/")
		repo := strings.Trim(os.Getenv("GITHUB_REPOSITORY"), "/")
		if server != "" && repo != "" && runID != "local" {
			artifactURL = server + "/" + repo + "/actions/runs/" + runID
		} else {
			artifactURL = "local:" + artifactName
		}
	}
	return WorkflowEvidence{
		WorkflowRunID: runID,
		ArtifactName:  artifactName,
		ArtifactURL:   artifactURL,
	}
}

// buildManifest 构建发布清单。
func buildManifest() (Manifest, error) {
	module, err := runTrimmed("go", "list", "-m")
	if err != nil {
		return Manifest{}, err
	}

	sourceDigest, trackedFileCount, err := sourceDigest()
	if err != nil {
		return Manifest{}, err
	}
	contracts, err := contractDigests()
	if err != nil {
		return Manifest{}, err
	}
	dependencies, err := moduleDigests()
	if err != nil {
		return Manifest{}, err
	}
	standardImpact, err := buildStandardImpactEvidence()
	if err != nil {
		return Manifest{}, err
	}
	debtEvidence, err := buildDebtEvidence()
	if err != nil {
		return Manifest{}, err
	}

	return Manifest{
		Module:                 module,
		Version:                envDefault("VERSION", defaultReleaseVersion),
		Commit:                 runTrimmedDefault("unknown", "git", "rev-parse", "HEAD"),
		TreeSHA:                runTrimmedDefault("unknown", "git", "rev-parse", "HEAD^{tree}"),
		SourceDigest:           sourceDigest,
		TrackedFileCount:       trackedFileCount,
		GoVersion:              runtime.Version(),
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		GeneratedBy:            envDefault("GENERATED_BY", "scripts/generate_manifest.sh"),
		TreeState:              treeState(),
		Checks:                 buildChecks(),
		Workflow:               buildWorkflowEvidence(),
		Docker:                 buildDockerEvidence(),
		Score:                  releasequality.Compute(releasequality.DefaultMinimum),
		Contracts:              contracts,
		Dependencies:           dependencies,
		StandardImpact:         standardImpact,
		Debt:                   debtEvidence,
		GovernanceRuntime:      buildGovernanceRuntime(),
		DownstreamSyncRequired: standardImpact.DownstreamSyncRequired,
		GeneratorEvidence:      buildGeneratorEvidence(),
		Tools: map[string]string{
			"go":            firstLine(runTrimmedDefault(runtime.Version(), "go", "version")),
			"golangci-lint": toolVersion("golangci-lint", "--version"),
			"govulncheck":   toolVersion("govulncheck", "-version"),
		},
		Artifacts: append([]string(nil), requiredArtifacts...),
		Notes: Notes{
			BreakingChanges: "none",
			KnownRisks:      []string{},
		},
	}, nil
}
