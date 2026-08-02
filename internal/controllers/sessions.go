package controllers

// TODO: Review properly

import (
	"encoding/json"
	"net/http"
	"strings"
)

type sessionWorkspaceService interface {
	Path(workspaceID string) (string, error)
}

type sessionTerminalService interface {
	ActiveWorkspaceIDs() []string
	DeleteAll(workspaceID string) int
	InputToSelectedAgent(workspaceID string, text string) (int, error)
}

type Sessions struct {
	workspaces sessionWorkspaceService
	terminals  sessionTerminalService
}

type sessionsResponse struct {
	Sessions []string `json:"sessions"`
} // @name handlers.sessionsResponse

type agentInputRequest struct {
	Text string `json:"text"`
} // @name handlers.agentInputRequest

func NewSessions(workspaceService sessionWorkspaceService, terminalService sessionTerminalService) Sessions {
	return Sessions{workspaces: workspaceService, terminals: terminalService}
}

// @Summary List active project sessions
// @ID listActiveProjectSessions
// @Tags Sessions
// @Produce json
// @Success 200 {object} sessionsResponse
// @Router /api/sessions [get]
func (h Sessions) ListSessions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, sessionsResponse{Sessions: h.terminals.ActiveWorkspaceIDs()})
}

// @Summary Close project session
// @ID closeProjectSession
// @Tags Sessions
// @Produce json
// @Param sessionName path string true "Session project name"
// @Success 204 "No Content"
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/sessions/{sessionName} [delete]
func (h Sessions) CloseSession(w http.ResponseWriter, r *http.Request) {
	workspaceID := strings.TrimSpace(r.PathValue("sessionName"))
	if workspaceID == "" {
		writeJSONError(w, http.StatusBadRequest, "session is required")
		return
	}
	if _, err := h.workspaces.Path(workspaceID); err != nil {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	h.terminals.DeleteAll(workspaceID)
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Send text to the active agent terminal
// @ID sendTextToAgentTerminal
// @Tags Sessions
// @Accept json
// @Produce json
// @Param projectName path string true "Project name"
// @Param request body agentInputRequest true "Agent input"
// @Success 204 "No Content"
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/sessions/{projectName}/agent [post]
func (h Sessions) SendToAgent(w http.ResponseWriter, r *http.Request) {
	var request agentInputRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid agent input request")
		return
	}
	if request.Text == "" {
		writeJSONError(w, http.StatusBadRequest, "text is required")
		return
	}

	workspaceID := strings.TrimSpace(r.PathValue("projectName"))
	if workspaceID == "" {
		writeJSONError(w, http.StatusBadRequest, "session is required")
		return
	}
	if _, err := h.workspaces.Path(workspaceID); err != nil {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	activeAgentTerminals, err := h.terminals.InputToSelectedAgent(workspaceID, request.Text)
	switch {
	case err != nil:
		writeJSONError(w, http.StatusInternalServerError, "unable to send text to agent session")
	case activeAgentTerminals == 0:
		writeJSONError(w, http.StatusNotFound, "agent session not found")
	case activeAgentTerminals > 1:
		writeJSONError(w, http.StatusConflict, "multiple active agent sessions")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
