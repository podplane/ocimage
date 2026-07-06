# Ruby examples

Ruby is an awkward fit for `ocimage`: Rails dependencies often include native extensions, and `ocimage` packages files into images without running build steps. That works well for languages with straightforward cross-compilation, but Ruby app teams cannot reliably produce a multi-arch `vendor/bundle` from a single-arch build environment.

These examples use the platform-base pattern:

- `examples/ruby/base` represents a platform-team image. It is built with standard Docker/buildx on infrastructure that can build multiple architectures. Run `make build IMAGE_PREFIX=registry.example.com/team/`; the image is published as `registry.example.com/team/ruby/base:v3.4`, uses `reg.mini.dev/ruby:v3.4` as its runtime base, preloads the common Rails bundle for this example, and runs as the `nonroot` user.
- `examples/ruby/ruby-rails` represents an app-team image. It uses the platform Ruby image as its base via optional `OCIMAGE_STORE_URL`, and only copies app code/assets with `ocimage`; it does not run `bundle install` in the app image and does not copy a platform-specific `vendor/bundle`.

This approach makes the app image multi-arch because every architecture-specific Ruby/native dependency is already solved by the platform base image. The tradeoff is that app Gemfiles must stay within the dependency set provided by the base. If an app needs arbitrary native gems, the platform base must be updated, or the app needs a per-architecture Linux build step outside `ocimage`. Note that the Rails example precompiles assets before packaging and disables Active Storage image variants, so it does not need image-processing libraries at runtime.
