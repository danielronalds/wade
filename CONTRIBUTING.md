# Contributing to WADE

## Development setup

WADE uses [mise](https://mise.jdx.dev/) to manage its Go and Node.js toolchain.

```sh
git clone https://github.com/danielronalds/wade.git
cd wade
mise install
mise run dev
```

Open <http://editor-dev.localhost:8090>.

`mise run dev` installs frontend dependencies, builds the embedded web assets and
starts Air for live reload. Development mode sets `WADE_DEV=1`, which takes
precedence over an inherited `WADE_ADDR`.

## Architecture

WADE has a Go backend and a Vue 3 and TypeScript frontend:

```text
cmd/wade/                 CLI entry point and controllers
internal/                 Backend application
web/src/                  Vue application
internal/web/.dist/       Generated embedded frontend
docs/                     Architecture documentation
```

Before changing the backend, start with
[`docs/backend-architecture.md`](docs/backend-architecture.md).

## Frontend organisation

Frontend code is organised by ownership:

```text
web/src/
  api/          Generated client and shared HTTP behaviour
  components/   App-wide reusable components
  composables/  App-wide reusable composables
  features/     Cross-route workflows
  stores/       App-wide Pinia state
  utils/        Pure app-wide helpers
  views/        Route screens and route-private UI
```

Use the `@/*` import alias. Keep route-private code with its route, cross-route
workflows in `features`, and only genuinely reusable code in the app-wide
folders. Use generated Orval functions directly for API calls and keep shared
HTTP behaviour in `web/src/api/httpClient.ts` and `web/src/api/http.ts`.

## Generated files

The frontend is bundled with esbuild and embedded into the Go binary. There is
no separate Vite development server.

```sh
mise run build
```

This generates the OpenAPI specification and embedded frontend, then writes the
binary to `.tmp/wade`.

After changing API annotations, regenerate and check the API artefacts:

```sh
mise run gen:openapi
npm --prefix web run gen:api
mise run lint:openapi
```

Commit changes to `internal/openapi/swagger.json`,
`internal/openapi/swagger.yaml` and `web/src/api/generated/wade.ts` when their
sources change. Do not commit `web/node_modules` or `internal/web/.dist`.

## Validation

Run the test task before opening a pull request:

```sh
mise run test
```

Use `mise run fmt` to format Go and frontend files. The complete required checks
are defined in [`.github/workflows/ci.yml`](.github/workflows/ci.yml).

Changes to daemon lifecycle behaviour also require an isolated smoke test using
a temporary `XDG_STATE_HOME` and port. Exercise `wade server`, `wade status` and
`wade stop`; confirm foreground mode creates no control socket and no stale WADE
process remains.

## Pull requests

Keep pull requests focused and briefly describe the change in the pull request
body.
