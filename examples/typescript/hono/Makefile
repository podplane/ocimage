# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

APP_DIR ?= demo
RUNTIME ?= node

ifeq ($(RUNTIME),node)
INSTALL = cd "$(APP_DIR)" && npm install
INSTALL_FROZEN = cd "$(APP_DIR)" && npm ci
RUN = npm run
else ifeq ($(RUNTIME),bun)
INSTALL = bun install --cwd "$(APP_DIR)"
INSTALL_FROZEN = bun install --cwd "$(APP_DIR)" --frozen-lockfile
RUN = bun run
else
$(error unsupported RUNTIME "$(RUNTIME)"; use node or bun)
endif

CTRFILE ?= Containerfile.$(RUNTIME)

include ../../Makefile

setup: ## Scaffold the Hono demo application
	@if [ -d "$(APP_DIR)" ]; then \
		echo "Demo app already exists. Please delete $(APP_DIR) and re-run make setup"; \
		exit 1; \
	fi
	cp -R template "$(APP_DIR)"
	$(INSTALL)
	@echo "To run demo:"
	@echo "make dev RUNTIME=$(RUNTIME)"

dev: ## Run the Hono server locally
	cd "$(APP_DIR)" && PORT="$(PORT)" $(RUN) dev:$(RUNTIME)

compile: ## Build the Hono server bundle
	$(INSTALL_FROZEN)
	cd "$(APP_DIR)" && $(RUN) build:$(RUNTIME)

clean: ## Remove generated demo application
	rm -rf "$(APP_DIR)"
