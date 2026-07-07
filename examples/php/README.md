# PHP examples

These examples use [FrankenPHP](https://frankenphp.dev/) as the PHP runtime and web server. FrankenPHP is built on Caddy and runs PHP in one foreground process, so the app image does not need a separate Caddy plus PHP-FPM process supervisor.

The examples use the platform-base pattern:

- `examples/php/base` represents a platform-team image. It is built with Docker/buildx because it installs runtime PHP extensions and configures the base server image. Run `make build` to store it locally as `php/base:8.4-frankenphp`, or `make push IMAGE_PREFIX=registry.example.com/team/` to publish it as `registry.example.com/team/php/base:8.4-frankenphp`.
- `examples/php/laravel` represents an app-team image. It packages a prebuilt Laravel app with `ocimage`; it does not run Composer or frontend build steps inside the image.

The base image owns the PHP runtime and extensions. This demo intentionally omits SQLite and includes PostgreSQL and MySQL PDO drivers, matching the production databases this example expects.

The Laravel example requires PHP locally. `make setup` uses `php` from `PATH`, an already-installed `mise` `php@8.4`, or a command passed with `PHP_CMD='path/to/php'`. It downloads Composer into the example's `.tools/` directory, so a global Composer install is not required.

The Laravel image is a local packaging demo, not a production deployment pattern. It sets a fixed demo `APP_KEY` and uses file-backed cache and sessions so the example works with `make run`; do not copy those choices into production. Real deployments should inject `APP_KEY` from a secret manager or Kubernetes Secret, use environment-specific configuration, and use an appropriate external/session/cache store instead of relying on image-local runtime state.

Typical app workflow:

```sh
cd examples/php/laravel
make setup
make dev
make build TAG=registry.example.com/team/laravel-demo:latest
make push TAG=registry.example.com/team/laravel-demo:latest
```

The `make build` target invokes `make compile` as a dependent target, which installs production Composer dependencies and runs Laravel optimization commands before `ocimage build` packages the generated app directory.
