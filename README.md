# ocimage

ocimage packages prebuilt artifacts into OCI images and pushes them to OCI/Docker registries without Docker, Podman, Buildah, or Skopeo.

ocimage is intentionally a packaging tool, not a general-purpose container build executor. It does not run build steps and does not install system packages with tools like `apt-get`, `dnf`, or Alpine's `apk`. App builds should start from a base image that already contains the required operating system packages, runtime dependencies, users, and security policy. In many organisations, a platform team owns a small set of maintained base images with a central patching strategy; app teams then use ocimage in CI to copy prebuilt application artifacts into those bases quickly and reproducibly.

This makes ocimage a good fit for app developer workflows that need a small, fast CI dependency with no daemon or local container runtime requirement. It is not intended for every container image use case.

## Install

macOS/Linux via [Homebrew](https://brew.sh/):

```sh
brew install podplane/tap/ocimage
```

or via [Go](https://go.dev/):

```sh
go install github.com/podplane/ocimage@latest
```

## Example: Go application

Go is the simplest ocimage use case: cross-compile the binary first, then use ocimage to package it into one or more OCI images.

```sh
GOOS=linux GOARCH=amd64 go build -trimpath -o bin/linux_amd64/app ./cmd/app
GOOS=linux GOARCH=arm64 go build -trimpath -o bin/linux_arm64/app ./cmd/app
```

Use a packaging-only `Containerfile`:

```dockerfile
FROM scratch

ARG TARGETOS
ARG TARGETARCH

COPY bin/${TARGETOS}_${TARGETARCH}/app /app

ENTRYPOINT ["/app"]
```

Build and push a multi-arch image:

```sh
ocimage build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/octocat/app:v1 \
  --push \
  .
```

## Example: Ruby or Rails application

Ruby applications usually need system packages and native libraries, such as `libpq`, `libvips`, `jemalloc`, or timezone data. ocimage does not install these with `apt-get` or `apk`; instead, use a runtime base image that already contains them.

That base image might be maintained by a platform team:

```dockerfile
FROM ghcr.io/company/ruby-runtime:3.4-slim
```

The base should include the required OS packages, Ruby runtime dependencies, and runtime user, for example `nonroot`. Application CI can then install gems or build assets outside ocimage, place those artifacts in the build context, and use ocimage only for the final packaging layers:

```dockerfile
ARG OCIMAGE_STORE_URL=
FROM ${OCIMAGE_STORE_URL}ruby/base:v3.4

ENV RAILS_ENV=production \
    PORT=8080

WORKDIR /app

COPY --chown=nonroot:nonroot demo/app app
COPY --chown=nonroot:nonroot demo/bin bin
COPY --chown=nonroot:nonroot demo/config config
COPY --chown=nonroot:nonroot demo/db db
COPY --chown=nonroot:nonroot demo/lib lib
COPY --chown=nonroot:nonroot demo/public public
COPY --chown=nonroot:nonroot demo/config.ru demo/Gemfile demo/Gemfile.lock demo/Rakefile ./

CMD ["bundle", "exec", "puma", "-C", "config/puma.rb"]
```

## Zot-compatible local store

The local store is a Zot-compatible collection of OCI image layouts. Set `OCIMAGE_STORE` to control it.

```sh
ocimage build -t ghcr.io/octocat/app:v1 .
zot serve --root "$OCIMAGE_STORE"
```

Zot can then serve the repository `ghcr.io/octocat/app:v1` directly from the store.

Builds also resolve base images from this store before pulling from a registry, so ocimage can package images offline when the required bases are already mirrored into the store. Use `ocimage build --pull` to force a registry pull.

## Docker-compatible image archives

`ocimage save` saves images from the ocimage store as Docker-compatible image archives. It aligns to Docker's `save` command: write to stdout by default, or use `-o`/`--output` to write a tar file.

```sh
ocimage build -t ghcr.io/octocat/app:v1 .
ocimage save ghcr.io/octocat/app:v1 -o app.tar
docker load -i app.tar
```

For multi-platform images, pass `--platform` to save the image variant Docker should load:

```sh
ocimage save --platform linux/arm64 ghcr.io/octocat/app:v1 | docker load
```

## Generate a Containerfile

`ocimage init` detects common language and framework project files and writes a `Containerfile` from the built-in examples. It only auto-generates when detection finds exactly one template:

```sh
ocimage init .
```

If the project cannot be detected or multiple templates match, choose one explicitly:

```sh
ocimage init --list
ocimage init --template typescript-hono-node .
```

Existing files are left untouched unless `--force` is passed.

## Learn More

Learn more about Podplane at the official project website: [podplane.dev](https://podplane.dev)

## License

Podplane is licensed under the Apache License, Version 2.0.
Copyright The Podplane Authors.

See the [LICENSE](./LICENSE) file for details.
