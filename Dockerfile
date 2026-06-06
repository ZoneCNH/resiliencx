# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.23
ARG GOLANGCI_LINT_VERSION=v2.1.6
ARG GOVULNCHECK_VERSION=v1.1.4

FROM golang:${GO_VERSION}-bookworm AS toolchain

ARG GOLANGCI_LINT_VERSION=v2.1.6
ARG GOVULNCHECK_VERSION=v1.1.4

ENV CGO_ENABLED=1 \
    GOWORK=off \
    XLIB_CONTEXT=docker_toolchain \
    GOLANGCI_LINT_VERSION=${GOLANGCI_LINT_VERSION} \
    GOVULNCHECK_VERSION=${GOVULNCHECK_VERSION}

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      bash \
      ca-certificates \
      curl \
      git \
      jq \
      make \
      python3-yaml \
    && git config --global --add safe.directory /workspace \
    && rm -rf /var/lib/apt/lists/*

RUN go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}" \
    && go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}" \
    && golangci-lint version \
    && govulncheck -version

WORKDIR /workspace

CMD ["bash"]
