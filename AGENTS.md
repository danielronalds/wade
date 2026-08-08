# AGENTS.md

## Project

WADE is a Web-based Agentic Development Environment. It is a local-first
browser workspace for agentic coding sessions, backed by real project shells
through Go, PTYs, WebSockets and xterm.js.

The backend is a Go HTTP server bound to localhost. It creates PTYs with
`github.com/creack/pty` and streams bytes over WebSockets with
`github.com/gorilla/websocket`.

The frontend lives in `web/src`, uses Vue 3 and TypeScript, and is bundled with
esbuild into `internal/web/.dist`. Avoid CDN JavaScript for the local shell page.

`mise run dev` serves WADE at `editor-dev.localhost:8090`. `WADE_DEV` takes
precedence over an inherited `WADE_ADDR`. A directly built binary uses
`editor.localhost:8765` by default and can be overridden with `WADE_ADDR`.

`wade server` manages one background daemon through
`${XDG_STATE_HOME:-~/.local/state}/wade/server.sock` and writes output to
`server.log` in the same directory. Use `wade status` to inspect the daemon and
`wade stop` to stop it gracefully. `wade server --foreground` remains unmanaged
for Air, development, and smoke tests.

## Running

```sh
mise run dev
```

Then open <http://editor-dev.localhost:8090>.

## Product intent

WADE has moved beyond a simple browser terminal. Treat it as a small local
development environment for project-based agent work.

This project should be keyboard driven. Mouse interactions can exist, but the
primary workflow should be fast from the keyboard, especially project search,
tab switching, pane switching and terminal focus.

Key product ideas:

- The home screen shows recent workspaces and lets the user search all workspaces
  from the keyboard.
- The general command palette should expose workspace actions without requiring
  mouse interaction.
- Workspace pages keep agent, miscellaneous and server shells close together.
- Terminals persist for the lifetime of the server.
- Workspace metadata should reduce context switching, such as branch, Linear
  ticket and pull request links.
- The app should stay local-first and avoid exposing shells to other origins.

## Terminal behaviour

Use `xterm.js`, not `ghostty-web`, for now. `ghostty-web` was tried, but did
not work quite right for this app.

The workspace screen has a topbar, a sidebar, and terminal tabs. The Terminal tab
contains Agent and Misc panes. The Server tab contains one terminal.

Resize handling uses the xterm fit addon. The client sends JSON control
messages like this:

```json
{ "type": "resize", "cols": 120, "rows": 40 }
```

The server treats text WebSocket messages as control messages and binary
messages as terminal input.

Escape needs special handling. The frontend captures document `keydown` events
in the capture phase for the active terminal and sends raw `\x1b` to the PTY.
This keeps Escape reliable before xterm or browser focus handling can consume
it.

## Keyboard shortcuts

Keyboard shortcuts are part of the product design, not just convenience helpers.
When adding UI, prefer a clear keyboard path and predictable focus behaviour.

- `Ctrl + K`: open the general command palette.
- `Ctrl + P`: open the workspace picker.
- `Ctrl + S`: open the active workspace picker.
- `Ctrl + B`, then `1`: switch to the Terminal tab.
- `Ctrl + B`, then `2`: switch to the Server tab.
- `Ctrl + B`, then `3`: switch to the Review tab.
- `Ctrl + B`, then `4`: open the workspace scratchpad terminal.
- `Ctrl + B`, then `o`: switch to the next terminal pane in the active tab.

## Nerd Fonts

Nerd Font support is configured through the xterm `fontFamily` option. The
frontend bundles JetBrains Mono Nerd Font and supports a `font` query parameter:

```text
http://editor-dev.localhost:8090/?font=JetBrainsMono%20Nerd%20Font%20Mono
```

xterm renders text on a canvas, so computed CSS on `.xterm` may show the page
font, not the font used for terminal glyphs. Browser canvas rendering does not
expose the exact fallback font chosen for each glyph.

Nerd Font icons can bleed or overlap because private-use and fallback glyphs can
be wider than xterm's fixed cells. xterm also avoids rescaling Nerd Font and
Powerline glyphs. Keep this in mind before changing font options. `lineHeight`
is currently set to `1`.

## Frontend dependencies

Frontend dependencies are installed with npm in `web/node_modules`.
`go generate ./...` runs `npm --prefix ../../web run build` from
`internal/web`, which regenerates the TypeScript API client, typechecks with
`vue-tsc` and bundles with esbuild. The generated `internal/web/.dist`
directory is embedded into the Go binary.

