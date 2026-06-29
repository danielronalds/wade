# Web Terminal

A browser terminal backed by a real local shell via Go, a PTY, WebSockets and
xterm.js. npm is used to fetch xterm.js, then `go generate` stages those files
with the static frontend so they can be embedded into the Go binary.

## Setup

```sh
mise install
mise run dev
```

Open <http://127.0.0.1:8765>.

To use a different address:

```sh
WEB_TERMINAL_ADDR=127.0.0.1:8090 mise run dev
```

## Development

```sh
mise run dev
```

This installs frontend dependencies, stages embedded assets, and starts Air.
Air reloads the Go server when Go, HTML, CSS or JavaScript files change.

## Build

```sh
mise run build
```

This writes the binary to `.tmp/web-terminal`.

## Test

```sh
mise run test
```

## Build process

Static source files live in `web/static`. xterm.js files are fetched by npm into
`web/node_modules`. `go generate ./...` copies the static source files and the
required xterm.js files into `web/dist`, and the Go binary embeds `web/dist` with
`go:embed`.

The generated `web/dist` directory and `web/node_modules` are ignored by git.

The server binds to localhost by default and only accepts same-origin WebSocket
upgrades.

## Nerd Fonts

The frontend uses a Nerd Font-first font stack. Install one locally, for
example:

```sh
brew install --cask font-jetbrains-mono-nerd-font
```

You can force a specific installed font with the `font` query parameter:

```text
http://127.0.0.1:8765/?font=JetBrainsMono%20Nerd%20Font
```
