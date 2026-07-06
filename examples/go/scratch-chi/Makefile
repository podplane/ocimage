# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

include ../../Makefile

setup: ## Scaffold local demo inputs
	@echo "Nothing to do for make target: setup"

dev: ## Run the Go server locally
	cd src && go run .

compile: ## Build the Go binary for each target architecture
	@for arch in $$(printf '%s' '$(ARCH)' | tr ',' ' '); do \
		out="bin/$$arch"; \
		mkdir -p "$$out"; \
		CGO_ENABLED=0 GOOS=linux GOARCH="$$arch" go build \
			-C src \
			-o "../$$out/main" \
			-trimpath .; \
	done

clean: ## Remove Go build outputs
	rm -rf bin
