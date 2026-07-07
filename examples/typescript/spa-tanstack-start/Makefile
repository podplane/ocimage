# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

APP_DIR ?= demo
API_URL ?= http://localhost:8080
RUNTIME ?= node

ifeq ($(RUNTIME),node)
INSTALL = cd "$(APP_DIR)" && npm install
INSTALL_FROZEN = cd "$(APP_DIR)" && npm ci
RUN = npm run
RUN_DEV = npm run dev --
else ifeq ($(RUNTIME),bun)
INSTALL = bun install --cwd "$(APP_DIR)"
INSTALL_FROZEN = bun install --cwd "$(APP_DIR)" --frozen-lockfile
RUN = bun run
RUN_DEV = bun run dev --
else
$(error unsupported RUNTIME "$(RUNTIME)"; use node or bun)
endif

include ../../Makefile

setup: ## Scaffold the TanStack SPA demo application
	@if [ -d "$(APP_DIR)" ]; then \
		echo "Demo app already exists. Please delete $(APP_DIR) and re-run make setup"; \
		exit 1; \
	fi
	cp -R template "$(APP_DIR)"
	$(INSTALL)
	@echo "To run demo:"
	@echo "make dev RUNTIME=$(RUNTIME)"

dev: ## Run the TanStack SPA locally
	cd "$(APP_DIR)" && VITE_API_URL="$(API_URL)" $(RUN_DEV) --port "$(PORT)"

compile: ## Build the TanStack SPA
	$(INSTALL_FROZEN)
	cd "$(APP_DIR)" && VITE_API_URL="$(API_URL)" $(RUN) build

clean: ## Remove generated demo application
	rm -rf "$(APP_DIR)"
