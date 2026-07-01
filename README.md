# WADE

WADE is a Web-based Agentic Development Environment. It is a local-first
browser workspace for agentic coding sessions, backed by real project shells
through Go, PTYs, WebSockets and xterm.js.

WADE started as a simple browser terminal, but its purpose is now broader: make
it easy to open a project, reconnect to long-lived terminal sessions, and keep
useful project context close to the work.

WADE is intended to be keyboard driven. Mouse interactions are useful, but the
primary workflow should feel fast from the keyboard, especially project search,
tab switching, pane switching and terminal focus.

## What it provides

- A keyboard-first home screen with recent projects, a project command palette
  and a general command palette.
- Project discovery from local development directories.
- Persistent project terminal sessions for the lifetime of the server.
- Agent and Misc terminal panes for project work.
- A Server terminal tab for running the application under development.
- A project topbar with current branch, Linear ticket and pull request links.
- Embedded Nerd Font support for terminal icons.
- A local-only HTTP server with same-origin WebSocket checks.

## Setup

```sh
mise install
mise run dev
```

Open <http://editor-dev.localhost:8090>.

`mise run dev` sets `WADE_ADDR=editor-dev.localhost:8090` and starts Air. Air
sets `WADE_DEV=1` for the running binary. A directly built binary uses
<http://editor.localhost:8765> by default.

To use a different address:

```sh
WADE_ADDR=127.0.0.1:8090 mise run dev
```

## Projects

The root path shows the five most recently opened projects from browser
`localStorage`. Press `Ctrl + S` to open the command palette and search all
projects WADE can see.

Project pages use the project name as the path:

```text
http://editor-dev.localhost:8090/wade
```

The project name is resolved against the configured project directories. For
now, those directories are hard-coded from the local Projman shape.

Project terminal sessions are kept alive for the lifetime of the server.
Reopening a project reconnects to the existing shell session rather than
starting a new one.

## Keyboard shortcuts

Keyboard shortcuts are a core part of WADE's interaction model. New workflows
should prefer shortcuts and predictable focus behaviour over mouse-only flows.

- `Ctrl + P`: open the general command palette.
- `Ctrl + S`: open the project command palette.
- `Ctrl + B`, then `1`: switch to the Terminal tab.
- `Ctrl + B`, then `2`: switch to the Server tab.
- `Ctrl + B`, then `o`: switch to the next terminal pane in the active tab.

## Terminal behaviour

WADE uses xterm.js with the fit addon and WebGL renderer. The frontend sends
binary WebSocket messages as terminal input. Text WebSocket messages are JSON
control messages, currently used for resize events:

```json
{ "type": "resize", "cols": 120, "rows": 40 }
```

Escape has special handling. The frontend captures document `keydown` events in
the capture phase for the active terminal and sends raw `\x1b` to the PTY. This
keeps Escape reliable before xterm or browser focus handling can consume it.

## Development

```sh
mise run dev
```

This installs frontend dependencies, builds embedded assets, and starts Air.
Air reloads the Go server when Go, HTML, CSS, JavaScript, TypeScript, Vue,
JSON, TOML, Markdown or font files change.

## Build

```sh
mise run build
```

This writes the binary to `.tmp/wade`. Running that binary directly uses
<http://editor.localhost:8765> by default.

## Test

```sh
mise run test
```

For a smoke test, run the app on a temporary port, curl the static files, then
kill the process. Check that no stale `go run .` or `wade` processes remain on
test ports.

## Build process

Vue and TypeScript source files live in `web/src`. Static public assets live in
`web/static`. xterm.js, Vue and the frontend build tools are fetched by npm into
`web/node_modules`.

`go generate ./...` runs `npm run build` in `web`. That typechecks with
`vue-tsc`, bundles the frontend with esbuild into `web/dist`, and the Go binary
embeds `web/dist` with `go:embed`. There is no Vite dev server. Go serves the
built assets.

The generated `web/dist` directory and `web/node_modules` are ignored by git.

## Nerd Fonts

The frontend includes `JetBrainsMonoNerdFontMono-Regular.ttf` so Nerd Font
icons work without installing a local font. xterm uses this embedded font first,
then falls back to locally installed Nerd Fonts and normal monospace fonts.

Only the regular weight is bundled. Bold terminal text is browser-synthesised
from the regular face. If true bold rendering becomes important, we can also
bundle `JetBrainsMonoNerdFontMono-Bold.ttf`.

You can force a specific installed font with the `font` query parameter:

```text
http://editor-dev.localhost:8090/?font=JetBrainsMono%20Nerd%20Font
```