## OpenAPI and API client generation

The backend OpenAPI spec is generated with Swaggo annotations, stable `@ID`
operation IDs, and `--requiredByDefault`; run `mise run gen:openapi` after API
annotation changes and `mise run lint:openapi` to check drift. Frontend API
generation stays under `web/`: Orval uses `web/orval.config.ts` to generate the
fetch client at `web/src/api/generated/wade.ts`. Import generated API functions
directly, avoid thin wrapper modules.

## Backend structure

Use a direct MVC-style Controllers, Models, Infrastructure structure for the Go
backend:

```text
controllers -> models -> infrastructure
```

The migration to this structure is planned in
`docs/migration/PLAN.md`. Existing `internal/services` and
`internal/repositories` code is legacy migration code, not a pattern for new
work. Follow the active migration slice when changing those areas.

- `cmd/wade/main.go`: Top-level composition root for infrastructure and the
  Settings Model shared by CLI commands and server startup.
- `internal/app`: HTTP application composition root. Wires the remaining Models,
  HTTP controllers, and shutdown behaviour.
- `internal/daemon`: Managed background process runtime. Owns detached startup,
  readiness reporting, state paths, and Unix control-socket lifecycle. It does
  not construct the HTTP application or own CLI presentation.
- `internal/server`: HTTP server runtime. Owns the mux, route registration,
  origin checks, and server lifecycle.
- `internal/controllers`: One package with separate files per HTTP controller.
  Controllers handle HTTP and WebSocket transport concerns and thinly
  orchestrate workflows across aggregate Models.
- `internal/models/<aggregate>`: One package per aggregate Model. Models own
  domain behaviour, validation, state, concurrency, and high-level operations.
- `internal/infrastructure/<capability>`: Concrete filesystem, environment, Git,
  GitHub, Linear, and PTY integrations. Infrastructure owns mechanical external
  IO, command syntax, timeouts, and external-format parsing.
- `internal/web`: Embeds built frontend assets for serving.
- `internal/openapi`: Generated OpenAPI Go package and JSON spec.

Aggregate Model packages are:

```text
internal/models/
  remoterepositories/
  repositories/
  reviewsnapshots/
  settings/
  terminals/
  workspaces/
```

Worktrees and branches belong to the Repositories aggregate. They are not
independent Models.

Infrastructure packages are:

```text
internal/infrastructure/
  environment/
  filesystem/
  git/
  github/
  linear/
  pty/
```

Workspace discovery and settings-file access are cohesive capabilities inside
`infrastructure/filesystem`, not separate repository packages.

Dependency direction rules:

- The command-line server controller may depend on `internal/daemon`.
- `internal/daemon` must remain independent of controllers, Models, and
  infrastructure.
- Controllers may depend on Models, but must not call infrastructure directly.
- Models may depend on infrastructure, but must not depend on controllers.
- Infrastructure must not depend on Models or controllers.
- Avoid Model-to-Model dependencies. Share infrastructure capabilities where
  several Models need the same external lookup, such as workspace discovery.
- Controllers define one cohesive, aggregate-wide Model interface for the whole
  controller package in `model_interfaces.go`.
- Each Model defines the cohesive infrastructure interfaces it consumes.
  Concrete infrastructure implementations may satisfy interfaces for several
  Models.
- Infrastructure returns infrastructure-owned technical types. Models map them
  into aggregate-owned domain and API types.

Controller rules:

- Keep controllers as thin orchestration and transport layers.
- Controllers decode and validate HTTP syntax, coordinate calls across Models,
  compose cross-aggregate responses, map typed Model errors to HTTP problems,
  and write responses.
- Controllers must not reimplement domain validation or pass internal filesystem
  paths between Models.
- Decode directly into Model-owned command types when the transport and Model
  shapes match.
- Keep OpenAPI annotations on controllers.
- Store the Settings controller by pointer because it owns the mutex that
  serialises persistence and cross-Model runtime configuration application.

Model package rules:

- Expose one application-scoped, concurrency-safe `Model` per aggregate through
  normal dependency injection. Do not use package-global mutable state.
- Models own domain validation, typed domain errors, aggregate mutation locks,
  runtime state, and high-level external workflows.
