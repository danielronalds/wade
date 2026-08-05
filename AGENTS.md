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

Use a Controllers, Services, Repositories structure for the Go backend.

- `internal/app`: Composition root. Wires repositories, services, controllers,
  and shutdown behaviour.
- `internal/daemon`: Managed background process runtime. Owns detached startup,
  readiness reporting, state paths, and Unix control-socket lifecycle. It does
  not construct the HTTP application or own CLI presentation.
- `internal/server`: HTTP server runtime. Owns the mux, route registration,
  origin checks, and server lifecycle.
- `internal/controllers`: One package with separate files per controller.
  Controllers handle HTTP/WebSocket request and response concerns only.
- `internal/services/<service>`: One package per service. Services own
  application behaviour, orchestration, and validation.
- `internal/repositories`: One package with separate files per repository.
  Repositories own filesystem, process, command, cache, and external IO.
- `internal/web`: Embeds built frontend assets for serving.
- `internal/openapi`: Generated OpenAPI Go package and JSON spec.

Dependency direction rules:

- The command-line server controller may depend on `internal/daemon`.
- `internal/daemon` must remain independent of controllers, services, and
  repositories.
- Controllers may depend on services.
- Services may depend on repository interfaces.
- Repository interfaces should live in the service package that consumes them.
- Repositories should provide concrete implementations.
- Repositories must not depend on services or controllers.

Service package rules:

- Put the public service and its methods in `service.go`.
- Put pure validation helpers in `validators.go`.
- Put validation tests in `validators_test.go`.
- Put service request/result/domain types in `types.go` where needed.
- Put typed service errors in `errors.go` where needed.
- Keep OpenAPI annotations on controllers.
- Terminal PTY code belongs inside `internal/services/terminalsessions`, behind
  the public terminal session service API.

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
