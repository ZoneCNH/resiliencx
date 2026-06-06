#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-.}"
REPORT_DIR="${SECRET_CHECK_REPORT_DIR:-reports}"
ALLOWLIST_FILE="${SECRET_CHECK_ALLOWLIST:-.agent/security/secret-allowlist.yaml}"

echo "checking secrets..."

python3 - "$ROOT" "$REPORT_DIR" "$ALLOWLIST_FILE" <<'PY'
from __future__ import annotations

import json
import os
import re
import subprocess
import sys
from pathlib import Path

root = Path(sys.argv[1]).resolve()
report_dir = Path(sys.argv[2])
if not report_dir.is_absolute():
    report_dir = root / report_dir
allowlist_path = Path(sys.argv[3])
if not allowlist_path.is_absolute():
    allowlist_path = root / allowlist_path

exclude_dirs = {".git", ".omc", ".omx", ".worktree", "vendor", "inbox", "reports"}
exclude_files = {"go.sum", "check_secrets.sh", "goal.md"}

patterns: list[tuple[str, re.Pattern[str]]] = [
    (
        "RULE-SECRET-ASSIGNMENT",
        re.compile(
            r"(?i)\b(password|passwd|secret|token|access[_ -]?key|secret[_ -]?key|authorization|cookie)\b"
            r"\s*[:=]\s*['\"]?(?!\*{3,}|x{3,}|<redacted>|redacted|null|none|example|masked)"
            r"[A-Za-z0-9_./+=:-]{8,}"
        ),
    ),
    ("RULE-SECRET-AWS-ACCESS-KEY", re.compile(r"AKIA[0-9A-Z]{16}")),
    ("RULE-SECRET-GITHUB-PAT", re.compile(r"github_pat_[A-Za-z0-9_]{20,}")),
    ("RULE-SECRET-GITHUB-TOKEN", re.compile(r"ghp_[A-Za-z0-9_]{36,}")),
    ("RULE-SECRET-SLACK-TOKEN", re.compile(r"xox[baprs]-[A-Za-z0-9-]{10,}")),
    ("RULE-SECRET-PRIVATE-KEY", re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----|BEGIN (RSA|OPENSSH) PRIVATE KEY")),
]


def load_allowlist(path: Path) -> list[str]:
    if not path.exists():
        return []
    entries: list[str] = []
    for raw in path.read_text(encoding="utf-8", errors="ignore").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or line in {"allowlist:", "entries:"}:
            continue
        if line.startswith("-"):
            line = line[1:].strip()
        if line.startswith("literal:"):
            line = line.split(":", 1)[1].strip()
        if line.startswith("value:"):
            line = line.split(":", 1)[1].strip()
        line = line.strip('"\'')
        if line:
            entries.append(line)
    return entries


def is_allowlisted(entries: list[str], rel: str, line_no: int, text: str) -> bool:
    haystack = f"{rel}:{line_no}:{text}"
    exact_path = f"{rel}:{line_no}"
    return any(entry in {rel, exact_path} or entry in haystack for entry in entries)


def git_files() -> list[Path] | None:
    try:
        proc = subprocess.run(
            ["git", "-C", str(root), "ls-files", "-z"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
        )
    except (OSError, subprocess.CalledProcessError):
        return None
    names = [name for name in proc.stdout.decode("utf-8", errors="ignore").split("\0") if name]
    return [root / name for name in names]


def fallback_files() -> list[Path]:
    return [path for path in root.rglob("*") if path.is_file()]


def should_scan(path: Path) -> bool:
    try:
        rel_parts = path.relative_to(root).parts
    except ValueError:
        return False
    if not rel_parts:
        return False
    if any(part in exclude_dirs for part in rel_parts[:-1]):
        return False
    if path.name in exclude_files:
        return False
    return True


def redact(line: str) -> str:
    redacted = re.sub(
        r"(?i)(\b(password|passwd|secret|token|access[_ -]?key|secret[_ -]?key|authorization|cookie)\b\s*[:=]\s*['\"]?)[^'\"\\s]+",
        r"\1<redacted>",
        line.strip(),
    )
    redacted = re.sub(r"AKIA[0-9A-Z]{16}", "AKIA<redacted>", redacted)
    redacted = re.sub(r"ghp_[A-Za-z0-9_]{8,}", "ghp_<redacted>", redacted)
    redacted = re.sub(r"github_pat_[A-Za-z0-9_]{8,}", "github_pat_<redacted>", redacted)
    redacted = re.sub(r"xox[baprs]-[A-Za-z0-9-]{8,}", "xox<redacted>", redacted)
    return redacted[:200]


allowlist = load_allowlist(allowlist_path)
files = git_files() or fallback_files()
checked_files = 0
findings: list[dict[str, object]] = []

for path in files:
    if not should_scan(path):
        continue
    checked_files += 1
    try:
        text = path.read_text(encoding="utf-8", errors="ignore")
    except OSError:
        continue
    rel = path.relative_to(root).as_posix()
    for line_no, line in enumerate(text.splitlines(), start=1):
        for rule_id, pattern in patterns:
            if rule_id == "RULE-SECRET-ASSIGNMENT" and path.suffix == ".go":
                continue
            if not pattern.search(line):
                continue
            if is_allowlisted(allowlist, rel, line_no, line):
                continue
            findings.append(
                {
                    "rule_id": rule_id,
                    "file": rel,
                    "line": line_no,
                    "excerpt": redact(line),
                }
            )

status = "failed" if findings else "passed"
report = {
    "schema_version": "1.0",
    "command": "secret-check",
    "status": status,
    "checked_files": checked_files,
    "allowlist": allowlist_path.relative_to(root).as_posix() if allowlist_path.exists() else None,
    "findings": findings,
}
report_dir.mkdir(parents=True, exist_ok=True)
(report_dir / "secret-check.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")

if findings:
    lines = ["FAIL: secret-check", f"checked_files: {checked_files}", f"findings: {len(findings)}"]
    for finding in findings:
        lines.append(f"- {finding['rule_id']} {finding['file']}:{finding['line']} {finding['excerpt']}")
else:
    lines = ["PASS: secret-check", f"checked_files: {checked_files}", "findings: 0"]
(report_dir / "secret-check.txt").write_text("\n".join(lines) + "\n", encoding="utf-8")

if findings:
    print("ERROR: possible secret found; see reports/secret-check.json and reports/secret-check.txt", file=sys.stderr)
    sys.exit(7)

print("secret check passed")
PY
