# Examples

Each example directory has a `Makefile` so the following workflow is consistent across examples:

```sh
make setup build push TAG=registry.example.com/team/app:latest
```

The examples are intended to show realistic packaging flows across common languages and frameworks:

| Example | Purpose |
| --- | --- |
| [csharp/aspnet-chiseled](./csharp/aspnet-chiseled) | ASP.NET Core minimal API built on a .NET chiseled runtime image |
| [go/scratch-chi](./go/scratch-chi) | Go/chi JSON HTTP API built on `scratch` |
| [typescript/hono](./typescript/hono) | Hono JSON HTTP API built for Node.js or Bun with `RUNTIME` |
| [typescript/fullstack-tanstack-start](./typescript/fullstack-tanstack-start) | Full-stack TanStack Start app built for Node.js or Bun with `RUNTIME` |
| [typescript/spa-tanstack-start](./typescript/spa-tanstack-start) | TanStack Start SPA built with Node.js or Bun and served by Caddy |
| [ruby/ruby-rails](./ruby/ruby-rails) | Full-stack Ruby on Rails web app |
| [php/laravel](./php/laravel) | Full-stack Laravel web app built on FrankenPHP |
| [python/django](./python/django) | Full-stack Django web app built on Gunicorn Python |
| [rust/scratch](./rust/scratch) | Rust/Axum JSON HTTP API built on `scratch` |

Platform base examples, such as [ruby/base](./ruby/base), [php/base](./php/base), and [python/base](./python/base), exist to show how a specific base image can package runtime or dependency work that cannot live cleanly in a final application image built with ocimage.

## Demo application conventions

Full-stack application examples should present a small task/todo app. The UI can be framework-specific, but the app should be similar across frameworks: list tasks, add a task, and keep state in the simplest local/demo-appropriate way.

Backend-only language examples should implement the shared JSON HTTP task API. This keeps examples comparable across runtimes and lets the TanStack SPA act as a frontend for any compatible backend implementation.

The [TanStack SPA](./typescript/spa-tanstack-start) reads its backend from `API_URL` / `VITE_API_URL`, so it can be paired with the Hono, Go, or Rust/Axum examples, or any future backend that follows the same contract.

### Task API contract

Compatible backend examples should listen on port `8080` by default and expose:

#### `GET /`

Returns the current task titles as a JSON array.

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"tasks":[{"title": "Write docs"},{"title": "Ship image"}]}
```

#### `POST /`

Adds a task and returns the updated task list in the result.

```http
POST /
Content-Type: application/json

{"task":{"title":"Ship image"}}
```

Success response:

```http
HTTP/1.1 201 Created
Content-Type: application/json

{"result":{"error":false},"tasks":[{"title":"Ship image"}]}
```

Validation response for an empty or missing title:

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{"result":{"error":true,"message":"Task title is required."}}
```

## Makefile conventions

Application examples should include `../../Makefile` (the [Makefile](./Makefile) in this directory) and use its shared OCI image lifecycle targets:

- `build` packages the example with `ocimage build`.
- `tag` tags an image already stored by `ocimage`; pass `FROM=... TAG=...`.
- `push` publishes `TAG` with `ocimage push`.
- `run` saves the host-platform image from the ocimage store to a Docker-compatible tar, loads it into Docker, and runs it with the shared example convention of exposing port `8080`.

Do not change/override `build`, `tag`, `push`, or `run` in the Makefile of each example. Set variables such as `TAG`, `ARCH`, `PLATFORMS`, `RUN_PLATFORM`, `PORT`, or `TOOL` instead.

Every example must provide these extension targets (even if they simply print no action is required):

- `setup` scaffolds local demo inputs in the example directory.
- `compile` runs language or framework build tools to produce artifacts that `ocimage build` will package.
- `dev` runs the application locally without containers, where that is relevant.

Every example must define both targets, even when one is a no-op. The shared `build` target depends on `compile`, so users usually do not need to run `make compile` separately.

Include `../../Makefile` near the top of the example Makefile, after any variables that need to influence shared defaults. Define `setup` and `compile` in the same Makefile. The repo lint checks fail when a leaf example Makefile omits either target.

TypeScript examples accept `RUNTIME=node` or `RUNTIME=bun` to select the local toolchain and, for server-side examples, the matching `Containerfile.node` or `Containerfile.bun` packaged by ocimage.

## Target boundaries

Keep `setup` local and deterministic. It may scaffold demo app files under the example directory, but it should not build OCI images, push images, install operating system packages, require Docker, or mutate files outside the example directory.

Prefer putting small setup steps directly inline in the example Makefile. Larger scaffold scripts may stay in a local helper script as long as `make setup` remains the preferred entrypoint.

Keep `compile` focused on application artifacts. It may run language or framework build tools such as `go build`, `bun install`, or asset precompilation, but it should not call `ocimage build`, tag images, push images, install toolchains, or require Docker.

Keep `build` reserved for the shared `ocimage build` target.

Platform-base examples that intentionally demonstrate base-image production may be exceptions because they are not packaging-only application examples, for example [ruby/base/Makefile](./ruby/base/Makefile), [php/base/Makefile](./php/base/Makefile), and [python/base/Makefile](./python/base/Makefile).
