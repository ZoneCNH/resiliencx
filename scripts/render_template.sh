#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/render_template.sh --module-name NAME --module-path PATH --package-name NAME --out DIR [--enable-governance --layer LAYER --standard-version VERSION --standard-commit COMMIT]

Renders xlib-standard into a concrete base library by copying the repository,
moving pkg/templatex to pkg/<package>, and replacing template identifiers.
When --enable-governance is set, the rendered repository also receives an
xlib-standard.lock file and must retain the Repository Governance Pack.
USAGE
}

module_name=""
module_path=""
package_name=""
out_dir=""
enable_governance=0
layer=""
standard_version=""
standard_commit=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --module-name)
      module_name="${2:-}"
      shift 2
      ;;
    --module-path)
      module_path="${2:-}"
      shift 2
      ;;
    --package-name)
      package_name="${2:-}"
      shift 2
      ;;
    --out)
      out_dir="${2:-}"
      shift 2
      ;;
    --enable-governance)
      enable_governance=1
      shift
      ;;
    --layer)
      layer="${2:-}"
      shift 2
      ;;
    --standard-version)
      standard_version="${2:-}"
      shift 2
      ;;
    --standard-commit)
      standard_commit="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$module_name" || -z "$module_path" || -z "$package_name" || -z "$out_dir" ]]; then
  echo "ERROR: --module-name, --module-path, --package-name and --out are required" >&2
  usage >&2
  exit 2
fi

if [[ "$enable_governance" == "1" && ( -z "$layer" || -z "$standard_version" || -z "$standard_commit" ) ]]; then
  echo "ERROR: --enable-governance requires --layer, --standard-version and --standard-commit" >&2
  usage >&2
  exit 2
fi

if [[ "$package_name" =~ [^a-zA-Z0-9_] || "$package_name" =~ ^[0-9] ]]; then
  echo "ERROR: --package-name must be a valid Go package identifier" >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_abs="$(realpath "$repo_root")"
out_abs="$(realpath -m "$out_dir")"