- Models must not store request contexts.
- Return detached serialisable value snapshots, including defensive copies of
  nested mutable values. Do not expose pointers into internal state.
- Read filesystem, Git, GitHub, Linear, and settings state fresh unless profiling
  justifies a cache behind the Model API.
- Put the Model type, constructor, configuration, and lifecycle in `model.go`.
- Organise other files by cohesive concern, such as `worktrees.go`,
  `branches.go`, `types.go`, `errors.go`, and `validators.go`. Do not create
  empty convention files.
- In every file, place exported functions and methods above private functions
  and methods.
- Keep resource, command, and value types with their owning aggregate. Do not
  add a generic shared Model-types package.
- Document exported APIs with concise comments explaining their contracts or
  non-obvious behaviour.

Infrastructure rules:

- Keep infrastructure free of WADE API resource types and multi-step aggregate
  workflows. It may implement explicit data-source contracts such as configured
  workspace discovery, basename precedence, and canonical path resolution.
- Infrastructure may expose cohesive operations and parsed technical result
  types, but Models own aggregate invariants, idempotency, orchestration, and
  domain error selection.
- Keep low-level PTY process start, read, write, resize, and close behaviour in
  `internal/infrastructure/pty`.
- The Terminals Model owns terminal resources, process registry behaviour,
  buffering, clients, and live `TerminalSession` handles.

## Frontend structure

The frontend is organised by route views and features. Prefer the `@/*` import
alias for frontend imports instead of deep relative paths.

```text
web/src/
  api/                 Backend API clients and shared HTTP helpers
  assets/              Global CSS and frontend assets
  components/          App-wide reusable Vue components only
  composables/         App-wide reusable composables only
  features/            Cross-route feature areas
  router/              Vue Router setup
  stores/              App-wide Pinia stores shared across features
  types/               Shared TypeScript types
  utils/               Pure app-wide helpers
  views/               Route-level screens and their private UI
```

Follow these rules when adding or moving frontend code:

- Use the generated Orval client for backend API calls. Keep shared fetch/error
  behaviour in `web/src/api/httpClient.ts` and `web/src/api/http.ts` rather
  than adding hand-written endpoint clients.
- Put route-level screens in `web/src/views/<route>/<RouteView>.vue`.
- Put UI that is private to a route under that route's `components`, `tabs`, or
  `composables` folders. For example, project tab containers live under
  `web/src/views/project/tabs`.
- Put code that represents a cross-route workflow in `web/src/features`. Feature
  folders can contain `components`, `composables`, and local `types.ts` files.
- Keep truly generic shared composables in `web/src/composables`. For example,
  `useFuzzyItems` is generic, while project-specific fuzzy matching belongs in
  `features/projects`.
- Keep app-wide reusable components in `web/src/components`. Do not put
  route-only components there.
- Put app-wide Pinia stores in `web/src/stores` when the state is shared across
  multiple routes or feature areas, such as settings, projects, project details,
  active sessions, or command palette state.
- Keep feature-local or route-local stores inside the owning feature or view
  folder. For example, review-only state should stay under the review tab unless
  it becomes app-wide.
- Put pure app-wide helpers in `web/src/utils`, such as theme utilities.
- Do not introduce new components during a folder-structure refactor unless the
  task explicitly asks for component decomposition.

Current high-level shape:

```text
web/src/
  api/
    generated/
      wade.ts
    http.ts
    httpClient.ts
  assets/
    styles.css
  components/
    icons/
  composables/
    useFuzzyItems.ts
  features/
    command-palette/
    projects/
    sessions/
    terminal-session/
  router/
    index.ts
  stores/
    useSettingsStore.ts
  types/
  utils/
    theme.ts
  views/
    home/
    project/
      components/
      composables/
      tabs/
        review/
        server/
        terminal/
    settings/
```

## Coding standards

Document public exported Go APIs with concise doc comments that explain their
contracts or non-obvious behaviour.

When declaring several local variables together, group related variables and
separate unrelated groups with an empty line. This keeps setup and wiring blocks
easy to scan as they grow.

## Validation

Run:

```sh
mise run test
```

For a lifecycle smoke test, use an isolated `XDG_STATE_HOME` and temporary port,
then run `wade server`, `wade status`, `wade stop`, and `wade status`. Confirm the
final status exits with status `1`, foreground mode does not create a socket, and
no stale `go run .` or `wade` processes remain on test ports.
