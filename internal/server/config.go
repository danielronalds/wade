package server

import (
	"log"
	"net/http"

	"wade/internal/config"
)

// handleConfigReload reloads runtime-safe settings on demand.
// @Summary Reload runtime config
// @Tags Config
// @Produce plain
// @Success 204 "No Content"
// @Failure 400 {string} string "unable to reload config"
// @Router /api/config/reload [post]
func (s *Server) handleConfigReload(w http.ResponseWriter, r *http.Request) {
	if err := s.reloadConfig(); err != nil {
		log.Printf("config reload failed: %v", err)
		http.Error(w, "unable to reload config", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// reloadConfig applies settings that are safe to change at runtime.
func (s *Server) reloadConfig() error {
	configuration, err := config.Load()
	if err != nil {
		return err
	}

	s.projects.Reload(configuration.ProjectDirs)
	s.terminals.Configure(configuration.Shell, terminalAgents(configuration.Agents))
	return nil
}
