# Python examples

These examples use Gunicorn as the production Python app server. Django static files are served by Whitenoise so the app image stays single-process and does not need a bundled Caddy, nginx, or process supervisor.

The examples use the platform-base pattern:

- `examples/python/base` represents a platform-team image. It is built with Docker/buildx because it owns the Python runtime and production dependencies. Run `make build` to store it locally as `python/base:3.14`, or `make push IMAGE_PREFIX=registry.example.com/team/` to publish it as `registry.example.com/team/python/base:3.14`.
- `examples/python/django` represents an app-team image. It packages Django source and collected static files with `ocimage`; it does not copy a host-built `.venv` into the image.

The base image is opinionated for PostgreSQL and includes `psycopg[binary]`. Production platforms can replace or extend that dependency set for their supported database drivers.

Typical app workflow:

```sh
cd examples/python/django
make setup
make dev
make build TAG=registry.example.com/team/django-demo:latest
make push TAG=registry.example.com/team/django-demo:latest
```

`make setup`, `make dev`, and `make compile` use `uv` locally. `make compile` runs Django `collectstatic` before `ocimage build` packages the generated app directory.
