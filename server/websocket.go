package server

import (
	"log"
	"net/http"

	terminalmanager "wade/terminal/manager"

	"github.com/gorilla/websocket"
)

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: allowSameOrigin,
}

func terminalSessionKey(projectPath string, terminalName string) string {
	if terminalName == "" {
		terminalName = "terminal"
	}

	return projectPath + "\x00" + terminalName
}

func (s *Server) handleTerminalReload(w http.ResponseWriter, r *http.Request) {
	projectPath, err := s.projects.Path(r.URL.Query().Get("project"))
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	s.terminals.CloseSession(terminalSessionKey(projectPath, r.URL.Query().Get("terminal")))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	projectPath, err := s.projects.Path(r.URL.Query().Get("project"))
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	terminalName := r.URL.Query().Get("terminal")
	projectSession, err := s.terminals.GetOrStart(terminalSessionKey(projectPath, terminalName), terminalName, projectPath)
	if err != nil {
		log.Printf("pty start failed: %v", err)
		http.Error(w, "failed to start terminal", http.StatusInternalServerError)
		return
	}

	connection, err := websocketUpgrader.Upgrade(w, r, nil)
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

func streamTerminalToWebSocket(connection *websocket.Conn, client *terminalmanager.Client) {
	for data := range client.Output() {
		if err := connection.WriteMessage(websocket.BinaryMessage, data); err != nil {
			return
		}
	}
}

func streamWebSocketToTerminal(connection *websocket.Conn, session *terminalmanager.ProjectSession) {
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
