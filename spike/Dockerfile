# Minimal dev image for the M0 spike — just enough to prove ForgeCode -> gateway.
# (Real space images add per-stack layers like jdk21-maven; out of scope for M0.)
FROM debian:12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git jq \
    && rm -rf /var/lib/apt/lists/*

# Install ForgeCode. It installs to $HOME/.local/bin (= /root/.local/bin as root).
# Pin a version with: ... | sh -s -- <version>  (the version is a positional arg).
RUN curl -fsSL https://forgecode.dev/cli | sh
ENV PATH="/root/.local/bin:${PATH}"

COPY verify.sh /usr/local/bin/verify.sh
RUN chmod +x /usr/local/bin/verify.sh

WORKDIR /workspace
