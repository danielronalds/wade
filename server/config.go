package server

import (
	"log"
	"net/http"

	"wade/config"
)

// handleConfigReload reloads runtime-safe settings on demand.
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
	return nil
}
