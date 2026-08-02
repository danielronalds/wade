package controllers

import (
	"errors"
	"log"
	"net/http"
	"net/url"

	"wade/internal/services/terminals"
	"wade/internal/services/workspaces"

	"github.com/gorilla/websocket"
)

type Terminals struct {
	terminals *terminals.Service
	upgrader  websocket.Upgrader
}

type TerminalList struct {
	Items []*terminals.Terminal `json:"items"`
} // @name TerminalList

type TerminalInputRequest struct {
	Text string              `json:"text"`
	Mode terminals.InputMode `json:"mode"`
} // @name TerminalInputRequest

func NewTerminals(terminalService *terminals.Service, checkOrigin func(r *http.Request) bool) Terminals {
	return Terminals{
		terminals: terminalService,
		upgrader: websocket.Upgrader{
			CheckOrigin: checkOrigin,
		},
	}
}

// @Summary List workspace terminals
// @ID listWorkspaceTerminals
// @Tags Terminals
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} TerminalList
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/terminals [get]
func (h Terminals) List(w http.ResponseWriter, r *http.Request) {
	workspaceTerminals, err := h.terminals.List(r.PathValue("workspaceId"))
	if err != nil {
		writeServiceError(w, err, "Unable to list workspace terminals.")
		return
	}

	writeJSON(w, http.StatusOK, TerminalList{Items: workspaceTerminals})
}

// @Summary Close all workspace terminals
// @ID deleteWorkspaceTerminals
// @Tags Terminals
// @Param workspaceId path string true "Workspace ID"
// @Success 204 "No Content"
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/terminals [delete]
func (h Terminals) DeleteAll(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspaceId")
	if _, err := h.terminals.List(workspaceID); err != nil {
		writeServiceError(w, err, "Unable to close workspace terminals.")
		return
	}

	h.terminals.DeleteAll(workspaceID)
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Start or reconnect to a terminal
// @ID putWorkspaceTerminal
// @Tags Terminals
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param terminalId path string true "Terminal ID"
// @Success 200 {object} terminals.Terminal
// @Success 201 {object} terminals.Terminal
// @Header 201 {string} Location "Created terminal URL"
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/terminals/{terminalId} [put]
func (h Terminals) Put(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspaceId")
	terminal, created, err := h.terminals.Put(workspaceID, r.PathValue("terminalId"))
	if err != nil {
		writeServiceError(w, err, "Unable to start the terminal.")
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
		w.Header().Set("Location", terminalURL(workspaceID, terminal.ID))
	}
	writeJSON(w, status, terminal)
}

// @Summary Get a workspace terminal
// @ID getWorkspaceTerminal
// @Tags Terminals
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param terminalId path string true "Terminal ID"
// @Success 200 {object} terminals.Terminal
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/terminals/{terminalId} [get]
func (h Terminals) Get(w http.ResponseWriter, r *http.Request) {
	terminal, err := h.terminals.Get(r.PathValue("workspaceId"), r.PathValue("terminalId"))
	if err != nil {
		writeServiceError(w, err, "Unable to load the terminal.")
		return
	}

	writeJSON(w, http.StatusOK, terminal)
}

// @Summary Close a workspace terminal
// @ID deleteWorkspaceTerminal
// @Tags Terminals
// @Param workspaceId path string true "Workspace ID"
// @Param terminalId path string true "Terminal ID"
// @Success 204 "No Content"
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/terminals/{terminalId} [delete]
func (h Terminals) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.terminals.Delete(r.PathValue("workspaceId"), r.PathValue("terminalId")); err != nil {
		writeServiceError(w, err, "Unable to close the terminal.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary Send input to a workspace terminal
// @ID sendWorkspaceTerminalInput
// @Tags Terminals
// @Accept json
// @Param workspaceId path string true "Workspace ID"
// @Param terminalId path string true "Terminal ID"
// @Param request body TerminalInputRequest true "Terminal input"
// @Success 204 "No Content"
// @Failure 400 {object} Problem
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input [post]
func (h Terminals) Input(w http.ResponseWriter, r *http.Request) {
	var request TerminalInputRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeProblem(w, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain valid terminal input.")
		return
	}

	if err := h.terminals.Input(r.PathValue("workspaceId"), r.PathValue("terminalId"), request.Text, request.Mode); err != nil {
		writeServiceError(w, err, "Unable to send terminal input.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary Connect to a workspace terminal
// @Description Upgrades the connection to a WebSocket for terminal input, output, and control messages.
// @ID connectWorkspaceTerminal
// @Tags Terminals
// @Param workspaceId path string true "Workspace ID"
// @Param terminalId path string true "Terminal ID"
// @Success 101 "Switching Protocols"
// @Failure 404 {string} string "Terminal not found"
// @Failure 422 {string} string "Invalid terminal ID"
// @Failure 500 {string} string "Connection failed"
// @Router /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/socket [get]
func (h Terminals) Connect(w http.ResponseWriter, r *http.Request) {
	terminal, err := h.terminals.Get(r.PathValue("workspaceId"), r.PathValue("terminalId"))
	if err != nil {
		writeSocketError(w, err)
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

func writeSocketError(w http.ResponseWriter, err error) {
	var workspaceNotFound workspaces.WorkspaceNotFoundError
	var terminalNotFound terminals.TerminalNotFoundError
	var invalidTerminalID terminals.InvalidTerminalIDError
	var agentNotConfigured terminals.AgentNotConfiguredError

	switch {
	case errors.As(err, &workspaceNotFound), errors.As(err, &terminalNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.As(err, &invalidTerminalID), errors.As(err, &agentNotConfigured):
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	default:
		log.Printf("terminal socket lookup failed: %v", err)
		http.Error(w, "unable to connect to terminal", http.StatusInternalServerError)
	}
}

func terminalURL(workspaceID string, terminalID string) string {
	return "/api/v1/workspaces/" + url.PathEscape(workspaceID) + "/terminals/" + url.PathEscape(terminalID)
}
