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

`mise run dev` serves WADE at `editor-dev.localhost:8090`. A directly built
binary uses `editor.localhost:8765` by default. Override either with
`WADE_ADDR`.

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

- The home screen shows recent projects and lets the user search all projects
  from the keyboard.
- The general command palette should expose project actions without requiring
  mouse interaction.
- Project pages keep agent, miscellaneous and server shells close together.
- Terminal sessions persist for the lifetime of the server.
- Project metadata should reduce context switching, such as branch, Linear
  ticket and pull request links.
- The app should stay local-first and avoid exposing shells to other origins.

## Terminal behaviour

Use `xterm.js`, not `ghostty-web`, for now. `ghostty-web` was tried, but did
not work quite right for this app.

The project screen has a topbar, a sidebar, and terminal tabs. The Terminal tab
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
- `Ctrl + P`: open the project picker.
- `Ctrl + S`: open the project picker.
- `Ctrl + B`, then `1`: switch to the Terminal tab.
- `Ctrl + B`, then `2`: switch to the Server tab.
- `Ctrl + B`, then `3`: switch to the Review tab.
- `Ctrl + B`, then `4`: open the project scratchpad terminal.
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
`internal/web`, which typechecks with `vue-tsc` and bundles with esbuild. The
generated `internal/web/.dist` directory is embedded into the Go binary.

## Internal packages

- `internal/config`: Loads, validates and persists WADE runtime settings.
- `internal/project`: Discovers local projects and resolves project metadata.
- `internal/remote`: Lists and clones remote GitHub repositories.
- `internal/review`: Builds Git-based review data and file contents.
- `internal/server`: Wires HTTP routes, WebSockets and long-lived server state.
- `internal/server/handlers`: Implements HTTP API and page handlers.
- `internal/terminal`: Starts and controls PTY-backed terminal sessions.
- `internal/terminal/manager`: Manages persistent project terminal sessions and
  clients.
- `internal/web`: Embeds built frontend assets for serving.
- `internal/worktree`: Lists, creates and removes Git worktrees.

## Coding standards

When declaring several local variables together, group related variables and
separate unrelated groups with an empty line. This keeps setup and wiring blocks
easy to scan as they grow.

## Validation

Run:

```sh
mise run test
```

For a smoke test, run the app on a temporary port, curl the static files, then
kill the process. Check that no stale `go run .` or `wade` processes remain on
test ports.
