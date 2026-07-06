# ocimage — Agent Development Guide

## Important

- **Always use `make build`** to build the CLI for release/confidence checks. The Makefile injects version and commit metadata with linker flags.
- **Prefer Makefile targets** over raw tool commands for full-repo validation.
- Before editing, check `git status --short` and do not overwrite or revert other changes.
- ocimage is a packaging-only OCI image tool, with no runtime dependencies and no daemon or special permissions required. Do not add behavior that runs build steps, installs OS packages, or depends on Docker/Podman/Buildah/Skopeo unless explicitly requested.

## Build & Test Commands

- **Setup**: `make setup` — verify required tools and install git hooks.
- **Build**: `make build` — build `bin/ocimage` with version metadata.
- **Test**: `make test` — run all tests with the race detector.
- **Format**: `make fmt` — format Go source files.
- **Lint**: `make lint` — run golangci-lint.
- **Precommit**: `make precommit` — check formatting and run lint.
- **Clean**: `make clean` — remove `bin/`.
- **Focused tests**: `go test ./pkg/build ./pkg/containerfile` or another specific package list is acceptable for targeted iteration.

## CLI Command Guidance

- Keep `internal/cmd` focused on Cobra command wiring, flag parsing, output, and orchestration.
- Put reusable image-building, parsing, storage, or registry behavior in domain packages under `pkg/` or narrowly scoped `internal/` packages.
- Preserve the core contract: builds package prebuilt files from a context into OCI images and optionally push them; they should not execute Containerfile `RUN`-style build steps.
- Keep command output usable in CI logs. Avoid interactive prompts or terminal-only UI unless explicitly requested.
- Prefer explicit flags and clear errors for missing tags, unsupported Containerfile features, invalid platforms, and registry/store failures.

## Containerfile & OCI Behavior

- Treat supported Containerfile syntax as an explicit compatibility surface. When adding or changing syntax support, update parser tests and user-facing docs.
- Maintain Dockerfile/Containerfile semantics only where ocimage intentionally supports them. Unsupported build-executor features should fail clearly instead of being silently ignored.
- Keep local store behavior compatible with OCI image format and Zot Registry. Changes to image layout, tag handling, or push/pull behavior need focused tests in `pkg/store` or `pkg/build`.
- Avoid introducing daemon, runtime, or host-specific assumptions. ocimage should continue to work without Docker or a local container runtime.

## Code Style

- **File headers**: include the Podplane copyright and SPDX header on Go files.
- **Imports**: group imports as standard library -> third-party -> local (`github.com/podplane/ocimage/*`).
- **Naming**: use concise Go names. Avoid unnecessary `Get` prefixes.
- **Errors**: return early and wrap errors with `fmt.Errorf("...: %w", err)` when adding context.
- **Comments**: exported functions, types, constants, and variables should have comments beginning with the exported name.
- **Dependencies**: do not add new Go module dependencies without explicit confirmation.
- Prefer small, direct functions over one-use abstractions. Add helpers only when they clarify a real responsibility or are reused.

## Testing

- Use the standard `testing` package.
- Prefer focused package tests while iterating, then broader checks when touching shared behavior.
- Put tests in the package they exercise unless black-box behavior is important.
- Use hand-written fakes, temporary directories, and existing test hooks; avoid mock libraries.
- Use `t.Helper()` for helpers and `t.Cleanup()` for teardown.
- For filesystem, tar, OCI layout, and registry behavior, test observable outputs rather than implementation details where practical.

## Directory Structure

```text
internal/buildvars/     # Build/version metadata injected by the Makefile
internal/cmd/           # Cobra command wiring for build, push, tag, version
pkg/build/              # OCI image packaging, platform handling, push orchestration
pkg/containerfile/      # Supported Containerfile parser and instruction types
pkg/store/              # Zot-compatible local OCI image store helpers
examples/               # Example app packaging workflows and Containerfiles
scripts/git-hooks/      # Repository git hooks installed by make setup
bin/                    # Local build output, generated and ignored
```
