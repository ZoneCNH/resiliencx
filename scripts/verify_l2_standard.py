#!/usr/bin/env python3
"""Validate L2 standards artifacts without provider connections."""
from __future__ import annotations
import json
from pathlib import Path
import sys
import yaml
from jsonschema import Draft202012Validator

ROOT = Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / ".agent" / "evidence" / "l2-standard"

REGISTRIES = {
    ".agent/registry/l2-contract-packs.yaml": ".agent/schemas/l2-contract-packs.schema.json",
    ".agent/registry/l2-capability-families.yaml": None,
    ".agent/registry/l2-golden-samples.yaml": None,
    ".agent/registry/l2-release-levels.yaml": None,
}
SCHEMAS = [
    ".agent/schemas/l2-capabilities.schema.json",
    ".agent/schemas/l2-contract-packs.schema.json",
    ".agent/schemas/l2-release-readiness.schema.json",
    ".agent/schemas/l2-compliance-matrix.schema.json",
]
TEMPLATE_REQUIRED = [
    "templates/l2/.agent/l2-capabilities.yaml",
    "templates/l2/.agent/gates/l2gate.yaml",
    "templates/l2/.agent/evidence/README.md",
    "templates/l2/.agent/evidence/l2/.gitkeep",
    "templates/l2/test/contract/l2_contract_test.go",
    "templates/l2/test/integration/README.md",
    "templates/l2/test/chaos/README.md",
    "templates/l2/test/benchmark/README.md",
    "templates/l2/test/adoption/README.md",
    "templates/l2/docker-compose.test.yml",
    "templates/l2/Makefile",
    "templates/l2/.github/workflows/l2-gates.yml",
]
DOC_REQUIRED = [
    "docs/testing/l2-adapter-testing-standard.md",
    "docs/testing/l2-capability-manifest.md",
    "docs/testing/l2-contract-pack-registry.md",
    "docs/testing/l2-evidence-standard.md",
    "docs/testing/l2-release-gate.md",
    "docs/testing/l2-compliance-matrix.md",
    "docs/testing/l2-rollout-playbook.md",
    "docs/testing/l2-downstream-adoption.md",
    "docs/testing/l2-compatibility-matrix.md",
]
FORBIDDEN_TERMS = ["provider_endpoint:", "provider_credentials:", "password:", "secret:", "token:"]
REQUIRED_PACKS = {
    "common",
    "kv",
    "ttl",
    "sql",
    "transaction",
    "pool",
    "pubsub",
    "request_reply",
    "eventlog",
    "producer",
    "consumer",
    "offset_commit",
    "objectstore",
    "columnstore",
    "timeseries",
}
REQUIRED_PROFILES = {"unit", "contract", "integration", "chaos", "benchmark", "adoption"}
REQUIRED_LEVELS = ["L2-T0", "L2-T1", "L2-T2", "L2-T3", "L2-T4"]
TEMPLATE_MAKE_TARGETS = [
    "l2-capability-check",
    "l2-contract",
    "l2-integration",
    "l2-chaos",
    "l2-benchmark",
    "l2-adoption",
    "l2-evidence",
    "l2-release-readiness",
    "l2-manifest-check",
    "l2-contract-placeholder",
    "l2-evidence-check",
    "l2-release-readiness-check",
]
DOC_REQUIRED_TERMS = [
    "resiliencx",
    "testkitx",
    "xlibgate",
    ".agent/evidence/l2",
    "provider-neutral",
]

def load_yaml(rel: str):
    with (ROOT / rel).open() as f:
        return yaml.safe_load(f)

def load_json(rel: str):
    with (ROOT / rel).open() as f:
        return json.load(f)

def result(name, status, details):
    return {"check": name, "status": status, "details": details}

