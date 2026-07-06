# ocimage compatibility

ocimage is an OCI-native image packaging tool. It supports a strict packaging-focused subset of Docker/Containerfile behavior and intentionally does not execute build steps.

The guiding model is:

- use maintained base images that already contain operating system packages, runtime dependencies, users, and security policy
- build application artifacts before invoking ocimage
- use ocimage to copy those artifacts into an OCI image and push or serve them via an OCI-compatible registry such as Zot

## Table of contents

- [Containerfile compatibility](#containerfile-compatibility)
  - [Supported instructions](#supported-instructions)
    - [`FROM`](#from)
    - [`ARG`](#arg)
    - [`ENV`](#env)
    - [`LABEL`](#label)
    - [`WORKDIR`](#workdir)
    - [`COPY`](#copy)
    - [`ENTRYPOINT`](#entrypoint)
    - [`CMD`](#cmd)
  - [Supported `.dockerignore` behavior](#supported-dockerignore-behavior)
  - [Unsupported Containerfile instructions](#unsupported-containerfile-instructions)
    - [`RUN`](#run)
    - [`USER`](#user)
    - [`ADD`](#add)
    - [`EXPOSE`, `VOLUME`, `STOPSIGNAL`, `SHELL`, `ONBUILD`, `HEALTHCHECK`](#expose-volume-stopsignal-shell-onbuild-healthcheck)
- [Command compatibility](#command-compatibility)
  - [`ocimage build`](#ocimage-build)
  - [`ocimage push`](#ocimage-push)
  - [`ocimage tag`](#ocimage-tag)
  - [`ocimage version`](#ocimage-version)
- [Store compatibility](#store-compatibility)
- [Permanently unsupported integrations](#permanently-unsupported-integrations)

## Containerfile compatibility

### Supported instructions

#### `FROM`

Supported:

```dockerfile
FROM scratch
FROM alpine:3.20
FROM ghcr.io/company/runtime-base:latest
FROM ${BASE_IMAGE}
```

Behavior:

- resolves non-`scratch` base image tags from the local ocimage store first
- pulls non-`scratch` base images from registries when not present in the local store, or when `ocimage build --pull` is set
- selects the requested target platform with `ocimage build --platform`
- preserves the base image default user for non-`scratch` bases
- uses `65532:65532` as the default user for `FROM scratch`
- supports variable expansion from preceding `ARG` instructions

Unsupported:

```dockerfile
FROM --platform=linux/amd64 alpine
FROM golang:1.26 AS builder
```

Use `ocimage build --platform ...` instead of `FROM --platform`. Multi-stage builds are unsupported.

#### `ARG`

Supported:

```dockerfile
ARG BASE_IMAGE=alpine:3.20
ARG TARGETOS
ARG TARGETARCH
```

Behavior:

- supports build arguments passed with `--build-arg KEY=VALUE`
- supports default values in `ARG KEY=VALUE`
- supports automatic platform arguments:
  - `TARGETPLATFORM`
  - `TARGETOS`
  - `TARGETARCH`
  - `TARGETVARIANT`
  - `BUILDPLATFORM`
  - `BUILDOS`
  - `BUILDARCH`

#### `ENV`

Supported:

```dockerfile
ENV RAILS_ENV=production
ENV BUNDLE_DEPLOYMENT=1 BUNDLE_WITHOUT=development:test
ENV MESSAGE="hello world"
```

Behavior:

- supports quoted values parsed by the Dockerfile parser
- updates image config environment
- makes values available to later supported variable expansion where Dockerfile semantics require it

#### `LABEL`

Supported:

```dockerfile
LABEL org.opencontainers.image.source="https://github.com/example/app"
LABEL description="hello world"
```

Behavior:

- supports quoted values parsed by the Dockerfile parser
- updates image config labels
- CLI labels from `ocimage build --label KEY=VALUE` are applied after Containerfile labels

#### `WORKDIR`

Supported:

```dockerfile
WORKDIR /app
WORKDIR relative/path
```

Behavior:

- updates image config working directory
- affects relative COPY destinations

#### `COPY`

Supported:

```dockerfile
COPY app /app
COPY bin/ /app/bin/
COPY vendor/bundle /usr/local/bundle
COPY --chown=1000:1000 . .
COPY --chown=app:app . .
COPY --chmod=755 bin/app /app
```

Behavior:

- creates one OCI layer per `COPY` instruction
- supports file copies
- supports directory copies
- supports directory-content copy semantics, e.g. `COPY dir /app/`
- supports multiple sources when destination ends with `/`
- supports glob patterns through Go filepath glob behavior
- rejects sources outside the build context
- preserves symlinks without following them out of the context
- defaults copied file ownership to `0:0`
- supports `--chmod`
- supports numeric `--chown`
- supports named `--chown` when the base image contains matching `/etc/passwd` and `/etc/group` data
- supports `--chown=nonroot:nonroot` with `FROM scratch`, resolving to `65532:65532`
- handles relevant OCI whiteouts for `/etc/passwd` and `/etc/group` while resolving named `--chown`
- emits parent directory entries for copied paths

Unsupported:

```dockerfile
COPY --from=builder /app /app
```

`COPY --from` is unsupported because multi-stage builds are unsupported.

Known limitations:

- named `--chown` with `FROM scratch` only supports the built-in `nonroot:nonroot` identity; use numeric IDs for anything else
- named `--chown` only resolves static `/etc/passwd` and `/etc/group` data from base image layers
- runtime identity providers such as NSS, LDAP, or generated users are unsupported
- Windows ownership semantics are unsupported

#### `ENTRYPOINT`

Supported:

```dockerfile
ENTRYPOINT ["/app"]
ENTRYPOINT /app
```

Behavior:

- supports JSON exec form
- supports shell form
- shell form is stored as `/bin/sh -c ...`, matching Docker image config behavior

#### `CMD`

Supported:

```dockerfile
CMD ["bundle", "exec", "puma", "-C", "config/puma.rb"]
CMD bundle exec puma
```

Behavior:

- supports JSON exec form
- supports shell form
- shell form is stored as `/bin/sh -c ...`, matching Docker image config behavior

### Supported `.dockerignore` behavior

Supported:

- root `.dockerignore`
- Dockerfile-specific ignore files such as:
  - `Dockerfile.dockerignore`
  - `Containerfile.dockerignore`
  - `docker/Containerfile.dockerignore`
- comments
- blank lines
- cleaned paths
- leading `/` patterns
- `**` patterns
- negation with `!`
- re-including files below ignored directories
- skipping ignored directories when no negation rules are present

Behavior is implemented with Moby's `ignorefile` and `patternmatcher` packages.

### Unsupported Containerfile instructions

The following instructions are intentionally unsupported and produce friendly errors:

```dockerfile
ADD
RUN
USER
EXPOSE
VOLUME
STOPSIGNAL
SHELL
ONBUILD
HEALTHCHECK
```

#### `RUN`

Unsupported:

```dockerfile
RUN apt-get update && apt-get install -y libpq5
RUN apk add --no-cache tzdata
```

ocimage does not execute commands. Use a base image that already contains the required system packages and runtime dependencies.

#### `USER`

Unsupported:

```dockerfile
USER app
```

Behavior:

- non-`scratch` images preserve the base image default user
- `FROM scratch` defaults to `65532:65532`

Use a base image with the desired default runtime user already configured.

#### `ADD`

Unsupported:

```dockerfile
ADD archive.tar.gz /app
ADD https://example.com/file /file
```

Use `COPY` with pre-fetched and pre-extracted artifacts.

#### `EXPOSE`, `VOLUME`, `STOPSIGNAL`, `SHELL`, `ONBUILD`, `HEALTHCHECK`

These image metadata instructions are unsupported in ocimage's packaging subset.

## Command compatibility

## `ocimage build`

Supported form:

```sh
ocimage build [OPTIONS] PATH
```

Supported flags:

```text
-t, --tag KEY
-f, --file PATH
--platform PLATFORM[,PLATFORM...]
--build-arg KEY=VALUE
--label KEY=VALUE
--push
--pull
--sbom[=true|false]
```

Behavior:

- writes builds to the ocimage OCI store
- uses `OCIMAGE_STORE` when set
- otherwise uses a user-local default store path
- produces a single image for one platform
- produces an OCI image index for multiple platforms
- can push after building with `--push`
- can build offline when referenced base images are already present in the ocimage store
- uses locally stored base images by default and uses `--pull` to force a registry pull
- supports `--sbom=true` by invoking the `syft` binary from `PATH`, generating SPDX JSON, and storing it as an OCI subject referrer artifact
- pushes stored SBOM referrers when pushing the subject image

Examples:

```sh
ocimage build -t ghcr.io/octocat/app:v1 .
ocimage build -f Containerfile -t ghcr.io/octocat/app:v1 .
ocimage build --platform linux/amd64,linux/arm64 -t ghcr.io/octocat/app:v1 .
ocimage build --build-arg VERSION=1.2.3 --label org.opencontainers.image.version=1.2.3 -t ghcr.io/octocat/app:v1 .
ocimage build --push -t ghcr.io/octocat/app:v1 .
ocimage build --sbom -t ghcr.io/octocat/app:v1 .
```

Unsupported flags:

```text
--target
--no-cache
--provenance
-o, --output
```

Notes:

- `--target` is unsupported because multi-stage builds are unsupported
- `--no-cache` is unsupported because ocimage does not execute build steps or maintain a build cache
- `--provenance` is not supported yet
- `--sbom` only supports boolean Docker-compatible values; generator options are unsupported
- `--sbom=true` requires `syft` at runtime and errors if `syft` is unavailable
- `--output` is intentionally unsupported for now; builds are stored in the OCI/Zot-compatible local store

## `ocimage push`

Supported form:

```sh
ocimage push NAME[:TAG]
```

Behavior:

- reads `NAME[:TAG]` from the ocimage OCI store
- pushes the stored image or image index to the remote registry reference
- uses go-containerregistry authentication, including Docker config and credential helpers
- supports OCI-compatible registries and Docker Registry HTTP API v2-compatible registries

Example:

```sh
ocimage push ghcr.io/octocat/app:v1
```

Unsupported behavior:

- pushing from Docker's local daemon image store
- pushing from Podman's containers-storage image store
- pushing arbitrary OCI layout paths or tar archives directly

## `ocimage tag`

Supported form:

```sh
ocimage tag SOURCE_IMAGE TARGET_IMAGE
```

Behavior:

- creates another local store reference to the same image or image index
- stores tags as OCI `org.opencontainers.image.ref.name` annotations in the repository `index.json`
- copies blobs across repository layouts when tagging across repositories

Examples:

```sh
ocimage tag ghcr.io/octocat/app:v1 ghcr.io/octocat/app:latest
ocimage tag ghcr.io/octocat/app:v1 ghcr.io/octocat/app-copy:v1
```

Unsupported behavior:

- tagging images in Docker's local daemon image store
- tagging images in Podman's containers-storage image store

## `ocimage version`

Supported form:

```sh
ocimage version
```

Behavior:

- prints the linker-injected version and commit hash

## Store compatibility

ocimage's local store is a Zot-compatible collection of per-repository OCI image layouts.

For:

```sh
OCIMAGE_STORE=/tmp/ocimage-store ocimage build -t ghcr.io/octocat/app:v1 .
```

ocimage writes:

```text
/tmp/ocimage-store/ghcr.io/octocat/app/
├── oci-layout
├── index.json
└── blobs/
```

This can be served directly by Zot when Zot's storage root points at `OCIMAGE_STORE`:

```sh
zot serve --root "$OCIMAGE_STORE"
```

Clients then pull from the Zot registry host using the stored repository path, for example:

```sh
docker pull localhost:5000/ghcr.io/octocat/app:v1
```

## Permanently unsupported integrations

The following integrations are intentionally not planned for ocimage:

- Docker daemon local image store interoperability
- Podman containers-storage interoperability
- Skopeo-style containers-storage backends

ocimage uses its own OCI/Zot-compatible local store instead.
