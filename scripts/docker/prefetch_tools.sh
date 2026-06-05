#!/usr/bin/env bash
# scripts/docker/prefetch_tools.sh
# 提前下载所有 Docker 构建依赖到本地缓存目录
set -euo pipefail

GOLANGCI_VERSION="v2.1.6"
GOVULNCHECK_VERSION="v1.1.4"
CACHE_DIR="${DOCKER_TOOL_CACHE:-.cache/docker-tools}"

mkdir -p "${CACHE_DIR}/bin"

# 设置 Go 代理（优先中国镜像）
export GOPROXY="${GOPROXY:-https://goproxy.cn,https://proxy.golang.org,direct}"

# ── golangci-lint ──────────────────────────────────────────
GOLANGCI_BIN="${CACHE_DIR}/bin/golangci-lint"
if [[ -x "${GOLANGCI_BIN}" ]] && "${GOLANGCI_BIN}" version 2>/dev/null | grep -q "${GOLANGCI_VERSION}"; then
  echo "[skip] golangci-lint ${GOLANGCI_VERSION} already cached"
else
  echo "[fetch] golangci-lint ${GOLANGCI_VERSION}..."
  GOPROXY="${GOPROXY:-https://goproxy.cn,https://proxy.golang.org,direct}" \
    GOBIN="$(cd "${CACHE_DIR}/bin" && pwd)" \
    go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_VERSION}"
  echo "[done] golangci-lint ${GOLANGCI_VERSION}"
fi

# ── govulncheck ────────────────────────────────────────────
GOVULNCHECK_BIN="${CACHE_DIR}/bin/govulncheck"
if [[ -x "${GOVULNCHECK_BIN}" ]] && "${GOVULNCHECK_BIN}" -version 2>/dev/null | grep -q "${GOVULNCHECK_VERSION}"; then
  echo "[skip] govulncheck ${GOVULNCHECK_VERSION} already cached"
else
  echo "[fetch] govulncheck ${GOVULNCHECK_VERSION}..."
  GOBIN="$(cd "${CACHE_DIR}/bin" && pwd)" \
    go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"
  echo "[done] govulncheck ${GOVULNCHECK_VERSION}"
fi

# ── 校验 ────────────────────────────────────────────────────
echo ""
echo "=== Cached tools ==="
ls -lh "${CACHE_DIR}/bin/"
echo ""
echo "Cache directory: ${CACHE_DIR}"
