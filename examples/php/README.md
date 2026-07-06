# PHP examples

These examples use [FrankenPHP](https://frankenphp.dev/) as the PHP runtime and web server. FrankenPHP is built on Caddy and runs PHP in one foreground process, so the app image does not need a separate Caddy plus PHP-FPM process supervisor.

The examples use the platform-base pattern:

- `examples/php/base` represents a platform-team image. It is built with Docker/buildx because it installs runtime PHP extensions and configures the base server image. Run `make build IMAGE_PREFIX=registry.example.com/team/`; the image is published as `registry.example.com/team/php/base:8.4-frankenphp`.
- `examples/php/laravel` represents an app-team image. It packages a prebuilt Laravel app with `ocimage`; it does not run Composer or frontend build steps inside the image.

The base image owns the PHP runtime and extensions. This demo intentionally omits SQLite and includes PostgreSQL and MySQL PDO drivers, matching the production databases this example expects.

The Laravel example uses [mise](https://mise.jdx.dev/) to install and run PHP locally. `make setup` downloads Composer into the example's `.tools/` directory and runs it with `mise exec php@8.4`, so a global Composer install is not required.

Typical app workflow:

```sh
cd examples/php/laravel
make setup
make dev
make build TAG=registry.example.com/team/laravel-demo:latest
make push TAG=registry.example.com/team/laravel-demo:latest
```

The `make build` target invokes `make compile` as a dependent target, which installs production Composer dependencies and runs Laravel optimization commands before `ocimage build` packages the generated app directory.
