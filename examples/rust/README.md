# Rust examples

## scratch

Packages a small Rust HTTP API into a `scratch` image. The app exposes the same task API shape as the Bun/Hono example:

- `GET /` returns a text task list.
- `GET /tasks` returns a JSON array of task titles.
- `POST /tasks` accepts either raw text or JSON such as `{ "title": "Ship it" }`.

The example uses common Rust web crates (`axum`, `tokio`, `serde`, and `tower-http`) and compiles to Linux musl binaries that can run directly from `scratch`.
