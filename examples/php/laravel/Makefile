# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

APP_DIR ?= demo
MISE_PHP ?= php@8.4
TOOLS_DIR := .tools
COMPOSER := $(CURDIR)/$(TOOLS_DIR)/composer.phar
SYSTEM_PHP := $(shell command -v php 2>/dev/null)
MISE_PHP_CMD := $(shell command -v mise >/dev/null 2>&1 && mise where $(MISE_PHP) >/dev/null 2>&1 && printf 'mise exec $(MISE_PHP) -- php')
PHP_CMD ?= $(if $(SYSTEM_PHP),$(SYSTEM_PHP),$(MISE_PHP_CMD))

include ../../Makefile

.PHONY: check-php

check-php:
	@test -n "$(PHP_CMD)" || { echo "PHP is required. Install PHP, or pass PHP_CMD='path/to/php'."; exit 1; }

$(COMPOSER): check-php
	mkdir -p "$(TOOLS_DIR)"
	$(PHP_CMD) -r "copy('https://getcomposer.org/installer', '$(TOOLS_DIR)/composer-setup.php');"
	$(PHP_CMD) "$(TOOLS_DIR)/composer-setup.php" --install-dir="$(TOOLS_DIR)" --filename=composer.phar
	rm "$(TOOLS_DIR)/composer-setup.php"

setup: $(COMPOSER) ## Scaffold the Laravel demo application
	@if [ -d "$(APP_DIR)" ]; then \
		echo "Demo Laravel app already exists. Please delete $(APP_DIR) and re-run make setup"; \
		exit 1; \
	fi
	$(PHP_CMD) "$(COMPOSER)" create-project laravel/laravel "$(APP_DIR)"
	cp -R template/. "$(APP_DIR)/"
	$(PHP_CMD) "$(APP_DIR)/artisan" key:generate --ansi
	@echo "To run demo:"
	@echo "make dev"

dev: check-php ## Run the Laravel app locally
	cd "$(APP_DIR)" && $(PHP_CMD) artisan serve --host=0.0.0.0 --port="$(PORT)"

compile: $(COMPOSER) ## Install production dependencies and optimize Laravel
	$(PHP_CMD) "$(COMPOSER)" install --working-dir="$(APP_DIR)" --no-dev --optimize-autoloader
	cd "$(APP_DIR)" && $(PHP_CMD) artisan optimize:clear

clean: ## Remove generated demo application
	rm -rf "$(APP_DIR)"
