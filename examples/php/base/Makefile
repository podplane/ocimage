# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

ARCH ?= amd64,arm64
IMAGE_PREFIX ?=
IMAGE_NAME ?= php/base
IMAGE_TAG ?= 8.4-frankenphp
STORE_ROOT ?= $(shell if [ -n "$$OCIMAGE_STORE" ]; then printf '%s' "$$OCIMAGE_STORE"; elif [ -n "$$XDG_STATE_HOME" ]; then printf '%s' "$$XDG_STATE_HOME/ocimage/store"; else printf '%s' "$$HOME/.local/state/ocimage/store"; fi)
STORE_IMAGE_NAME ?= index.docker.io/$(IMAGE_NAME)

PLATFORMS := $(shell printf '%s' '$(ARCH)' | tr -d ' ' | sed 's/[^,][^,]*/linux\/&/g')
STORE_REPO := $(STORE_ROOT)/$(STORE_IMAGE_NAME)
IMAGE_REF := $(IMAGE_PREFIX)$(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: setup compile build push

setup:
	@echo "Nothing to do for make target: setup"

compile:
	@echo "Nothing to do for make target: compile"

build:
	@mkdir -p "$(STORE_REPO)"
	docker build \
		--file Containerfile \
		--platform "$(PLATFORMS)" \
		--tag "$(IMAGE_REF)" \
		--output "type=oci,dest=$(STORE_REPO),tar=false" \
		.
	@echo "Stored $(IMAGE_NAME):$(IMAGE_TAG) in $(STORE_REPO)"

push:
	docker build \
		--file Containerfile \
		--platform "$(PLATFORMS)" \
		--tag "$(IMAGE_REF)" \
		--push \
		.
