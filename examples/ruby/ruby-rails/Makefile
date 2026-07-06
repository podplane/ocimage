# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

APP_DIR ?= demo

include ../../Makefile

setup: ## Scaffold the Rails demo application
	APP_DIR="$(APP_DIR)" ./setup.sh

dev: ## Run the Rails server locally
	mise exec -C "$(APP_DIR)" ruby@3 -- bin/rails server -p "$(PORT)"

compile: ## Build Rails assets for packaging
	# ensure host is able to run Rails tasks
	mise exec -C "$(APP_DIR)" ruby@3 -- bundle config set path .bundle/build
	mise exec -C "$(APP_DIR)" ruby@3 -- bundle config set without development:test
	mise exec -C "$(APP_DIR)" ruby@3 -- bundle install
	# produce public/assets for container image
	RAILS_ENV=production SECRET_KEY_BASE_DUMMY=1 mise exec -C "$(APP_DIR)" ruby@3 -- bin/rails assets:precompile

clean: ## Remove generated demo application
	rm -rf "$(APP_DIR)"
