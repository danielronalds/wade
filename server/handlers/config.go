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
	ProjectDirectories                 []string `json:"projectDirectories"`
	AgentPaneCommand                   string   `json:"agentPaneCommand"`
	CopyIgnoredFilesOnWorktreeCreation bool     `json:"copyIgnoredFilesOnWorktreeCreation"`
	WorktreeCopyExcludes               []string `json:"worktreeCopyExcludes"`
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
		ProjectDirectories:                 settings.ProjectDirectories,
		AgentPaneCommand:                   settings.AgentPaneCommand,
		CopyIgnoredFilesOnWorktreeCreation: settings.CopyIgnoredFilesOnWorktreeCreation,
		WorktreeCopyExcludes:               settings.WorktreeCopyExcludes,
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

	agentPaneCommand := strings.TrimSpace(request.AgentPaneCommand)
	if err := config.ValidateAgentPaneCommand(agentPaneCommand); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	worktreeCopyExcludes := trimWorktreeCopyExcludes(request.WorktreeCopyExcludes)
	if err := config.ValidateWorktreeCopyExcludes(worktreeCopyExcludes); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	settings, err := config.LoadSettings()
	if err != nil {
		http.Error(w, "unable to load config", http.StatusInternalServerError)
		return
	}

	settings.ProjectDirectories = projectDirectories
	settings.AgentPaneCommand = agentPaneCommand
	settings.CopyIgnoredFilesOnWorktreeCreation = request.CopyIgnoredFilesOnWorktreeCreation
	settings.WorktreeCopyExcludes = worktreeCopyExcludes
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

func trimWorktreeCopyExcludes(excludes []string) []string {
	trimmed := make([]string, 0, len(excludes))
	for _, exclude := range excludes {
		exclude = strings.TrimSpace(exclude)
		if exclude == "" {
			continue
		}
		trimmed = append(trimmed, exclude)
	}

	return trimmed
}

func writeConfigPayload(w http.ResponseWriter, statusCode int, payload configPayload) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
