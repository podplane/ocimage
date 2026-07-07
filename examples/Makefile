# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

CURRENT_DIR := $(notdir $(CURDIR))
PARENT_DIR := $(notdir $(patsubst %/,%,$(dir $(CURDIR))))
TOOL ?= ocimage
TAG ?= $(PARENT_DIR)-$(CURRENT_DIR)
FROM ?=
PORT ?= 8080
CTRFILE ?=
ARCH ?= amd64,arm64
PLATFORMS ?= $(shell printf '%s' '$(ARCH)' | tr -d ' ' | sed 's/[^,][^,]*/linux\/&/g')
RUN_PLATFORM ?= linux/$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

.DEFAULT_GOAL := help

.PHONY: help build tag push run

help: ## Show available targets
	@echo "Usage: make <target>"
	@awk 'BEGIN {FS = ":.*?## "} /^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5)} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: compile ## Build the OCI image
	$(TOOL) build $(if $(CTRFILE),-f $(CTRFILE),) --platform $(PLATFORMS) -t $(TAG) .

tag: ## Tag FROM as TAG
	@test -n "$(FROM)" || { echo "FROM is required, for example: make tag FROM=local-demo TAG=registry.example.com/team/app:latest"; exit 1; }
	$(TOOL) tag $(FROM) $(TAG)

run: ## Run the built image
	tmp=$$(mktemp) && \
		trap 'rm -f "$$tmp"' EXIT && \
		$(TOOL) save --platform $(RUN_PLATFORM) -o "$$tmp" $(TAG) && \
		docker load -i "$$tmp" && \
		printf '\nRunning %s on http://localhost:%s\n' '$(TAG)' '$(PORT)' && \
		docker run --rm -p $(PORT):8080 $(TAG)

push: ## Push the built image
	$(TOOL) push $(TAG)
