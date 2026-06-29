# Local Web Terminal POC

A browser terminal backed by a real local shell via Go, a PTY, WebSockets and
xterm.js. The terminal emulator is vendored under `static/vendor/xterm` so the
local shell page does not execute CDN JavaScript.

## Run

```sh
go run .
```

Open <http://127.0.0.1:8765>.

To use a different port:

```sh
WEB_TERMINAL_ADDR=127.0.0.1:8090 go run .
```

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

The server binds to localhost and only accepts same-origin WebSocket upgrades.
