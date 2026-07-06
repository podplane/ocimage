# C# examples

## aspnet-chiseled

Packages an ASP.NET Core minimal API into a .NET chiseled runtime image. The app exposes the shared task API contract:

- `GET /` returns `{ "tasks": [{ "title": "..." }] }`.
- `POST /` accepts `{ "task": { "title": "..." } }` and returns `{ "result": { "error": false }, "tasks": [...] }`.

The example publishes framework-dependent app files with `dotnet publish`, then packages them with `ocimage` on top of `mcr.microsoft.com/dotnet/aspnet:10.0-noble-chiseled`. This avoids requiring a native Linux linker toolchain on macOS while still keeping the final image small.
