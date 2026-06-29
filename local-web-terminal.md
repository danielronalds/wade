# Local Web Terminal with a Real Shell Session

A guide to building a browser-based terminal that talks to a real local shell, styled to feel like Ghostty.

## Viability

A real interactive shell in a local webpage is straightforward. Ghostty itself **cannot** run in a browser — it's a native terminal emulator (Zig + AppKit/GTK) with no WASM/browser target. What you build instead is the standard web-terminal architecture, themed to look like Ghostty:

```
Browser (xterm.js) <-> WebSocket <-> Server (PTY) <-> real shell
```

- **Frontend:** [xterm.js](https://xtermjs.org/) renders the terminal — handles ANSI escape codes, input, rendering. Same library VS Code and Replit use.
- **Transport:** WebSocket streams bytes bidirectionally.
- **Backend:** Server spawns a real shell attached to a pseudoterminal (PTY) via `github.com/creack/pty`. This gives a genuine session — job control, TUI apps (vim, htop), tab completion.

For a **local-only** session bound to `127.0.0.1` with a single trusted user, you can drop almost all the security machinery (auth, TLS) that a networked deployment would require.

## Backend (Go)

```go
package main

import (
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func handleTerminal(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cmd := exec.Command(shell)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return
	}
	defer ptmx.Close()
	defer cmd.Process.Kill()

	// PTY output -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// WebSocket input -> PTY
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		ptmx.Write(data)
	}
}

func main() {
	http.HandleFunc("/ws", handleTerminal)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.ListenAndServe("127.0.0.1:8080", nil) // localhost only
}
```

## Frontend (static/index.html)

```html
<!DOCTYPE html>
<html>
<head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@xterm/xterm/css/xterm.css" />
  <script src="https://cdn.jsdelivr.net/npm/@xterm/xterm/lib/xterm.js"></script>
</head>
<body style="margin:0">
  <div id="term"></div>
  <script>
    const term = new Terminal({ cursorBlink: true });
    term.open(document.getElementById('term'));
    const ws = new WebSocket('ws://127.0.0.1:8080/ws');
    ws.binaryType = 'arraybuffer';
    ws.onmessage = e => term.write(new Uint8Array(e.data));
    term.onData(d => ws.send(d));
  </script>
</body>
</html>
```

That's a fully working interactive local shell in the browser — vim, htop, tab completion, and job control all work.

## Refinements

### Resize handling

Without it, `vim` and `htop` render at the wrong dimensions. Send the terminal size from the client and call `pty.Setsize` on the server. Use the `@xterm/addon-fit` addon to auto-size, send `{cols, rows}` as a JSON control message, and have the server distinguish control messages from keystrokes (e.g. a small JSON envelope or a separate message type).

### Ghostty look

Since real Ghostty can't run in a browser, replicate its feel by passing a theme into the `Terminal` constructor: Ghostty's default color palette, a font like `JetBrains Mono`, and matching cursor style and padding.

## Security note

Binding to `127.0.0.1` means only processes on your machine can reach the server, so you can skip auth and TLS. The one residual risk: any local browser tab on any site can connect to `ws://127.0.0.1:8080` unless you check the `Origin` header. Add a one-line origin check in `upgrader.CheckOrigin` even for local use.

## Reference implementations

Existing tools that implement this PTY-over-WebSocket pattern:

- **gotty** (Go) — cleanest reference for the exact pattern here
- **ttyd** (C) — lightweight
- **wetty** (Node) — SSH/login over the web