def main() -> int:
    EVIDENCE.mkdir(parents=True, exist_ok=True)
    results = []

    parsed_registries = {}
    for rel, schema_rel in REGISTRIES.items():
        data = load_yaml(rel)
        parsed_registries[rel] = data
        if schema_rel:
            Draft202012Validator(load_json(schema_rel)).validate(data)
        results.append(result(rel, "PASS", "YAML parsed" + (" and schema validated" if schema_rel else "")))

    for rel in SCHEMAS:
        schema = load_json(rel)
        Draft202012Validator.check_schema(schema)
        results.append(result(rel, "PASS", "JSON schema parsed and meta-schema checked"))

    manifest = load_yaml("templates/l2/.agent/l2-capabilities.yaml")
    Draft202012Validator(load_json(".agent/schemas/l2-capabilities.schema.json")).validate(manifest)
    results.append(result("templates/l2/.agent/l2-capabilities.yaml", "PASS", "template manifest validates against capability schema"))

    missing = [rel for rel in TEMPLATE_REQUIRED + DOC_REQUIRED if not (ROOT / rel).exists()]
    if missing:
        raise AssertionError(f"missing required artifacts: {missing}")
    results.append(result("required-artifacts", "PASS", f"{len(TEMPLATE_REQUIRED)} template files and {len(DOC_REQUIRED)} docs present"))

    offenders = []
    for rel in TEMPLATE_REQUIRED + DOC_REQUIRED + list(REGISTRIES) + SCHEMAS:
        text = (ROOT / rel).read_text()
        for term in FORBIDDEN_TERMS:
            if term in text:
                offenders.append({"path": rel, "term": term})
    if offenders:
        raise AssertionError(f"forbidden provider/secret terms found: {offenders}")
    results.append(result("provider-boundary", "PASS", "no provider credentials/endpoints or secret fields in standards artifacts"))

    packs = parsed_registries[".agent/registry/l2-contract-packs.yaml"]["packs"]
    levels = parsed_registries[".agent/registry/l2-release-levels.yaml"]["levels"]
    pack_names = set(packs)
    missing_packs = sorted(REQUIRED_PACKS - pack_names)
    if missing_packs:
        raise AssertionError(f"missing required L2 contract packs: {missing_packs}")
    for name, pack in packs.items():
        profiles = set(pack.get("profiles", []))
        if not profiles:
            raise AssertionError(f"contract pack {name} has no profiles")
        if not profiles <= (REQUIRED_PROFILES | {"skeleton", "retrospective"}):
            raise AssertionError(f"contract pack {name} has unknown profiles: {sorted(profiles)}")
        for field in ["family", "title", "required_evidence", "capabilities"]:
            if not pack.get(field):
                raise AssertionError(f"contract pack {name} missing {field}")
    backlog = parsed_registries[".agent/registry/l2-contract-packs.yaml"].get("extension_backlog", [])
    if "ttl" not in backlog:
        raise AssertionError("extension_backlog must explicitly track ttl")
    results.append(result("contract-pack-invariants", "PASS", {"packs": sorted(pack_names), "extension_backlog_contains": "ttl"}))

    if list(levels) != REQUIRED_LEVELS:
        raise AssertionError(f"release levels must be ordered {REQUIRED_LEVELS}, got {list(levels)}")
    if levels["L2-T3"].get("release_allowed") is not True or levels["L2-T4"].get("factory_grade_allowed") is not True:
        raise AssertionError("release level flags must preserve L2-T3 release and L2-T4 factory-grade semantics")
    for level_name, level in levels.items():
        if not set(level.get("required_profiles", [])):
            raise AssertionError(f"release level {level_name} has no required profiles")
    results.append(result("release-level-invariants", "PASS", {"levels": list(levels)}))

    makefile_text = (ROOT / "templates/l2/Makefile").read_text()
    missing_targets = [target for target in TEMPLATE_MAKE_TARGETS if f"{target}:" not in makefile_text]
    if missing_targets:
        raise AssertionError(f"template Makefile missing L2 targets: {missing_targets}")
    results.append(result("template-make-targets", "PASS", TEMPLATE_MAKE_TARGETS))

    contract_test_text = (ROOT / "templates/l2/test/contract/l2_contract_test.go").read_text()
    if "t.Skip(" in contract_test_text:
        raise AssertionError("template contract test must not unconditionally skip")
    for snippet in ["os.ReadFile", "../../.agent/l2-capabilities.yaml", "schema_version", "contract_packs"]:
        if snippet not in contract_test_text:
            raise AssertionError(f"template contract test missing manifest shape check snippet: {snippet}")
    results.append(result("template-contract-test", "PASS", "local manifest shape check without skip"))

    compose_text = (ROOT / "templates/l2/docker-compose.test.yml").read_text()
    for snippet in ["provider-neutral", "profiles:", "placeholder", 'network_mode: "none"', "l2-standards-placeholder"]:
        if snippet not in compose_text:
            raise AssertionError(f"docker-compose.test.yml missing provider-neutral placeholder snippet: {snippet}")
    results.append(result("template-compose", "PASS", "provider-neutral placeholder with no default network access"))

    thin_docs = []
    missing_doc_terms = []
    for rel in DOC_REQUIRED:
        text = (ROOT / rel).read_text()
        if len(text) < 900:
            thin_docs.append(rel)
        missing_terms = [term for term in DOC_REQUIRED_TERMS if term not in text]
        if missing_terms:
            missing_doc_terms.append({"path": rel, "missing": missing_terms})
    if thin_docs:
        raise AssertionError(f"L2 guidance docs need expanded guidance: {thin_docs}")
    if missing_doc_terms:
        raise AssertionError(f"L2 guidance docs missing boundary terms: {missing_doc_terms}")
    results.append(result("docs-guidance-depth", "PASS", f"{len(DOC_REQUIRED)} L2 docs contain expanded provider-neutral guidance"))

    results.append(result("registry-summary", "PASS", {"contract_packs": sorted(packs), "release_levels": list(levels)}))

    (EVIDENCE / "schema-validate.json").write_text(json.dumps({"status": "PASS", "checks": [r for r in results if "schema" in r["details"] or "schema" in r["check"]]}, indent=2) + "\n")
    (EVIDENCE / "registry-check.json").write_text(json.dumps({"status": "PASS", "checks": [r for r in results if "registry" in r["check"]]}, indent=2) + "\n")
    (EVIDENCE / "template-check.json").write_text(json.dumps({"status": "PASS", "checks": [r for r in results if "template" in r["check"] or r["check"] in ["required-artifacts", "provider-boundary"]]}, indent=2) + "\n")
    (EVIDENCE / "verification-summary.json").write_text(json.dumps({"status": "PASS", "checks": results}, indent=2) + "\n")
    print(json.dumps({"status": "PASS", "checks": len(results), "evidence_dir": str(EVIDENCE.relative_to(ROOT))}, indent=2))
    return 0

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:
        EVIDENCE.mkdir(parents=True, exist_ok=True)
        (EVIDENCE / "verification-summary.json").write_text(json.dumps({"status": "FAIL", "error": str(exc)}, indent=2) + "\n")
        print(f"FAIL: {exc}", file=sys.stderr)
        raise