if [[ "$out_abs" == "$repo_abs" || "$out_abs" == "$repo_abs"/* ]]; then
  echo "ERROR: output directory must be outside the template repository: $out_abs" >&2
  exit 2
fi

if [[ -e "$out_abs" && ! -d "$out_abs" ]]; then
  echo "ERROR: output path exists but is not a directory: $out_abs" >&2
  exit 2
fi

if [[ -e "$out_abs/.git" || -e "$out_abs/go.mod" ]]; then
  echo "ERROR: output directory looks like an existing repository: $out_abs" >&2
  exit 2
fi

if [[ -d "$out_abs" ]] && find "$out_abs" -mindepth 1 -maxdepth 1 | read -r _; then
  echo "ERROR: output directory must be empty: $out_abs" >&2
  exit 2
fi

mkdir -p "$out_abs"
out_dir="$out_abs"

copy_from_live_tree() {
  (
    cd "$repo_root"
    tar \
    --exclude='./.git' \
    --exclude='./.omc' \
    --exclude='./.omx' \
    --exclude='./.worktree' \
    --exclude='./.agent/inbox' \
    --exclude='./docs/adr' \
    --exclude='./docs/goal.md' \
    --exclude='./tmp' \
    --exclude='./dist' \
    --exclude='./node_modules' \
    --exclude='./coverage.out' \
    --exclude='./coverage.*' \
    --exclude='./*.coverprofile' \
    --exclude='./profile.cov' \
    --exclude='./release/manifest/latest.json' \
    --exclude='./release/manifest/latest.json.sha256' \
    --exclude='./release/standard-impact/latest.md' \
    --exclude='./release/downstream-sync/latest.md' \
    --exclude='./release/debt/latest.json' \
    --exclude='./release/debt/latest.md' \
    --exclude='./release/debt/latest.json.sha256' \
    -cf - .
  ) | (
    cd "$out_dir"
    tar -xf -
  )
}

prune_render_omissions() {
  rm -rf "$out_dir/.omc"
  rm -rf "$out_dir/.omx"
  rm -rf "$out_dir/.worktree"
  rm -rf "$out_dir/.agent/inbox"
  rm -rf "$out_dir/docs/adr"
  rm -f "$out_dir/docs/goal.md"
  rm -f "$out_dir/release/manifest/latest.json"
  rm -f "$out_dir/release/manifest/latest.json.sha256"
  rm -f "$out_dir/release/standard-impact/latest.md"
  rm -f "$out_dir/release/downstream-sync/latest.md"
  rm -f "$out_dir/release/debt/latest.json"
  rm -f "$out_dir/release/debt/latest.md"
  rm -f "$out_dir/release/debt/latest.json.sha256"
}

copy_from_git_archive() {
  git -C "$repo_root" archive --format=tar HEAD | (
    cd "$out_dir"
    tar -xf -
  )
  prune_render_omissions
}

use_git_archive=0
if [[ "${XLIB_RENDER_FORCE_GIT_ARCHIVE:-0}" == "1" ]]; then
  use_git_archive=1
elif git -C "$repo_root" rev-parse --is-inside-work-tree >/dev/null 2>&1 && \
  [[ -z "$(git -C "$repo_root" status --porcelain=v1 --untracked-files=no)" ]]; then
  use_git_archive=1
fi

if [[ "$use_git_archive" == "1" ]]; then
  copy_from_git_archive
else
  copy_from_live_tree
fi

# Raw inbox archives are intentionally omitted from rendered downstream repos.
# Keep the rendered control-plane index aligned with that reduced file set.
index_path="$out_dir/.agent/index.yaml"
if [[ -f "$index_path" ]]; then
  awk '
    /^  - path: \.agent\/inbox\// {
      skip = 1
      next
    }
    skip && /^    / {
      next
    }
    {
      skip = 0
      print
    }
  ' "$index_path" > "$index_path.tmp"
  mv "$index_path.tmp" "$index_path"
fi

source_package_dir=""
if [[ -d "$out_dir/pkg/templatex" ]]; then
  source_package_dir="$out_dir/pkg/templatex"
elif [[ -d "$out_dir/pkg/resiliencx" ]]; then
  source_package_dir="$out_dir/pkg/resiliencx"
fi

if [[ -z "$source_package_dir" ]]; then
  echo "ERROR: rendered source package not found: expected pkg/templatex or pkg/resiliencx" >&2
  exit 1
fi

if [[ "$source_package_dir" != "$out_dir/pkg/$package_name" ]]; then
  mkdir -p "$out_dir/pkg"
  mv "$source_package_dir" "$out_dir/pkg/$package_name"
fi

collect_text_files() {
  find "$out_dir" -type f \( \
    -name '*.go' -o \
    -name '*.md' -o \
    -name '*.json' -o \
    -name '*.py' -o \
    -name '*.sh' -o \
    -name '*.yml' -o \
    -name '*.yaml' -o \
    -name 'Makefile' -o \
    -name 'go.mod' \
  \) -print0
}

mapfile -d '' render_text_files < <(collect_text_files)

replace_in_text_files() {
  local find_text="$1"
  local replace_text="$2"

  if [[ "${#render_text_files[@]}" -eq 0 ]]; then
    return 0
  fi

  FIND_TEXT="$find_text" REPLACE_TEXT="$replace_text" perl -0pi -e 's/\Q$ENV{FIND_TEXT}\E/$ENV{REPLACE_TEXT}/g' "${render_text_files[@]}"
}

replace_in_text_files '{{MODULE_NAME}}' "$module_name"
replace_in_text_files '{{MODULE_PATH}}' "$module_path"
replace_in_text_files '{{PACKAGE_NAME}}' "$package_name"
replace_in_text_files 'github.com/ZoneCNH/xlib-standard' "$module_path"
replace_in_text_files 'github.com/ZoneCNH/resiliencx' "$module_path"
replace_in_text_files 'github.com/ZoneCNH/baselib-template' "$module_path"
replace_in_text_files 'xlib-standard' "$module_name"
replace_in_text_files 'resiliencx' "$module_name"
replace_in_text_files 'baselib-template' "$module_name"
package_title="$(printf '%s%s' "$(printf '%s' "${package_name:0:1}" | tr '[:lower:]' '[:upper:]')" "${package_name:1}")"
package_upper="$(printf '%s' "$package_name" | tr '[:lower:]' '[:upper:]')"
replace_in_text_files 'templatex_' "${package_name}_"
replace_in_text_files 'resiliencx_' "${package_name}_"
replace_in_text_files 'Templatex' "$package_title"
replace_in_text_files 'Resiliencx' "$package_title"
replace_in_text_files 'TEMPLATEX' "$package_upper"
replace_in_text_files 'RESILIENCX' "$package_upper"
replace_in_text_files 'templatex' "$package_name"
replace_in_text_files 'resiliencx' "$package_name"

write_governance_lock() {
  cat > "$out_dir/xlib-standard.lock" <<EOF
schema_version: "1.0"
standard_name: "xlib-standard"
standard_repo: "https://github.com/ZoneCNH/xlib-standard"
standard_version: "$standard_version"
standard_commit: "$standard_commit"
module_name: "$module_name"
module_path: "$module_path"
package_name: "$package_name"
layer: "$layer"
adoption_check: "GOWORK=off make adoption-check"
EOF
}

verify_governance_pack() {
  local required=(
    "xlib-standard.lock"
    ".githooks/pre-commit"
    ".githooks/pre-push"
    ".github/workflows/adoption-check.yml"
    ".github/rulesets/protect-main.json"
    "mk/governance.mk"
    ".agent/harness/harness.yaml"
  )
  local path
  for path in "${required[@]}"; do
    if [[ ! -f "$out_dir/$path" ]]; then
      echo "ERROR: --enable-governance render missing $path" >&2
      exit 1
    fi
  done
}

if [[ "$enable_governance" == "1" ]]; then
  write_governance_lock
  verify_governance_pack
else
  rm -f "$out_dir/xlib-standard.lock"
fi

(
  cd "$out_dir"
  gofmt -w ./pkg ./internal ./contracts ./examples ./testkit
)

echo "rendered $module_name at $out_dir"
