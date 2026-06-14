# drydock Makefile. Recipes use ;-form (no tabs required).
BIN ?= $(HOME)/.local/bin
MODULE := github.com/reroute-retake/drydock
VERSION ?= $(shell git describe --tags --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
  -X $(MODULE)/internal/version.Version=$(VERSION) \
  -X $(MODULE)/internal/version.Commit=$(COMMIT) \
  -X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build install dev test vet fmt doctor spike spike-down integration release-check snapshot clean

build: ; go build -ldflags "$(LDFLAGS)" -o bin/dock ./cmd/dock

# Install to a USER-OWNED PATH dir so self-update/install need no root (11A).
install: ; go build -ldflags "$(LDFLAGS)" -o $(BIN)/dock ./cmd/dock && echo "installed dock -> $(BIN)/dock"

# Drydock-in-drydock inner loop: build the dev binary as dock-dev (11A).
dev: ; go build -ldflags "$(LDFLAGS)" -o $(BIN)/dock-dev ./cmd/dock && echo "installed dock-dev -> $(BIN)/dock-dev"

test: ; go test ./...

vet: ; go vet ./...

fmt: ; gofmt -w .

doctor: ; go run ./cmd/dock doctor

# The M0 proof: bring up the LiteLLM gateway + dev container, then verify routing
# to one hosted and one local model. Requires Docker + spike/.env (see spike/).
spike: ; cd spike && docker compose up -d --build && docker compose exec dev bash /usr/local/bin/verify.sh

spike-down: ; cd spike && docker compose down -v

# Real Docker-host integration test (builds jdk21-maven, gateway routing, telemetry).
integration: ; bash test/integration.sh

release-check: ; goreleaser check

snapshot: ; goreleaser release --snapshot --clean

clean: ; rm -rf bin
