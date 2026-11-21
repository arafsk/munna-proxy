SHELL := /bin/bash

BIN_DIR := bin
DIST_DIR := dist

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/arafsk/munna-proxy/internal/version.Version=$(VERSION) \
	-X github.com/arafsk/munna-proxy/internal/version.Commit=$(COMMIT) \
	-X github.com/arafsk/munna-proxy/internal/version.Date=$(DATE)

GO ?= go
GOFLAGS ?=

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build all binaries into ./bin
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/munna ./cmd/munna
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/munna-relay ./cmd/munna-relay
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/munna-helper ./cmd/munna-helper

.PHONY: run
run: ## Run the CLI
	$(GO) run ./cmd/munna

.PHONY: test
test: ## Run unit tests with the race detector
	$(GO) test -race -count=1 ./...

.PHONY: cover
cover: ## Write a coverage profile
	$(GO) test -race -count=1 -coverprofile=coverage.out ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: fmt
fmt: ## Format Go sources
	gofmt -w $$(git ls-files --cached --others --exclude-standard '*.go')

.PHONY: fmt-check
fmt-check: ## Fail if any Go file is unformatted
	@out="$$(gofmt -l $$(git ls-files --cached --others --exclude-standard '*.go'))"; \
	if [[ -n "$$out" ]]; then echo "unformatted files:"; echo "$$out"; exit 1; fi

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: ci
ci: fmt-check vet lint test ## Reproduce the CI checks locally

.PHONY: cross
cross: ## Cross-compile every release target
	@set -euo pipefail; \
	for target in darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 linux/amd64 linux/arm64; do \
		os=$${target%/*}; arch=$${target#*/}; \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 $(GO) build ./...; \
	done

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out

.PHONY: snapshot
snapshot: ## Build a local goreleaser snapshot
	goreleaser release --snapshot --clean
