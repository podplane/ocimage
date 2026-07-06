# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

include ../../Makefile

setup: ## Verify dotnet is installed
	@command -v dotnet >/dev/null 2>&1 || { echo "dotnet is required but not installed"; exit 1; }

dev: ## Run the ASP.NET Core API locally
	cd src && ASPNETCORE_URLS="http://0.0.0.0:$(PORT)" dotnet run

compile: ## Publish framework-dependent app files for each target architecture
	@set -e; \
	for arch in $$(printf '%s' '$(ARCH)' | tr ',' ' '); do \
		case "$$arch" in \
			amd64|arm64) ;; \
			*) echo "unsupported architecture: $$arch"; exit 1 ;; \
		esac; \
		out="bin/$$arch"; \
		dotnet publish src/Ocimage.Tasks.Api.csproj \
			--configuration Release \
			--no-self-contained \
			/p:UseAppHost=false \
			--output "$$out"; \
	done

clean: ## Remove dotnet build outputs
	rm -rf bin src/bin src/obj
