# Podplane <https://podplane.dev>
# Copyright The Podplane Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: setup compile dev clean

CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER ?= rust-lld
CARGO_TARGET_AARCH64_UNKNOWN_LINUX_MUSL_LINKER ?= rust-lld

export CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER
export CARGO_TARGET_AARCH64_UNKNOWN_LINUX_MUSL_LINKER

include ../../Makefile

setup: ## Verify Rust targets are installed
	@command -v cargo >/dev/null 2>&1 || { echo "cargo is required but not installed"; exit 1; }
	@command -v rustup >/dev/null 2>&1 || { echo "rustup is required but not installed"; exit 1; }
	@for arch in $$(printf '%s' '$(ARCH)' | tr ',' ' '); do \
		case "$$arch" in \
			amd64) target=x86_64-unknown-linux-musl ;; \
			arm64) target=aarch64-unknown-linux-musl ;; \
			*) echo "unsupported architecture: $$arch"; exit 1 ;; \
		esac; \
		rustup target list --installed | grep -qx "$$target" || rustup target add "$$target"; \
	done

dev: ## Run the Rust API server locally
	cd src && PORT="$(PORT)" cargo run

compile: ## Build the Rust binary for each target architecture
	@for arch in $$(printf '%s' '$(ARCH)' | tr ',' ' '); do \
		case "$$arch" in \
			amd64) target=x86_64-unknown-linux-musl ;; \
			arm64) target=aarch64-unknown-linux-musl ;; \
			*) echo "unsupported architecture: $$arch"; exit 1 ;; \
		esac; \
		out="bin/$$arch"; \
		mkdir -p "$$out"; \
		cargo build --manifest-path src/Cargo.toml --release --target "$$target"; \
		cp "src/target/$$target/release/ocimage-rust-scratch" "$$out/main"; \
	done

clean: ## Remove Rust build outputs
	rm -rf bin src/target
