package controllers

// TODO: Review properly

import (
	"log"
	"net/http"

	projectservice "wade/internal/services/projects"
	"wade/internal/services/terminalsessions"

	"github.com/gorilla/websocket"
)

type Terminals struct {
	projects  projectservice.Service
	terminals *terminalsessions.Service
	upgrader  websocket.Upgrader
}

func NewTerminals(projects projectservice.Service, terminals *terminalsessions.Service, checkOrigin func(r *http.Request) bool) Terminals {
	return Terminals{
		projects:  projects,
		terminals: terminals,
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
	projectPath, err := h.projects.Path(r.URL.Query().Get("project"))
	if err != nil {
		WriteJSONError(w, http.StatusNotFound, "project not found")
		return
	}

	h.terminals.CloseTerminal(r.URL.Query().Get("terminal"), r.URL.Query().Get("agent"), projectPath)
	w.WriteHeader(http.StatusNoContent)
}

func (h Terminals) Connect(w http.ResponseWriter, r *http.Request) {
	projectPath, err := h.projects.Path(r.URL.Query().Get("project"))
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	terminalName := r.URL.Query().Get("terminal")
	projectSession, err := h.terminals.GetOrStart(terminalName, r.URL.Query().Get("agent"), projectPath)
	if err != nil {
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

	client := projectSession.Attach()
	defer client.Close()

	done := make(chan struct{}, 2)

	go func() {
		streamTerminalToWebSocket(connection, client)
		done <- struct{}{}
	}()

	go func() {
		streamWebSocketToTerminal(connection, projectSession)
		done <- struct{}{}
	}()

	<-done
}

const (
	terminalReplayStartMessage = `{"type":"replayStart"}`
	terminalReplayEndMessage   = `{"type":"replayEnd"}`
)

func streamTerminalToWebSocket(connection *websocket.Conn, client *terminalsessions.Client) {
	for output := range client.Output() {
		if err := writeTerminalOutput(connection, output); err != nil {
			return
		}
	}
}

func writeTerminalOutput(connection *websocket.Conn, output terminalsessions.ClientOutput) error {
	switch output.Kind {
	case terminalsessions.ClientOutputKindReplayStart:
		return connection.WriteMessage(websocket.TextMessage, []byte(terminalReplayStartMessage))
	case terminalsessions.ClientOutputKindReplayEnd:
		return connection.WriteMessage(websocket.TextMessage, []byte(terminalReplayEndMessage))
	default:
		return connection.WriteMessage(websocket.BinaryMessage, output.Data)
	}
}

func streamWebSocketToTerminal(connection *websocket.Conn, session *terminalsessions.ProjectSession) {
	for {
		messageType, data, err := connection.ReadMessage()
		if err != nil {
			return
		}

		switch messageType {
		case websocket.BinaryMessage:
			_, _ = session.Write(data)
		case websocket.TextMessage:
			session.ApplyControlMessage(data)
		}
	}
}
