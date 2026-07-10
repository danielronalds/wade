package controllers

// TODO: Review properly

import (
	"errors"
	"net/http"

	"wade/internal/services/sessions"
)

type Sessions struct {
	sessions sessions.Service
}

type sessionsResponse struct {
	Sessions []string `json:"sessions"`
} // @name handlers.sessionsResponse

func NewSessions(sessions sessions.Service) Sessions {
	return Sessions{sessions: sessions}
}

// @Summary List active project sessions
// @ID listActiveProjectSessions
// @Tags Sessions
// @Produce json
// @Success 200 {object} sessionsResponse
// @Router /api/sessions [get]
func (h Sessions) ListSessions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, sessionsResponse{Sessions: h.sessions.List()})
}

// @Summary Close project session
// @ID closeProjectSession
// @Tags Sessions
// @Produce json
// @Param sessionName path string true "Session project name"
// @Success 204 "No Content"
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /api/session/{sessionName} [delete]
func (h Sessions) CloseSession(w http.ResponseWriter, r *http.Request) {
	err := h.sessions.Close(r.PathValue("sessionName"))
	if errors.Is(err, sessions.ErrSessionRequired) {
		writeJSONError(w, http.StatusBadRequest, "session is required")
		return
	}
	if errors.Is(err, sessions.ErrSessionNotFound) {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "unable to close session")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
