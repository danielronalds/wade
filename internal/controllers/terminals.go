package controllers

// TODO: Review properly

import (
	"errors"
	"log"
	"net/http"

	"wade/internal/services/terminals"
	"wade/internal/services/workspaces"

	"github.com/gorilla/websocket"
)

type Terminals struct {
	terminals *terminals.Service
	upgrader  websocket.Upgrader
}

func NewTerminals(terminalService *terminals.Service, checkOrigin func(r *http.Request) bool) Terminals {
	return Terminals{
		terminals: terminalService,
		upgrader: websocket.Upgrader{
			CheckOrigin: checkOrigin,
		},
	}
}

// @Summary Reload terminal session
// @ID reloadTerminalSession
// @Tags Terminals
// @Produce json
// @Param project query string true "Project name"
// @Param terminal query string false "Terminal name"
// @Param agent query string false "Agent name"
// @Success 204 "No Content"
// @Failure 404 {object} errorResponse
// @Router /api/terminal/reload [post]
func (h Terminals) Reload(w http.ResponseWriter, r *http.Request) {
	terminalID, err := h.terminals.LegacyTerminalID(r.URL.Query().Get("terminal"), r.URL.Query().Get("agent"))
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, "terminal not found")
		return
	}

	err = h.terminals.Delete(r.URL.Query().Get("project"), terminalID)
	var notFoundError terminals.TerminalNotFoundError
	if err != nil && !errors.As(err, &notFoundError) {
		if isWorkspaceLookupError(err) {
			WriteJSONError(w, http.StatusNotFound, "project not found")
			return
		}

		WriteJSONError(w, http.StatusInternalServerError, "unable to reload terminal")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary Connect to terminal session
// @Description Upgrades the connection to a WebSocket for terminal input, output, and control messages.
// @ID connectTerminalSession
// @Tags Terminals
// @Param project query string true "Project name"
// @Param terminal query string false "Terminal name"
// @Param agent query string false "Agent name"
// @Success 101 "Switching Protocols"
// @Failure 404 {string} string "Project not found"
// @Failure 500 {string} string "Failed to start terminal"
// @Router /ws [get]
func (h Terminals) Connect(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("project")
	terminalID, err := h.terminals.LegacyTerminalID(r.URL.Query().Get("terminal"), r.URL.Query().Get("agent"))
	if err != nil {
		http.Error(w, "terminal not found", http.StatusNotFound)
		return
	}

	terminal, _, err := h.terminals.Put(workspaceID, terminalID)
	if err != nil {
		if isWorkspaceLookupError(err) {
			http.Error(w, "project not found", http.StatusNotFound)
			return
		}

		log.Printf("pty start failed: %v", err)
		http.Error(w, "failed to start terminal", http.StatusInternalServerError)
		return
	}

	connection, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer connection.Close()

	client := terminal.Attach()
	defer client.Close()

	done := make(chan struct{}, 2)

	go func() {
		streamTerminalToWebSocket(connection, client)
		done <- struct{}{}
	}()

	go func() {
		streamWebSocketToTerminal(connection, terminal)
		done <- struct{}{}
	}()

	<-done
}

const (
	terminalReplayStartMessage = `{"type":"replayStart"}`
	terminalReplayEndMessage   = `{"type":"replayEnd"}`
)

func streamTerminalToWebSocket(connection *websocket.Conn, client *terminals.Client) {
	for output := range client.Output() {
		if err := writeTerminalOutput(connection, output); err != nil {
			return
		}
	}
}

func writeTerminalOutput(connection *websocket.Conn, output terminals.ClientOutput) error {
	switch output.Kind {
	case terminals.ClientOutputKindReplayStart:
		return connection.WriteMessage(websocket.TextMessage, []byte(terminalReplayStartMessage))
	case terminals.ClientOutputKindReplayEnd:
		return connection.WriteMessage(websocket.TextMessage, []byte(terminalReplayEndMessage))
	default:
		return connection.WriteMessage(websocket.BinaryMessage, output.Data)
	}
}

func isWorkspaceLookupError(err error) bool {
	var notFoundError workspaces.WorkspaceNotFoundError
	var invalidIDError workspaces.InvalidWorkspaceIDError
	return errors.As(err, &notFoundError) || errors.As(err, &invalidIDError)
}

func streamWebSocketToTerminal(connection *websocket.Conn, terminal *terminals.Terminal) {
	for {
		messageType, data, err := connection.ReadMessage()
		if err != nil {
			return
		}

		switch messageType {
		case websocket.BinaryMessage:
			_, _ = terminal.Write(data)
		case websocket.TextMessage:
			terminal.ApplyControlMessage(data)
		}
	}
}
