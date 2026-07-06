# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

APP_DIR ?= demo
MISE_PHP ?= php@8.4
TOOLS_DIR := .tools
COMPOSER := $(CURDIR)/$(TOOLS_DIR)/composer.phar

include ../../Makefile

$(COMPOSER):
	@command -v mise >/dev/null 2>&1 || { echo "mise is required but not installed"; exit 1; }
	mise plugins ls | awk '{ print $$1 }' | grep -qx php || mise plugins install php https://github.com/asdf-community/asdf-php.git
	mise install $(MISE_PHP)
	mkdir -p "$(TOOLS_DIR)"
	mise exec $(MISE_PHP) -- php -r "copy('https://getcomposer.org/installer', '$(TOOLS_DIR)/composer-setup.php');"
	mise exec $(MISE_PHP) -- php "$(TOOLS_DIR)/composer-setup.php" --install-dir="$(TOOLS_DIR)" --filename=composer.phar
	rm "$(TOOLS_DIR)/composer-setup.php"

setup: $(COMPOSER) ## Scaffold the Laravel demo application
	@if [ -d "$(APP_DIR)" ]; then \
		echo "Demo Laravel app already exists. Please delete $(APP_DIR) and re-run make setup"; \
		exit 1; \
	fi
	mise exec $(MISE_PHP) -- php "$(COMPOSER)" create-project laravel/laravel "$(APP_DIR)"
	cp -R template/. "$(APP_DIR)/"
	mise exec $(MISE_PHP) -- php "$(APP_DIR)/artisan" key:generate --ansi
	@echo "To run demo:"
	@echo "make dev"

dev: ## Run the Laravel app locally
	mise exec -C "$(APP_DIR)" $(MISE_PHP) -- php artisan serve --host=0.0.0.0 --port="$(PORT)"

compile: $(COMPOSER) ## Install production dependencies and optimize Laravel
	mise exec $(MISE_PHP) -- php "$(COMPOSER)" install --working-dir="$(APP_DIR)" --no-dev --optimize-autoloader
	mise exec -C "$(APP_DIR)" $(MISE_PHP) -- php artisan config:cache
	mise exec -C "$(APP_DIR)" $(MISE_PHP) -- php artisan route:cache
	mise exec -C "$(APP_DIR)" $(MISE_PHP) -- php artisan view:cache

clean: ## Remove generated demo application
	rm -rf "$(APP_DIR)"
