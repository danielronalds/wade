package handlers

import (
	"net/http"
	"strings"

	"wade/internal/project"
)

type sessionCloser interface {
	CloseSessionsForDirectory(directory string) int
}

type activeSessionLister interface {
	ActiveDirectories() []string
}

type sessionManager interface {
	sessionCloser
	activeSessionLister
}

type Sessions struct {
	projects  project.Store
	terminals sessionManager
}

type sessionsResponse struct {
	Sessions []string `json:"sessions"`
}

func NewSessions(projects project.Store, terminals sessionManager) Sessions {
	return Sessions{projects: projects, terminals: terminals}
}

// @Summary List active project sessions
// @ID listActiveProjectSessions
// @Tags Sessions
// @Produce json
// @Success 200 {object} sessionsResponse
// @Router /api/sessions [get]
func (h Sessions) ListSessions(w http.ResponseWriter, r *http.Request) {
	activeProjects := h.projects.NamesForDirectories(h.terminals.ActiveDirectories())

	writeJSON(w, http.StatusOK, sessionsResponse{Sessions: activeProjects})
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
	sessionName := strings.TrimSpace(r.PathValue("sessionName"))
	if sessionName == "" {
		writeJSONError(w, http.StatusBadRequest, "session is required")
		return
	}

	projectPath, err := h.projects.Path(sessionName)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	h.terminals.CloseSessionsForDirectory(projectPath)

	w.WriteHeader(http.StatusNoContent)
}
