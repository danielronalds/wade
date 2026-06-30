# Web Terminal

A browser terminal backed by a real local shell via Go, a PTY, WebSockets and
xterm.js. The frontend is Vue 3 and TypeScript, bundled with esbuild. The
`go generate` command builds the frontend into `web/dist` so it can be embedded
into the Go binary.

## Setup

```sh
mise install
mise run dev
```

Open <http://editor-dev.localhost:8765>.

Air sets `WEB_TERMINAL_DEV=1`, which makes the development server use
`editor-dev.localhost`. The normal binary uses `editor.localhost` by default.

To use a different address:

```sh
WEB_TERMINAL_ADDR=127.0.0.1:8090 mise run dev
```

## Projects

The root path shows the five most recently opened projects from browser
`localStorage`.

Open a project terminal with:

```text
http://editor.localhost:8765/web-terminal
```

The project name is resolved against the configured project directories. For
now, those directories are hard-coded from the local Projman config.

Project terminals are kept alive for the lifetime of the server. Reopening a
project reconnects to the existing shell session rather than starting a new one.

## Development

```sh
mise run dev
```

This installs frontend dependencies, builds embedded assets, and starts Air.
Air reloads the Go server when Go, HTML, CSS, JavaScript, TypeScript, Vue or
font files change.

## Build

```sh
mise run build
```

This writes the binary to `.tmp/web-terminal`. Running that binary directly uses
<http://editor.localhost:8765> by default.

## Test

```sh
mise run test
```

## Build process

Vue and TypeScript source files live in `web/src`. Static public assets live in
`web/static`. xterm.js, Vue and the frontend build tools are fetched by npm into
`web/node_modules`.

`go generate ./...` runs `npm run build` in `web`. That typechecks with
`vue-tsc`, bundles the frontend with esbuild into `web/dist`, and the Go binary
embeds `web/dist` with `go:embed`. There is no Vite dev server. Go serves the
built assets.

The generated `web/dist` directory and `web/node_modules` are ignored by git.

The server binds to localhost by default and only accepts same-origin WebSocket
upgrades.

## Nerd Fonts

The frontend includes `JetBrainsMonoNerdFontMono-Regular.ttf` so Nerd Font icons
work without installing a local font. xterm uses this embedded font first, then
falls back to locally installed Nerd Fonts and normal monospace fonts.

Only the regular weight is bundled. Bold terminal text is browser-synthesised
from the regular face. If true bold rendering becomes important, we can also
bundle `JetBrainsMonoNerdFontMono-Bold.ttf`.

You can force a specific installed font with the `font` query parameter:

```text
http://127.0.0.1:8765/?font=JetBrainsMono%20Nerd%20Font
```
