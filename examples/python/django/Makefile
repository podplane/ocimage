# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

APP_DIR ?= demo
UV ?= uv

include ../../Makefile

setup: ## Scaffold the Django demo application
	@if [ -d "$(APP_DIR)" ]; then \
		echo "Demo Django app already exists. Please delete $(APP_DIR) and re-run make setup"; \
		exit 1; \
	fi
	mkdir -p "$(APP_DIR)"
	cp template/pyproject.toml "$(APP_DIR)/pyproject.toml"
	cd "$(APP_DIR)" && $(UV) sync
	cd "$(APP_DIR)" && $(UV) run django-admin startproject config .
	cd "$(APP_DIR)" && $(UV) run python manage.py startapp tasks
	cat template/config/settings.py >> "$(APP_DIR)/config/settings.py"
	cp template/config/urls.py "$(APP_DIR)/config/urls.py"
	cp -R template/tasks/. "$(APP_DIR)/tasks/"
	@echo "To run demo:"
	@echo "make dev"

dev: ## Run the Django app locally
	cd "$(APP_DIR)" && $(UV) run python manage.py runserver 0.0.0.0:$(PORT)

compile: ## Collect Django static files
	cd "$(APP_DIR)" && $(UV) run python manage.py collectstatic --noinput

clean: ## Remove generated demo application
	rm -rf "$(APP_DIR)"
