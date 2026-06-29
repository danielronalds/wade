package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type terminalControlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: allowSameOrigin,
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleTerminal)
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	address := serverAddress()
	log.Printf("open http://%s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

func handleTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	connection, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer connection.Close()

	command := exec.Command(shellPath())
	command.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")

	terminal, err := pty.StartWithSize(command, &pty.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		log.Printf("pty start failed: %v", err)
		return
	}
	defer func() {
		_ = terminal.Close()
		_ = command.Process.Kill()
		_ = command.Wait()
	}()

	done := make(chan struct{}, 2)

	go func() {
		streamTerminalToWebSocket(connection, terminal)
		done <- struct{}{}
	}()

	go func() {
		streamWebSocketToTerminal(connection, terminal)
		done <- struct{}{}
	}()

	<-done
}

func streamTerminalToWebSocket(connection *websocket.Conn, terminal *os.File) {
	buffer := make([]byte, 4096)

	for {
		bytesRead, err := terminal.Read(buffer)
		if err != nil {
			return
		}

		if err := connection.WriteMessage(websocket.BinaryMessage, buffer[:bytesRead]); err != nil {
			return
		}
	}
}

func streamWebSocketToTerminal(connection *websocket.Conn, terminal *os.File) {
	for {
		messageType, data, err := connection.ReadMessage()
		if err != nil {
			return
		}

		switch messageType {
		case websocket.BinaryMessage:
			_, _ = terminal.Write(data)
		case websocket.TextMessage:
			applyControlMessage(terminal, data)
		}
	}
}

func applyControlMessage(terminal *os.File, data []byte) {
	var message terminalControlMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return
	}

	if message.Type != "resize" || message.Cols == 0 || message.Rows == 0 {
		return
	}

	_ = pty.Setsize(terminal, &pty.Winsize{Cols: message.Cols, Rows: message.Rows})
}

func allowSameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https" {
		return false
	}

	return parsedOrigin.Host == r.Host
}

func serverAddress() string {
	if address := os.Getenv("WEB_TERMINAL_ADDR"); address != "" {
		return address
	}

	return "127.0.0.1:8765"
}

func shellPath() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	return "/bin/bash"
}
