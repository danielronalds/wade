# AGENTS.md

## Project

This is a quick Go POC for a local browser terminal backed by a real shell.

The backend is a Go HTTP server bound to localhost. It creates a PTY with
`github.com/creack/pty` and streams bytes over WebSockets with
`github.com/gorilla/websocket`.

The frontend lives in `static/index.html` and uses vendored `xterm.js` files
from `static/vendor/xterm`. Avoid CDN JavaScript for the local shell page.

The default address is `127.0.0.1:8765`. Override it with
`WEB_TERMINAL_ADDR`.

## Running

```sh
go run .
```

Then open <http://127.0.0.1:8765>.

## Terminal behaviour

Use `xterm.js`, not `ghostty-web`, for now. `ghostty-web` was tried, but did
not work quite right for this POC.

The terminal is full screen, with a small floating connection status in the
bottom right. The status widget toggles between open text and closed dot-only
modes when clicked.

Resize handling uses the xterm fit addon. The client sends JSON control
messages like this:

```json
{ "type": "resize", "cols": 120, "rows": 40 }
```

The server treats text WebSocket messages as control messages and binary
messages as terminal input.

Escape needs special handling. The frontend captures document `keydown` events
in the capture phase and sends raw `\x1b` to the PTY. This works around Escape
not reliably reaching the shell through normal xterm input handling.

## Nerd Fonts

Nerd Font support is configured through the xterm `fontFamily` option. The
frontend has a Nerd Font-first stack and supports a `font` query parameter:

```text
http://127.0.0.1:8765/?font=JetBrainsMono%20Nerd%20Font%20Mono
```

Install a local Nerd Font if icons do not render:

```sh
brew install --cask font-jetbrains-mono-nerd-font
```

xterm renders text on a canvas, so computed CSS on `.xterm` may show the page
font, not the font used for terminal glyphs. Browser canvas rendering does not
expose the exact fallback font chosen for each glyph.

Nerd Font icons can bleed or overlap because private-use and fallback glyphs can
be wider than xterm's fixed cells. xterm also avoids rescaling Nerd Font and
Powerline glyphs. Keep this in mind before changing font options. `lineHeight`
is currently set to `1`.

## Vendored frontend dependencies

The required xterm files are vendored under `static/vendor/xterm`:

- `xterm.js`
- `xterm.js.map`
- `xterm.css`
- `addon-fit.js`
- `addon-fit.js.map`
- licence files

To update them, use `npm pack @xterm/xterm @xterm/addon-fit`, extract the
packages, and copy the built files from `lib` and `css`.

## Validation

Run:

```sh
go test ./...
```

For a smoke test, run the app on a temporary port, curl the static files, then
kill the process. Check that no stale `go run .` or `local-web-terminal`
processes remain on test ports.
