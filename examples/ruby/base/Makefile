# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

ARCH ?= amd64,arm64
IMAGE_PREFIX ?=
IMAGE_NAME ?= ruby/base
IMAGE_TAG ?= v3.4

PLATFORMS := $(shell printf '%s' '$(ARCH)' | tr -d ' ' | sed 's/[^,][^,]*/linux\/&/g')
ARCH_COUNT := $(shell printf '%s' '$(ARCH)' | tr ',' '\n' | awk 'NF { count++ } END { print count + 0 }')
OUTPUT := $(if $(filter 1,$(ARCH_COUNT)),--load,--push)

.PHONY: setup compile build

setup:
	@echo "Nothing to do for make target: setup"

compile:
	@echo "Nothing to do for make target: compile"

build:
	docker build \
		--file Containerfile \
		--platform "$(PLATFORMS)" \
		--tag "$(IMAGE_PREFIX)$(IMAGE_NAME):$(IMAGE_TAG)" \
		$(OUTPUT) \
		.
