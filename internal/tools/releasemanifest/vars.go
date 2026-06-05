// SPDX-License-Identifier: Apache-2.0
package main

var checkNames = []string{
	"fmt",
	"vet",
	"lint",
	"unit_test",
	"race_test",
	"boundary",
	"secret_scan",
	"security",
	"contract",
	"integration",
	"dependency_check",
	"standard_impact",
	"docs_check",
	"property",
	"golden",
	"fuzz_smoke",
	"debt",
	"architecture",
	"domain",
	"docs_drift",
	"dependency_debt",
	"security_debt",
	"testing_debt",
	"implementation_debt",
	"downstream_debt",
	"docker_toolchain_check",
	"docker_build_check",
	"docker_ci",
	"docker_release_check",
	"docker_release_final_check",
	"docker_goalcli_image",
	"docker_goalcli_version",
	"docker_runtime_check",
	"docker_drift_check",
	"docker_contract",
}

var checkEnvNames = map[string]string{
	"fmt":                        "FMT_STATUS",
	"vet":                        "VET_STATUS",
	"lint":                       "LINT_STATUS",
	"unit_test":                  "UNIT_TEST_STATUS",
	"race_test":                  "RACE_TEST_STATUS",
	"boundary":                   "BOUNDARY_STATUS",
	"secret_scan":                "SECRET_SCAN_STATUS",
	"security":                   "SECURITY_STATUS",
	"contract":                   "CONTRACT_STATUS",
	"integration":                "INTEGRATION_STATUS",
	"dependency_check":           "DEPENDENCY_CHECK_STATUS",
	"standard_impact":            "STANDARD_IMPACT_STATUS",
	"docs_check":                 "DOCS_CHECK_STATUS",
	"property":                   "PROPERTY_STATUS",
	"golden":                     "GOLDEN_STATUS",
	"fuzz_smoke":                 "FUZZ_SMOKE_STATUS",
	"debt":                       "DEBT_STATUS",
	"architecture":               "ARCHITECTURE_STATUS",
	"domain":                     "DOMAIN_STATUS",
	"docs_drift":                 "DOCS_DRIFT_STATUS",
	"dependency_debt":            "DEPENDENCY_DEBT_STATUS",
	"security_debt":              "SECURITY_DEBT_STATUS",
	"testing_debt":               "TESTING_DEBT_STATUS",
	"implementation_debt":        "IMPLEMENTATION_DEBT_STATUS",
	"downstream_debt":            "DOWNSTREAM_DEBT_STATUS",
	"docker_toolchain_check":     "DOCKER_TOOLCHAIN_CHECK_STATUS",
	"docker_build_check":         "DOCKER_BUILD_CHECK_STATUS",
	"docker_ci":                  "DOCKER_CI_STATUS",
	"docker_release_check":       "DOCKER_RELEASE_CHECK_STATUS",
	"docker_release_final_check": "DOCKER_RELEASE_FINAL_CHECK_STATUS",
	"docker_goalcli_image":       "DOCKER_GOALCLI_IMAGE_STATUS",
	"docker_goalcli_version":     "DOCKER_GOALCLI_VERSION_STATUS",
	"docker_runtime_check":       "DOCKER_RUNTIME_CHECK_STATUS",
	"docker_drift_check":         "DOCKER_DRIFT_CHECK_STATUS",
	"docker_contract":            "DOCKER_CONTRACT_STATUS",
}

var contractFiles = []string{
	"contracts/config.schema.json",
	"contracts/error.schema.json",
	"contracts/health.schema.json",
	"contracts/metrics.md",
	"contracts/docker-toolchain.schema.json",
}

var dockerEvidenceValidators = []string{
	"docker-toolchain-check",
	"docker-build-check",
	"docker-ci",
	"docker-release-check",
	"docker-release-final-check",
	"docker-goalcli-image",
	"docker-goalcli-version",
	"docker-runtime-check",
	"docker-drift-check",
	"docker-contract",
}

var requiredArtifacts = []string{
	defaultManifestOutputPath,
	defaultManifestChecksumPath,
	"release/debt/latest.json",
	"release/debt/latest.md",
	"release/debt/latest.json.sha256",
	"release/docker/toolchain-check.md",
	"release/evidence/docker-toolchain-summary.json",
}

const standardImpactReportPath = "release/standard-impact/latest.md"
const debtReportPath = "release/debt/latest.json"
const debtMarkdownPath = "release/debt/latest.md"
const debtChecksumPath = "release/debt/latest.json.sha256"
const governanceRuntimeVersion = "v2.9.3"

var downstreamReleaseDecisionValues = []string{
	"required",
	"not_required",
}

var repositoryRulesReleaseDecisionValues = []string{
	"audit_required",
	"not_required",
}

var generatorEvidenceTargets = []GeneratorTarget{
	{Name: "kernel", ModulePath: "github.com/ZoneCNH/kernel", PackageName: "kernel"},
	{Name: "configx", ModulePath: "github.com/ZoneCNH/configx", PackageName: "configx"},
	{Name: "redisx", ModulePath: "github.com/ZoneCNH/redisx", PackageName: "redisx"},
}

var governanceRuntimeGateStatuses = map[string]string{
	"governance": "passed",
}

var governanceRuntimeProfileStatuses = map[string]string{
	"p1_governance": "passed",
	"p2_runtime":    "passed",
}
