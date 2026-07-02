package handlers

import (
	"net/http"
	"strings"

	"wade/project"
)

type sessionCloser interface {
	CloseSessionsForDirectory(directory string) int
}

type Sessions struct {
	projects  project.Store
	terminals sessionCloser
}

func NewSessions(projects project.Store, terminals sessionCloser) Sessions {
	return Sessions{projects: projects, terminals: terminals}
}

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

	if h.terminals != nil {
		h.terminals.CloseSessionsForDirectory(projectPath)
	}

	w.WriteHeader(http.StatusNoContent)
}
