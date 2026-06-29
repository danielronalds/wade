package server

import (
	"log"
	"net/http"

	"web-terminal/terminal"

	"github.com/gorilla/websocket"
)

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: allowSameOrigin,
}

func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
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

	session, err := terminal.Start(s.configuration.Shell, terminal.Size{Cols: 80, Rows: 24})
	if err != nil {
		log.Printf("pty start failed: %v", err)
		return
	}
	defer session.Close()

	done := make(chan struct{}, 2)

	go func() {
		streamTerminalToWebSocket(connection, session)
		done <- struct{}{}
	}()

	go func() {
		streamWebSocketToTerminal(connection, session)
		done <- struct{}{}
	}()

	<-done
}

func streamTerminalToWebSocket(connection *websocket.Conn, session *terminal.Session) {
	buffer := make([]byte, 4096)

	for {
		bytesRead, err := session.Read(buffer)
		if err != nil {
			return
		}

		if err := connection.WriteMessage(websocket.BinaryMessage, buffer[:bytesRead]); err != nil {
			return
		}
	}
}

func streamWebSocketToTerminal(connection *websocket.Conn, session *terminal.Session) {
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
