# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.DEFAULT_GOAL := help

BINARY_NAME=ocimage
MODULE=github.com/podplane/ocimage
BUILDVARS_PKG=$(MODULE)/internal/buildvars
BUILD_VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT_HASH?=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
COMMIT_DATE?=$(shell git show -s --format=%cI HEAD 2>/dev/null || echo unknown)
COMMIT_BRANCH?=$(shell git branch --show-current 2>/dev/null | sed 's/^$$/unknown/')
LDFLAGS=-s -w -X $(BUILDVARS_PKG).buildVersion=$(BUILD_VERSION) -X $(BUILDVARS_PKG).buildDate=$(BUILD_DATE) -X $(BUILDVARS_PKG).commitHash=$(COMMIT_HASH) -X $(BUILDVARS_PKG).commitDate=$(COMMIT_DATE) -X $(BUILDVARS_PKG).commitBranch=$(COMMIT_BRANCH)

.PHONY: help setup fmt lint precommit test build clean

help: ## Show available targets
	@echo "Usage: make <target>"
	@awk 'BEGIN {FS = ":.*?## "} /^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Verify required tools and enable git hooks
	@command -v go >/dev/null 2>&1 || { echo "go is required but not installed"; exit 1; }
	@echo "All required tools are installed."
	@cp scripts/git-hooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@cp scripts/git-hooks/commit-msg .git/hooks/commit-msg
	@chmod +x .git/hooks/commit-msg
	@echo "Git hooks installed."

fmt: ## Format Go files
	@go fmt ./...

lint: ## Run linter
	@status=0; \
	for file in examples/*/*/Makefile; do \
		[ -f "$$file" ] || continue; \
		if ! awk '/^[[:space:]]*setup[[:space:]]*:/ { found = 1 } END { exit found ? 0 : 1 }' "$$file"; then \
			echo "$$file: missing setup target"; \
			status=1; \
		fi; \
		if ! awk '/^[[:space:]]*compile[[:space:]]*:/ { found = 1 } END { exit found ? 0 : 1 }' "$$file"; then \
			echo "$$file: missing compile target"; \
			status=1; \
		fi; \
	done; \
	exit $$status
	@golangci-lint run --timeout=5m

precommit: ## Run precommit checks
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || { gofmt -l $$(find . -name '*.go' -not -path './.git/*'); exit 1; }
	@$(MAKE) lint

test: ## Run tests
	@go test -v -race ./...

build: ## Build ocimage
	@mkdir -p bin
	@go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY_NAME) .

clean: ## Remove build outputs
	@rm -rf bin
