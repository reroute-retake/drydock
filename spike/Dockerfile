# Minimal dev image for the M0 spike — just enough to prove ForgeCode -> gateway.
# (Real space images add per-stack layers like jdk21-maven; out of scope for M0.)
FROM debian:12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git jq \
    && rm -rf /var/lib/apt/lists/*

# Install ForgeCode. Pin a version in real builds; non-fatal here so the gateway
# checks still run even if the harness install changes — verify.sh reports status.
RUN curl -fsSL https://forgecode.dev/cli | sh || true
ENV PATH="/root/.forge/bin:${PATH}"

COPY verify.sh /usr/local/bin/verify.sh
RUN chmod +x /usr/local/bin/verify.sh

WORKDIR /workspace
