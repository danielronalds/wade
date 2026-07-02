package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"wade/config"
)

// ConfigHandler reads and writes editable WADE settings.
type ConfigHandler struct{}

type configPayload struct {
	ProjectDirectories []string `json:"projectDirectories"`
}

// NewConfig creates a handler for reading and writing WADE settings.
func NewConfig() ConfigHandler {
	return ConfigHandler{}
}

// ServeHTTP routes config requests by method.
func (h ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w)
	case http.MethodPost:
		h.saveConfig(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// getConfig returns the editable settings payload.
func (h ConfigHandler) getConfig(w http.ResponseWriter) {
	settings, err := config.LoadSettings()
	if err != nil {
		http.Error(w, "unable to load config", http.StatusInternalServerError)
		return
	}

	writeConfigPayload(w, http.StatusOK, configPayload{
		ProjectDirectories: settings.ProjectDirectories,
	})
}

// saveConfig validates and saves editable settings.
func (h ConfigHandler) saveConfig(w http.ResponseWriter, r *http.Request) {
	var request configPayload
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid config request", http.StatusBadRequest)
		return
	}

	projectDirectories := trimProjectDirectories(request.ProjectDirectories)
	if err := config.ValidateProjectDirectories(projectDirectories); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	settings, err := config.LoadSettings()
	if err != nil {
		http.Error(w, "unable to load config", http.StatusInternalServerError)
		return
	}

	settings.ProjectDirectories = projectDirectories
	if err := settings.Save(); err != nil {
		http.Error(w, "unable to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func trimProjectDirectories(projectDirectories []string) []string {
	trimmed := make([]string, 0, len(projectDirectories))
	for _, projectDirectory := range projectDirectories {
		trimmed = append(trimmed, strings.TrimSpace(projectDirectory))
	}

	return trimmed
}

func writeConfigPayload(w http.ResponseWriter, statusCode int, payload configPayload) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
