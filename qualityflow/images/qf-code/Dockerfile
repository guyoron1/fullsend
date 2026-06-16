# QualityFlow sandbox image — arm64-native with Go, gopls, and pyright.
#
# Based on the OpenShell community base (arm64) with the same tools as
# fullsend-code (Go, gitleaks) plus LSP language servers (gopls, pyright).
FROM ghcr.io/nvidia/openshell-community/sandboxes/base:latest

USER root

# CA certificates
RUN apt-get update -qq && apt-get install -y -qq ca-certificates curl git && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates 2>/dev/null || true
RUN git config --system http.sslCAInfo /etc/ssl/certs/ca-certificates.crt 2>/dev/null || true

# Go toolchain (arm64)
ARG GO_VERSION=1.24.13
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-arm64.tar.gz" -o /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm /tmp/go.tar.gz

ENV PATH="/usr/local/go/bin:/sandbox/go/bin:${PATH}" \
    GOPATH="/sandbox/go" \
    GOMODCACHE="/sandbox/go/pkg/mod"

# gopls — Go language server for LSP analysis
RUN /usr/local/go/bin/go install golang.org/x/tools/gopls@latest \
    && ln -sf /sandbox/go/bin/gopls /usr/local/bin/gopls

# pyright — Python language server (if npm available)
RUN npm install -g pyright 2>/dev/null \
    || echo "pyright install skipped (npm not available)"

# gitleaks — secret scanner
ARG GITLEAKS_VERSION=8.30.1
RUN curl -fsSL \
      "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_arm64.tar.gz" \
      -o /tmp/gitleaks.tar.gz \
    && tar xzf /tmp/gitleaks.tar.gz -C /usr/local/bin gitleaks \
    && chmod +x /usr/local/bin/gitleaks \
    && rm /tmp/gitleaks.tar.gz

USER sandbox
