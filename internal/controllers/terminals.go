package controllers

import (
	"errors"
	"log"
	"net/http"
	"net/url"

	"wade/internal/models/terminals"

	"github.com/gorilla/websocket"
)

// Terminals serves detached resources and coordinates live terminal sessions.
type Terminals struct {
	terminals TerminalsModel
	upgrader  websocket.Upgrader
}

// TerminalList is the collection response for workspace terminals.
type TerminalList struct {
	Items []terminals.Terminal `json:"items"`
} // @name TerminalList

// NewTerminals constructs the Terminals controller.
func NewTerminals(terminalModel TerminalsModel, checkOrigin func(r *http.Request) bool) Terminals {
	return Terminals{terminals: terminalModel, upgrader: websocket.Upgrader{CheckOrigin: checkOrigin}}
}

// List returns detached terminal resources for a workspace.
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
	workspaceTerminals, err := h.terminals.List(r.Context(), r.PathValue("workspaceId"))
	if err != nil {
		writeModelError(w, err, "Unable to list workspace terminals.")
		return
	}
	writeJSON(w, http.StatusOK, TerminalList{Items: workspaceTerminals})
}

// DeleteAll closes every terminal belonging to a workspace.
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
	if _, err := h.terminals.DeleteAll(r.Context(), r.PathValue("workspaceId")); err != nil {
		writeModelError(w, err, "Unable to close workspace terminals.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Put idempotently starts or returns a workspace terminal.
// @Summary Start or reconnect to a terminal
// @ID putWorkspaceTerminal
// @Tags Terminals
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param terminalId path string true "Terminal ID: misc; server; scratchpad; or agent:<lowercase-agent-name> for a configured agent"
// @Success 200 {object} terminals.Terminal
// @Success 201 {object} terminals.Terminal
// @Header 201 {string} Location "Created terminal URL"
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/terminals/{terminalId} [put]
func (h Terminals) Put(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.PathValue("workspaceId")
	terminal, created, err := h.terminals.Put(r.Context(), workspaceID, r.PathValue("terminalId"))
	if err != nil {
		writeModelError(w, err, "Unable to start the terminal.")
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
		w.Header().Set("Location", terminalURL(workspaceID, terminal.ID))
	}
	writeJSON(w, status, terminal)
}

// Get returns one detached workspace terminal resource.
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
	terminal, err := h.terminals.Get(r.Context(), r.PathValue("workspaceId"), r.PathValue("terminalId"))
	if err != nil {
		writeModelError(w, err, "Unable to load the terminal.")
		return
	}
	writeJSON(w, http.StatusOK, terminal)
}

// Delete closes one workspace terminal.
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
	if err := h.terminals.Delete(r.Context(), r.PathValue("workspaceId"), r.PathValue("terminalId")); err != nil {
		writeModelError(w, err, "Unable to close the terminal.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Input sends validated text to one workspace terminal.
// @Summary Send input to a workspace terminal
// @ID sendWorkspaceTerminalInput
// @Tags Terminals
// @Accept json
// @Param workspaceId path string true "Workspace ID"
// @Param terminalId path string true "Terminal ID"
// @Param request body terminals.Input true "Terminal input"
// @Success 204 "No Content"
// @Failure 400 {object} Problem
// @Failure 404 {object} Problem
// @Failure 422 {object} Problem
// @Failure 500 {object} Problem
// @Router /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/input [post]
func (h Terminals) Input(w http.ResponseWriter, r *http.Request) {
	var input terminals.Input
	if err := decodeJSONBody(r, &input); err != nil {
		writeProblem(w, http.StatusBadRequest, "malformed_json", "Malformed JSON", "The request body must contain valid terminal input.")
		return
	}
	input.WorkspaceID = r.PathValue("workspaceId")
	input.TerminalID = r.PathValue("terminalId")
	if err := h.terminals.Input(r.Context(), input); err != nil {
		writeModelError(w, err, "Unable to send terminal input.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Connect upgrades a request to a live terminal WebSocket session.
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
// @x-wade-cli-ignore true
// @Router /api/v1/workspaces/{workspaceId}/terminals/{terminalId}/socket [get]
func (h Terminals) Connect(w http.ResponseWriter, r *http.Request) {
	session, err := h.terminals.Connect(r.Context(), r.PathValue("workspaceId"), r.PathValue("terminalId"))
	if err != nil {
		writeSocketError(w, err)
		return
	}
	defer session.Close()

	connection, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer connection.Close()

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

const (
	terminalReplayStartMessage = `{"type":"replayStart"}`
	terminalReplayEndMessage   = `{"type":"replayEnd"}`
)

func streamTerminalToWebSocket(connection *websocket.Conn, session *terminals.TerminalSession) {
	for output := range session.Output() {
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

func streamWebSocketToTerminal(connection *websocket.Conn, session *terminals.TerminalSession) {
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

func writeSocketError(w http.ResponseWriter, err error) {
	var workspaceNotFound terminals.WorkspaceNotFoundError
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
